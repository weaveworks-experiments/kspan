package events

import (
	"context"

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

	recent         map[objectReference]recentInfo
	recentTopLevel map[objectReference]recentInfo

	resources map[source]*resource.Resource
}

// Info about what happened recently with an object
type recentInfo struct {
	trace.SpanContext
	event *corev1.Event // most recent event
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

	// Find which object the Event relates to
	involved, err := getObject(ctx, r.Client, event.InvolvedObject.APIVersion, event.InvolvedObject.Kind, event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil // again, happens so often it's not worth logging
		}
		log.Error(err, "unable to fetch involved object")
		return ctrl.Result{}, nil
	}

	// See if we can map this object to a trace
	remoteContext, err := r.spanContextFromObject(ctx, involved)
	if err != nil {
		log.Error(err, "unable to find span")
		return ctrl.Result{}, nil // nothing else we can do
	}
	if !remoteContext.HasTraceID() {
		return ctrl.Result{}, nil // no parent context found; don't create a span for this event
	}

	// Send out a span from the event details
	span := r.eventToSpan(&event, remoteContext)

	r.Exporter.ExportSpans(ctx, []*tracesdk.SpanData{span})

	ref := refFromObjRef(event.InvolvedObject)
	r.recent[ref] = recentInfo{
		SpanContext: span.SpanContext,
		event:       &event,
	}

	return ctrl.Result{}, nil
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
		ref := refFromOwner(ownerRef, m.GetNamespace())
		if recent, found := r.recent[ref]; found {
			return recent.SpanContext, nil
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
		ref := refFromObjectMeta(obj, m)
		if recent, found := r.recentTopLevel[ref]; found {
			return recent.SpanContext, nil
		}
		spanData, err := r.createTraceFromTopLevelObject(ctx, obj)
		if err != nil {
			return noTrace, err
		}
		r.recentTopLevel[refFromObjectMeta(obj, m)] = recentInfo{
			SpanContext: spanData.SpanContext,
		}
		return spanData.SpanContext, nil
	}
	return noTrace, nil
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	if r.recent == nil {
		r.recent = make(map[objectReference]recentInfo)
	}
	if r.recentTopLevel == nil {
		r.recentTopLevel = make(map[objectReference]recentInfo)
	}
	if r.resources == nil {
		r.resources = make(map[source]*resource.Resource)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
