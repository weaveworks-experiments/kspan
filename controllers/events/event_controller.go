package events

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/tracing"
)

// EventWatcher listens to Events
type EventWatcher struct {
	client.Client
	Log      logr.Logger
	Exporter tracesdk.SpanExporter
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

	var event corev1.Event
	if err := r.Get(ctx, req.NamespacedName, &event); err != nil {
		if apierrors.IsNotFound(err) {
			// we get this on deleted events, which happen all the time; just ignore it.
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Event")
		return ctrl.Result{}, err
	}

	totalEventsNum.WithLabelValues(event.Type, event.InvolvedObject.Kind, event.Reason).Inc()

	involved, err := getObjectFromReference(event.InvolvedObject, r.Client)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil // again, happens so often it's not worth logging
		}
		log.Error(err, "unable to fetch involved object")
		return ctrl.Result{}, nil
	}

	remoteContext, err := spanContextFromObject(ctx, involved, r.Client)
	if err != nil {
		log.Error(err, "unable to find span")
		return ctrl.Result{}, nil // nothing else we can do
	}
	if !remoteContext.HasTraceID() {
		return ctrl.Result{}, nil // no parent context found; don't create a span for this event
	}

	span := eventToSpan(remoteContext, &event)

	r.Exporter.ExportSpans(ctx, []*tracesdk.SpanData{span})

	return ctrl.Result{}, nil
}

func eventToSpan(remoteContext trace.SpanContext, event *corev1.Event) *tracesdk.SpanData {
	// resource says which component the span is seen as coming from
	res := resource.New(semconv.ServiceNameKey.String(event.Source.Component)) // TODO: cache these

	attrs := []label.KeyValue{
		label.String("type", event.Type),
		label.String("kind", event.InvolvedObject.Kind),
		label.String("namespace", event.InvolvedObject.Namespace),
		label.String("name", event.InvolvedObject.Name),
		label.String("message", event.Message),
	}

	return &tracesdk.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: remoteContext.TraceID,
			SpanID:  eventToSpanID(event),
		},
		//ParentSpanID:    remoteContext.SpanID,
		SpanKind:        trace.SpanKindInternal,
		Name:            fmt.Sprintf("%s.%s", event.InvolvedObject.Kind, event.Reason),
		StartTime:       event.LastTimestamp.Time,
		EndTime:         event.LastTimestamp.Time,
		Attributes:      attrs,
		HasRemoteParent: true,
		Resource:        res,
		//InstrumentationLibrary instrumentation.Library
	}
}

// generate a spanID from an event.  The first time this event is issued has a span ID that can be derived from the event UID
func eventToSpanID(event *corev1.Event) trace.SpanID {
	f := fnv.New64a()
	f.Write([]byte(event.UID))
	if event.Count > 0 {
		fmt.Fprint(f, event.Count)
	}
	var h trace.SpanID
	_ = f.Sum(h[:0])
	return h
}

func getObjectFromReference(ref corev1.ObjectReference, c client.Client) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	apiVersion := ref.APIVersion
	if apiVersion == "" { // this happens with Node references
		apiVersion = "v1" // TODO: find a more general solution
	}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(ref.Kind)
	key := client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}
	err := c.Get(context.Background(), key, obj)
	return obj, err
}

func spanContextFromObject(ctx context.Context, obj runtime.Object, c client.Client) (trace.SpanContext, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return trace.EmptySpanContext(), err
	}
	remoteContext := tracing.SpanContextFromAnnotations(ctx, m.GetAnnotations())
	if remoteContext.HasTraceID() {
		return remoteContext, nil
	}
	// This object doesn't have a span context; see if one of its owner chain does
	for _, ownerRef := range m.GetOwnerReferences() {
		owner := &unstructured.Unstructured{}
		owner.SetAPIVersion(ownerRef.APIVersion)
		owner.SetKind(ownerRef.Kind)
		err = c.Get(context.Background(), client.ObjectKey{
			Namespace: m.GetNamespace(),
			Name:      ownerRef.Name,
		}, owner)
		if err != nil {
			return trace.EmptySpanContext(), err
		}
		remoteContext, err := spanContextFromObject(ctx, owner, c)
		if err != nil {
			return trace.EmptySpanContext(), err
		}
		if remoteContext.HasTraceID() {
			return remoteContext, nil
		}
	}
	return trace.EmptySpanContext(), nil
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
