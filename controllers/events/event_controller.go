package events

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/api/trace"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	noTrace = trace.EmptySpanContext()
)

// EventWatcher listens to Events
type EventWatcher struct {
	client.Client
	Log      logr.Logger
	Exporter tracesdk.SpanExporter

	recent recentInfoStore

	resources map[source]*resource.Resource
}

type parentChild struct {
	parent objectReference
	child  objectReference
}

func (p parentChild) String() string {
	return fmt.Sprintf("parent: %s, child: %s", p.parent, p.child) // TODO improve this
}

// Info about the source of an event, e.g. kubelet
type source struct {
	name     string
	instance string
}

var (
	totalEventsNum = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kspan",
			Subsystem: "events",
			Name:      "total",
			Help:      "The total number of events.",
		},
		[]string{"type", "involved_object", "reason"})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(totalEventsNum)
}

// Reconcile gets called every time an Event changes
func (r *EventWatcher) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("event", req.NamespacedName)

	// Fetch the Event object
	var event corev1.Event
	if err := r.Get(ctx, req.NamespacedName, &event); err != nil {
		if apierrors.IsNotFound(err) {
			// we get this on deleted events, which happen all the time; just ignore it.
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Event")
		return ctrl.Result{}, err
	}

	// Bump Prometheus metrics
	totalEventsNum.WithLabelValues(event.Type, event.InvolvedObject.Kind, event.Reason).Inc()

	err := r.handleEvent(ctx, log, &event)
	if err != nil {
		log.Error(err, "unable to handle event")
	}

	return ctrl.Result{}, nil
}

func isNotFound(err error) bool {
	var unwrapped apierrors.APIStatus
	if errors.As(err, &unwrapped) {
		return apierrors.IsNotFound(unwrapped.(error))
	}
	return apierrors.IsNotFound(err)
}

func (r *EventWatcher) handleEvent(ctx context.Context, log logr.Logger, event *corev1.Event) error {
	ref := parentChildFromEvent(event)
	// Find which object the Event relates to
	objRef := ref.child
	if objRef.blank() {
		objRef = ref.parent
	}
	if objRef.blank() {
		return fmt.Errorf("No involved object")
	}
	involved, err := getObject(ctx, r.Client, event.InvolvedObject.APIVersion, objRef.Kind, objRef.Namespace, objRef.Name)
	if err != nil {
		if isNotFound(err) {
			return nil // again, happens so often it's not worth logging
		}
		return err
	}

	var remoteContext trace.SpanContext

	// Special case: A flux sync event can be the top level trigger
	if event.Source.Component == "flux" && event.Reason == "Sync" {
		m, _ := meta.Accessor(involved)
		remoteContext.TraceID = objectToTraceID(m)
		ref = parentChild{child: refFromObjectMeta(involved, m)}
	} else {
	// See if we can map this object to a trace
		remoteContext, err = r.spanContextFromObject(ctx, involved)
	if err != nil {
		return err
	}
	if !remoteContext.HasTraceID() {
		return nil // no parent context found; don't create a span for this event
	}
	}

	// Send out a span from the event details
	span := r.eventToSpan(event, remoteContext)
	r.emitSpan(ctx, span)
	r.recent.store(ref, span.SpanContext)

	return nil
}

func (r *EventWatcher) emitSpan(ctx context.Context, span *tracesdk.SpanData) {
	// TODO: consider building up all the spans then sending in one go
	r.Exporter.ExportSpans(ctx, []*tracesdk.SpanData{span})
}

func (r *EventWatcher) getResource(s source) *resource.Resource {
	res, found := r.resources[s]
	if !found {
		// Make a new resource and cache for later.  TODO: cache eviction
		res = resource.New(semconv.ServiceNameKey.String(s.name), semconv.ServiceInstanceIDKey.String(s.instance))
		r.resources[s] = res
	}
	return res
}

func (r *EventWatcher) spanContextFromObject(ctx context.Context, obj runtime.Object) (trace.SpanContext, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return noTrace, err
	}
	remoteContext := tracing.SpanContextFromAnnotations(ctx, m.GetAnnotations())
	if remoteContext.HasTraceID() {
		return remoteContext, nil
	}
	// This object doesn't have a span context; see if one of its owner chain does
	for _, ownerRef := range m.GetOwnerReferences() {
		ref := parentChild{
			parent: refFromOwner(ownerRef, m.GetNamespace()),
			child:  refFromObjectMeta(obj, m),
		}
		if spanContext, found := r.recent.lookupSpanContext(ref); found {
			return spanContext, nil
		}
		// Try the parent on its own
		if spanContext, found := r.recent.lookupSpanContext(parentChild{parent: ref.parent}); found {
			return spanContext, nil
		}
		owner, err := getObject(ctx, r.Client, ownerRef.APIVersion, ownerRef.Kind, m.GetNamespace(), ownerRef.Name)
		if err != nil {
			return noTrace, err
		}
		remoteContext, err := r.spanContextFromObject(ctx, owner)
		if err != nil {
			return noTrace, err
		}
		if remoteContext.HasTraceID() {
			return remoteContext, nil
		}
	}
	if len(m.GetOwnerReferences()) == 0 {
		ref := parentChild{ // parent is blank
			child: refFromObjectMeta(obj, m),
		}
		if spanContext, found := r.recent.lookupSpanContext(ref); found {
			return spanContext, nil
		}
		spanData, err := r.createTraceFromTopLevelObject(ctx, obj)
		if err != nil {
			return noTrace, err
		}
		r.emitSpan(ctx, spanData)
		r.recent.store(ref, spanData.SpanContext)
		return spanData.SpanContext, nil
	}
	return noTrace, nil
}

func (r *EventWatcher) initialize() {
	r.recent = newRecentInfoStore()
	r.resources = make(map[source]*resource.Resource)
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	r.initialize()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
