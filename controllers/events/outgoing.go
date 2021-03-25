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
		r.Exporter.ExportSpans(ctx, []*tracesdk.SpanSnapshot{prev})
		// We do not remove from bySpanID at this time, in case it is needed for parent chains
	}
	r.outgoing.byRef[ref] = span
	r.outgoing.bySpanID[span.SpanContext.SpanID()] = span

	for parentID := span.ParentSpanID; parentID.IsValid(); {
		if parent, found := r.outgoing.bySpanID[parentID]; found {
			//r.Log.Info("adjusting endtime", "parent", parent.Name, "from", parent.EndTime.Format(timeFmt), "to", span.EndTime.Format(timeFmt))
			parent.EndTime = span.EndTime
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
			r.Exporter.ExportSpans(ctx, []*tracesdk.SpanSnapshot{span})
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
