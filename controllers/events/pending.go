package events

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *EventWatcher) checkPending(ctx context.Context) error {
	log := r.Log
	// Take everything off the pending queue before walking it, so nobody else messes with those events
	r.Lock()
	pending := make([]*corev1.Event, len(r.pending))
	copy(pending, r.pending)
	r.pending = r.pending[:0]
	r.Unlock()
	for { // repeat if we do generate any new spans
		anyEmitted := false
		for i := 0; i < len(pending); {
			ev := pending[i]
			emitted, err := r.emitSpanFromEvent(ctx, log, ev)
			if err != nil {
				return err
			}
			if emitted {
				// delete entry from pending
				pending = append(pending[:i], pending[i+1:]...)
				anyEmitted = true
			} else {
				i++
			}
		}
		if !anyEmitted {
			break
		}
	}
	// Now copy back what we didn't process, adding any new items on r.pending after the ones that were there before
	r.Lock()
	r.pending = append(pending, r.pending...)
	r.Unlock()
	return nil
}

// After we've given up waiting, walk further up the owner chain to look for recent activity;
// if necessary create a new span based off the topmost owner.
func (r *EventWatcher) checkOlderPending(ctx context.Context, threshold time.Time) error {
	r.Lock()
	var olderPending []*corev1.Event
	// Collect older events and remove them from pending, which we unlock before calling any other methods
	for i := 0; i < len(r.pending); {
		event := r.pending[i]
		if eventTime(event).Before(threshold) {
			olderPending = append(olderPending, event)
			r.pending = append(r.pending[:i], r.pending[i+1:]...)
		} else {
			i++
		}
	}
	r.Unlock()
	// Now go through the older events; if we can't map at this point we give up and drop them
	for _, event := range olderPending {
		success, ref, remoteContext, err := r.makeSpanContextFromEvent(ctx, r.Client, event)
		if err != nil {
			if !isNotFound(err) {
				r.Log.Error(err, "dropping span", "name", event.UID)
			}
			continue
		}
		if success {
			span := r.eventToSpan(event, remoteContext)
			r.emitSpan(ctx, ref.object, span)
			if !(ref.IsTopLevel() && remoteContext.HasSpanID()) { // Only store for top-level object if top-level span
				r.recent.store(ref, remoteContext, span.SpanContext)
			}
		}
	}
	return nil
}

// Map the topmost owning object to a span, perhaps creating a new trace
func (r *EventWatcher) makeSpanContextFromEvent(ctx context.Context, client client.Client, event *corev1.Event) (success bool, ref actionReference, remoteContext trace.SpanContext, err error) {
	var apiVersion string
	ref, apiVersion, err = objectFromEvent(ctx, client, event)
	if err != nil {
		return
	}

	if ref.actor.Name != "" {
		// See if we have a recent event matching exactly this ref
		_, remoteContext, success = r.recent.lookupSpanContext(ref)
		if !success {
			// Try the owner on its own, and if found use that as the parent
			remoteContext, _, success = r.recent.lookupSpanContext(actionReference{object: ref.actor})
		}
	}
	if !success {
		var involved runtime.Object
		involved, err = getObject(ctx, r.Client, apiVersion, ref.object.Kind, ref.object.Namespace, ref.object.Name)
		if err != nil {
			if isNotFound(err) { // TODO: could apply naming heuristic to go from a deleted pod to its ReplicaSet
				err = nil
			}
			return
		}

		// See if we can map this object to a trace
		remoteContext, err = r.makeSpanContextFromObject(ctx, involved, eventTime(event))
		if err != nil {
			return
		}
		success = remoteContext.HasTraceID()
	}
	return
}
