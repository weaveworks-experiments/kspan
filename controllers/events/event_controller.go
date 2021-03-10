package events

import (
	"context"
	"errors"
	"sync"
	"time"

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
	sync.Mutex
	Client    client.Client
	Log       logr.Logger
	Exporter  tracesdk.SpanExporter
	ticker    *time.Ticker
	startTime time.Time
	recent    recentInfoStore
	pending   []*corev1.Event
	resources map[source]*resource.Resource
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
	if err := r.Client.Get(ctx, req.NamespacedName, &event); err != nil {
		if isNotFound(err) {
			// we get this on deleted events, which happen all the time; just ignore it.
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Event")
		return ctrl.Result{}, err
	}

	if eventTime(&event).Before(r.startTime.Add(-r.recent.recentWindow)) {
		// too old - ignore
		return ctrl.Result{}, nil
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

// handleEvent is the meat of Reconcile, broken out for ease of testing.
func (r *EventWatcher) handleEvent(ctx context.Context, log logr.Logger, event *corev1.Event) error {
	log.Info("event", "kind", event.InvolvedObject.Kind, "reason", event.Reason, "source", event.Source.Component)

	emitted, err := r.emitSpanFromEvent(ctx, log, event)
	if err != nil {
		if isNotFound(err) { // can't find something - suppress reporting because this happens often
			err = nil
		}
		return err
	}
	if emitted { // a new span may allow us to map one that was saved from earlier
		err := r.checkPending(ctx)
		if err != nil && !isNotFound(err) {
			log.Error(err, "while checking pending events")
		}
	} else {
		// keep this event pending for a bit, see if something shows up that will let us map it.
		r.Lock()
		r.pending = append(r.pending, event)
		r.Unlock()
	}
	return nil
}

// If our rules tell us to map this event immediately to a context, do that.
func mapEventDirectlyToContext(ctx context.Context, client client.Client, event *corev1.Event, involved runtime.Object) (success bool, remoteContext trace.SpanContext, err error) {
	// Special case: A flux sync event can be the top level trigger
	if event.Source.Component == "flux" && event.Reason == "Sync" {
		m, _ := meta.Accessor(involved)
		remoteContext.TraceID = objectToTraceID(m)
		success = true
	}
	return
}

// attempt to map an Event to one or more Spans; return true if a Span was emitted
func (r *EventWatcher) emitSpanFromEvent(ctx context.Context, log logr.Logger, event *corev1.Event) (bool, error) {
	involved, ref, err := objectFromEvent(ctx, r.Client, event)
	if err != nil {
		return false, err
	}

	// If our rules tell us to map this event immediately to a context, do that.
	success, remoteContext, err := mapEventDirectlyToContext(ctx, r.Client, event, involved)
	if err != nil {
		return false, err
	}
	if !success {
		// If the involved object (or its owner) maps to recent activity, make a span parented off that.
		remoteContext, err = recentSpanContextFromObject(ctx, involved, &r.recent)
		if err != nil {
			return false, err
		}
		success = remoteContext.HasTraceID()
	}
	if !success {
		return false, nil
	}

	// Send out a span from the event details
	span := r.eventToSpan(event, remoteContext)
	r.emitSpan(ctx, span)
	r.recent.store(ref, remoteContext, span.SpanContext)

	return true, nil
}

func (r *EventWatcher) emitSpan(ctx context.Context, span *tracesdk.SpanData) {
	// TODO: consider building up all the spans then sending in one go
	r.Exporter.ExportSpans(ctx, []*tracesdk.SpanData{span})
}

func (r *EventWatcher) getResource(s source) *resource.Resource {
	r.Lock()
	defer r.Unlock()
	res, found := r.resources[s]
	if !found {
		// Make a new resource and cache for later.  TODO: cache eviction
		res = resource.New(semconv.ServiceNameKey.String(s.name), semconv.ServiceInstanceIDKey.String(s.instance))
		r.resources[s] = res
	}
	return res
}

func recentSpanContextFromObject(ctx context.Context, obj runtime.Object, recent *recentInfoStore) (trace.SpanContext, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return noTrace, err
	}
	// If no owners, this is a top-level object
	if len(m.GetOwnerReferences()) == 0 {
		objRef := refFromObject(m)
		if spanContext, _, found := recent.lookupSpanContext(actionReference{object: objRef}); found {
			return spanContext, nil
		}
	}
	// See if we have any recent event for an owner
	for _, ownerRef := range m.GetOwnerReferences() {
		ref := actionReference{
			actor:  refFromOwner(ownerRef, m.GetNamespace()),
			object: refFromObject(m),
		}
		if spanContext, _, found := recent.lookupSpanContext(ref); found {
			return spanContext, nil
		}
		// See if we can find a sibling event for the object on its own
		if _, parentContext, found := recent.lookupSpanContext(actionReference{object: ref.object}); found {
			return parentContext, nil
		}
		// Try the owner on its own; make that the parent if found
		if spanContext, _, found := recent.lookupSpanContext(actionReference{object: ref.actor}); found {
			return spanContext, nil
		}
	}
	return noTrace, err
}

func (r *EventWatcher) makeSpanContextFromObject(ctx context.Context, obj runtime.Object) (trace.SpanContext, error) {
	// See if we have any recent relevant event
	if sc, err := recentSpanContextFromObject(ctx, obj, &r.recent); err != nil || sc.HasTraceID() {
		return sc, err
	}

	m, err := meta.Accessor(obj)
	if err != nil {
		return noTrace, err
	}
	// If no recent event, recurse over owners
	for _, ownerRef := range m.GetOwnerReferences() {
		owner, err := getObject(ctx, r.Client, ownerRef.APIVersion, ownerRef.Kind, m.GetNamespace(), ownerRef.Name)
		if err != nil {
			return noTrace, err
		}
		remoteContext, err := r.makeSpanContextFromObject(ctx, owner)
		if err != nil {
			return noTrace, err
		}
		if remoteContext.HasTraceID() {
			return remoteContext, nil
		}
	}
	// If no owners and no recent data, create a span based off this top-level object
	if len(m.GetOwnerReferences()) == 0 {
		ref := actionReference{
			object: refFromObject(m),
		}
		spanData, err := r.createTraceFromTopLevelObject(ctx, obj)
		if err != nil {
			return noTrace, err
		}
		r.emitSpan(ctx, spanData)
		r.recent.store(ref, noTrace, spanData.SpanContext)
		return spanData.SpanContext, nil
	}
	return noTrace, nil
}

func (r *EventWatcher) runTicker() {
	// Need to check more often than the window, otherwise things will be too old.
	r.ticker = time.NewTicker(r.recent.recentWindow / 2)
	for range r.ticker.C {
		err := r.checkOlderPending(context.Background(), time.Now().Add(-r.recent.recentWindow))
		if err != nil {
			r.Log.Error(err, "from checkOlderPending")
		}
		r.recent.expire()
	}
}

func (r *EventWatcher) initialize() {
	r.Lock()
	r.startTime = time.Now()
	r.recent = newRecentInfoStore()
	r.resources = make(map[source]*resource.Resource)
	r.Unlock()
	go r.runTicker()
}

func (r *EventWatcher) stop() {
	if r.ticker != nil {
		r.ticker.Stop()
	}
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	r.initialize()
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
