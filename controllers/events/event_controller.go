package events

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
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

	return ctrl.Result{}, nil
}

// SetupWithManager to set up the watcher
func (r *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
