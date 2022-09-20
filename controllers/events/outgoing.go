package events

import (
	"context"
	"sync"
	"time"

	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

type outgoing struct {
	sync.Mutex

	byRef    map[objectReference]*tracesdk.SpanSnapshot
	bySpanID map[apitrace.SpanID]*tracesdk.SpanSnapshot
}

func newOutgoing() *outgoing {
	return &outgoing{
		byRef:    make(map[objectReference]*tracesdk.SpanSnapshot),
		bySpanID: make(map[apitrace.SpanID]*tracesdk.SpanSnapshot),
	}
}

const timeFmt = "15:04:05.000"

// note we do not return errors, just log them here, because the one place it
// can happen refers to a previous span, so not something the caller can react to.
func (r *EventWatcher) emitSpan(ctx context.Context, ref objectReference, span *tracesdk.SpanSnapshot) {
	r.Log.Info("adding span", "ref", ref, "name", span.Name, "start", span.StartTime.Format(timeFmt), "end", span.EndTime.Format(timeFmt))
	r.outgoing.Lock()
	defer r.outgoing.Unlock()

	if prev, found := r.outgoing.byRef[ref]; found {
		if !prev.StartTime.After(span.StartTime) {
			prev.EndTime = span.StartTime
		} else {
			r.Log.Info("New span before old span", "oldSpan", prev.Name, "oldTime", prev.StartTime.Format(timeFmt), "newSpan", span.Name, "newTime", span.StartTime.Format(timeFmt))
		}
		r.Log.Info("emitting span", "ref", ref, "name", prev.Name)
		err := r.Exporter.ExportSpans(ctx, []*tracesdk.SpanSnapshot{prev})
		if err != nil {
			r.Log.Error(err, "failed to emit span", "ref", ref, "name", prev.Name)
		}
		// We do not remove from bySpanID at this time, in case it is needed for parent chains
	}
	r.outgoing.byRef[ref] = span
	r.outgoing.bySpanID[span.SpanContext.SpanID()] = span

	for parentID := span.ParentSpanID; parentID.IsValid(); {
		if parent, found := r.outgoing.bySpanID[parentID]; found {
			if span.EndTime.After(parent.EndTime) {
				//r.Log.Info("adjusting endtime", "parent", parent.Name, "from", parent.EndTime.Format(timeFmt), "to", span.EndTime.Format(timeFmt))
				parent.EndTime = span.EndTime
			}
			if span.StartTime.Before(parent.StartTime) {
				parent.StartTime = span.StartTime
			}
			if parentID == parent.ParentSpanID {
				r.Log.Info("infinite loop!", "span", span.Name, "parent", parent.Name, "parentid", parentID)
				break
			}
			parentID = parent.ParentSpanID
		} else {
			break
		}
	}
}

func (r *EventWatcher) flushOutgoing(ctx context.Context, threshold time.Time) {
	r.outgoing.Lock()
	defer r.outgoing.Unlock()
	for k, span := range r.outgoing.byRef {
		if !span.EndTime.After(threshold) {
			r.Log.Info("deferred emit", "ref", k, "name", span.Name, "endTime", span.EndTime, "threshold", threshold)
			err := r.Exporter.ExportSpans(ctx, []*tracesdk.SpanSnapshot{span})
			if err != nil {
				r.Log.Error(err, "failed to emit span", "ref", k, "name", span.Name)
			}
			delete(r.outgoing.byRef, k)
			delete(r.outgoing.bySpanID, span.SpanContext.SpanID())
		}
	}
	// Now clear out anything old that is still in bySpanID
	for k, span := range r.outgoing.bySpanID {
		if !span.EndTime.After(threshold) {
			delete(r.outgoing.bySpanID, k)
		}
	}
}
