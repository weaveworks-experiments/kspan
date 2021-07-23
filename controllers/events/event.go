package events

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// bundle up two object references to track who did what to whom
type actionReference struct {
	actor  objectReference // this is what did something (may be blank if not known)
	object objectReference // this is what it was done to
}

func (p actionReference) String() string {
	return fmt.Sprintf("actor: %s, object: %s", p.actor, p.object) // TODO improve this
}

func (p actionReference) IsTopLevel() bool {
	return p.actor.Blank()
}

// look for the string 'marker' in 'message' and return the following space-separated word.
// if anything goes wrong, return an empty string.
func extractWordAfter(message, marker string) string {
	pos := strings.Index(message, marker)
	if pos == -1 {
		return ""
	}
	pos += len(marker)
	end := strings.IndexByte(message[pos:], ' ')
	if end == -1 {
		return message[pos:]
	}
	return message[pos : pos+end]
}

// Get the object relating to an event, after applying some heuristics
// or a blank struct if this can't be done
func objectFromEvent(ctx context.Context, client client.Client, event *corev1.Event) (actionReference, string, error) {
	if event.InvolvedObject.Name == "" {
		return actionReference{}, "", fmt.Errorf("no involved object")
	}

	objRef := refFromObjRef(event.InvolvedObject)
	ret := actionReference{
		object: objRef,
	}
	apiVersion := event.InvolvedObject.APIVersion

	// TODO: generalise these, take away source- and kind-specific checks
	switch {
	case event.Source.Component == "deployment-controller" && event.InvolvedObject.Kind == "Deployment":
		// if we have a message like "Scaled down replica set foobar-7ff854f459 to 0"; extract the ReplicaSet name
		name := extractWordAfter(event.Message, "replica set ")
		if name == "" {
			break
		}
		ret.actor = ret.object
		ret.object = objectReference{Kind: "ReplicaSet", Namespace: lc(ret.object.Namespace), Name: lc(name)}
	case event.Source.Component == "replicaset-controller" && event.InvolvedObject.Kind == "ReplicaSet":
		// if we have a message like "Created pod: foo-5c5df9754b-4w2hj"; extract the Pod name
		name := extractWordAfter(event.Message, "pod: ")
		if name == "" {
			break
		}
		ret.actor = ret.object
		ret.object = objectReference{Kind: "Pod", Namespace: lc(ret.object.Namespace), Name: lc(name)}
		apiVersion = "v1"
	case event.Source.Component == "statefulset-controller" && event.InvolvedObject.Kind == "StatefulSet":
		// if we have a message like "create Pod ingester-3 in StatefulSet ingester successful"; extract the Pod name
		name := extractWordAfter(event.Message, "Pod ")
		if name == "" {
			break
		}
		ret.actor = ret.object
		ret.object = objectReference{Kind: "Pod", Namespace: lc(ret.object.Namespace), Name: lc(name)}
		apiVersion = "v1"
	}

	return ret, apiVersion, nil
}

func (r *EventWatcher) eventToSpan(event *corev1.Event, remoteContext trace.SpanContext) *tracesdk.SpanSnapshot {
	// resource says which component the span is seen as coming from
	res := r.getResource(eventSource(event))

	attrs := []attribute.KeyValue{
		attribute.String("kind", event.InvolvedObject.Kind),
		attribute.String("namespace", event.InvolvedObject.Namespace),
		attribute.String("object", event.InvolvedObject.Name),
	}

	if event.Reason != "" {
		attrs = append(attrs, attribute.String("reason", event.Reason))
	}
	if event.Message != "" {
		attrs = append(attrs, attribute.String("message", event.Message)) // maybe this should be a log?
	}
	if event.Name != "" {
		attrs = append(attrs, attribute.String("eventID", event.Namespace+"/"+event.Name))
	}

	statusCode := codes.Ok
	if event.Type != corev1.EventTypeNormal {
		statusCode = codes.Error
	}

	attrs = append(attrs, r.Resource.Attributes()...)

	return &tracesdk.SpanSnapshot{
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: remoteContext.TraceID(),
			SpanID:  eventToSpanID(event),
		}),
		ParentSpanID:    remoteContext.SpanID(),
		SpanKind:        trace.SpanKindInternal,
		Name:            fmt.Sprintf("%s.%s", event.InvolvedObject.Kind, event.Reason),
		StartTime:       eventTime(event),
		EndTime:         eventTime(event),
		Attributes:      attrs,
		StatusCode:      statusCode,
		HasRemoteParent: true,
		Resource:        res,
		//InstrumentationLibrary instrumentation.Library
	}
}

// generate a spanID from an event.  The first time this event is issued has a span ID that can be derived from the event UID
func eventToSpanID(event *corev1.Event) trace.SpanID {
	f := fnv.New64a()
	_, _ = f.Write([]byte(event.UID))
	if event.Count > 0 {
		fmt.Fprint(f, event.Count)
	}
	var h trace.SpanID
	_ = f.Sum(h[:0])
	return h
}

// If time has zero ms, and is close to wall-clock time, use wall-clock time
func adjustEventTime(event *corev1.Event, now time.Time) {
	if event.LastTimestamp.Time.IsZero() {
		return
	}
	if event.LastTimestamp.Time.Nanosecond() == 0 && now.Sub(event.LastTimestamp.Time) < time.Second {
		event.LastTimestamp.Time = now
	}
}

// Some events have just an EventTime; if LastTimestamp is present we prefer that.
func eventTime(event *corev1.Event) time.Time {
	if !event.LastTimestamp.Time.IsZero() {
		return event.LastTimestamp.Time
	}
	return event.EventTime.Time
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
