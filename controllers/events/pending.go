package events

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/api/trace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *EventWatcher) checkPending(ctx context.Context) error {
	log := r.Log
	log.Info("checkPending")
	for { // repeat if we do generate any new spans
		anyEmitted := false
		for i := 0; i < len(r.pending); { // TODO: locking
			ev := r.pending[i]
			emitted, err := r.emitSpanFromEvent(ctx, log, ev)
			if err != nil {
				return err
			}
			if emitted {
				// delete entry from pending (TODO: locking)
				r.pending = append(r.pending[:i], r.pending[i+1:]...)
				anyEmitted = true
			} else {
				i++
			}
		}
		if !anyEmitted {
			break
		}
	}
	return nil
}

// After we've given up waiting, walk further up the owner chain to look for recent activity;
// if necessary create a new span based off the topmost owner.
func (r *EventWatcher) checkOlderPending(ctx context.Context, threshold time.Time) error {
	// Only go through once; if we can't map at this point we give up
	for i := 0; i < len(r.pending); { // TODO: locking
		event := r.pending[i]
		// skip over recent events
		if eventTime(event).After(threshold) {
			i++
			continue
		}
		success, ref, remoteContext, err := r.makeSpanContextFromEvent(ctx, r.Client, event)
		if err != nil {
			return err
		}
		if success {
			span := r.eventToSpan(event, remoteContext)
			r.emitSpan(ctx, span)
			r.recent.store(ref, span.SpanContext)
		}
		// remove event from pending queue
		r.pending = append(r.pending[:i], r.pending[i+1:]...)
	}
	return nil
}

// Map the topmost owning object to a span, perhaps creating a new trace
func (r *EventWatcher) makeSpanContextFromEvent(ctx context.Context, client client.Client, event *corev1.Event) (success bool, ref parentChild, remoteContext trace.SpanContext, err error) {
	var involved runtime.Object
	involved, ref, err = objectFromEvent(ctx, client, event)
	if err != nil {
		if apierrors.IsNotFound(err) { // TODO: could apply naming heuristic to go from a deleted pod to its ReplicaSet
			err = nil
		}
		return
	}

	// See if we can map this object to a trace
	remoteContext, err = r.makeSpanContextFromObject(ctx, involved)
	if err != nil {
		return
	}
	success = remoteContext.HasTraceID()
	return
}
