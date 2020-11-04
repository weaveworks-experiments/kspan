package events

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// EventWatcher listens to Events
type EventWatcher struct {
	client.Client
	Log logr.Logger
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

	_ = involved

	operationName := fmt.Sprintf("%s.%s", event.InvolvedObject.Kind, event.Reason)
	ctx, span := global.Tracer("event-controller").Start(ctx, operationName)
	/*span, err := spanFromObject(ctx, operationName, involved, r.Client)
	if err != nil {
		log.Error(err, "unable to find span")
		return ctrl.Result{}, nil // nothing else we can do
	}
	if span == nil {
		return ctrl.Result{}, nil // no parent context found; don't create a span for this event
	}*/
	span.SetAttributes(
		semconv.HostNameKey.String(event.Source.Host),
		label.String("type", event.Type),
		label.String("kind", event.InvolvedObject.Kind),
		label.String("namespace", event.InvolvedObject.Namespace),
		label.String("name", event.InvolvedObject.Name),
		label.String("message", event.Message),
	)
	span.End()

	return ctrl.Result{}, nil
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

/*func spanFromObject(ctx context.Context, name string, obj runtime.Object, c client.Client) (trace.Span, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	span, err := tracing.SpanFromAnnotations(name, m.GetAnnotations())
	if err != nil {
		return nil, err
	}
	if span == nil {
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
				return nil, err
			}
			span, err := spanFromObject(ctx, name, owner, c)
			if err != nil {
				return nil, err
			}
			if span != nil {
				return span, nil
			}
		}
	}
	return span, nil
}*/

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
