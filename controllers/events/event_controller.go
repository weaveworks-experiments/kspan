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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/tracing"
)

var (
	noTrace = trace.EmptySpanContext()
)

// EventWatcher listens to Events
type EventWatcher struct {
	client.Client
	Log      logr.Logger
	Exporter tracesdk.SpanExporter

	recent map[objectReference]recentInfo

	resources map[source]*resource.Resource
}

// This is how we fetch objects - it's a subset of corev1.ObjectReference
type objectReference struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
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

func eventSource(event *corev1.Event) source {
	if event.Source.Component != "" {
		return source{
			name:     event.Source.Component,
			instance: event.Source.Host,
		}
	}
	return source{
		name:     event.ReportingController,
		instance: event.ReportingInstance,
	}
}

func (r *EventWatcher) eventToSpan(event *corev1.Event, remoteContext trace.SpanContext) *tracesdk.SpanData {
	// resource says which component the span is seen as coming from
	source := eventSource(event)
	res, found := r.resources[source]
	if !found {
		// Make a new resource and cache for later.  TODO: cache eviction
		res = resource.New(semconv.ServiceNameKey.String(source.name), semconv.ServiceInstanceIDKey.String(source.instance))
		r.resources[source] = res
	}

	attrs := []label.KeyValue{
		label.String("type", event.Type),
		label.String("kind", event.InvolvedObject.Kind),
		label.String("namespace", event.InvolvedObject.Namespace),
		label.String("name", event.InvolvedObject.Name),
		label.String("message", event.Message),
		label.String("eventID", event.Namespace+"/"+event.Name),
	}

	// Some events have just an EventTime; if LastTimestamp is present we prefer that.
	spanTime := event.EventTime.Time
	if !event.LastTimestamp.Time.IsZero() {
		spanTime = event.LastTimestamp.Time
	}

	return &tracesdk.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: remoteContext.TraceID,
			SpanID:  eventToSpanID(event),
		},
		ParentSpanID:    remoteContext.SpanID,
		SpanKind:        trace.SpanKindInternal,
		Name:            fmt.Sprintf("%s.%s", event.InvolvedObject.Kind, event.Reason),
		StartTime:       spanTime,
		EndTime:         spanTime,
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

func getObject(ctx context.Context, c client.Client, apiVersion, kind, namespace, name string) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	if apiVersion == "" { // this happens with Node references
		apiVersion = "v1" // TODO: find a more general solution
	}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	key := client.ObjectKey{Namespace: namespace, Name: name}
	err := c.Get(ctx, key, obj)
	return obj, err
}

func refFromObjRef(oRef corev1.ObjectReference) objectReference {
	apiVersion := oRef.APIVersion
	if apiVersion == "" { // this happens with Node references
		apiVersion = "v1" // TODO: find a more general solution
	}
	return objectReference{
		APIVersion: apiVersion,
		Kind:       oRef.Kind,
		Namespace:  oRef.Namespace,
		Name:       oRef.Name,
	}
}

func refFromOwner(oRef v1.OwnerReference, namespace string) objectReference {
	return objectReference{
		APIVersion: oRef.APIVersion,
		Kind:       oRef.Kind,
		Namespace:  namespace,
		Name:       oRef.Name,
	}
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
	return noTrace, nil
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	if r.recent == nil {
		r.recent = make(map[objectReference]recentInfo)
	}
	if r.resources == nil {
		r.resources = make(map[source]*resource.Resource)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
