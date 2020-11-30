package events

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This is how we refer to objects - it's a subset of corev1.ObjectReference
type objectReference struct {
	Kind      string
	Namespace string
	Name      string
}

func (r objectReference) String() string {
	return fmt.Sprintf("%s:%s/%s", r.Kind, r.Namespace, r.Name)
}

func (r objectReference) blank() bool {
	return r == objectReference{}
}

// Given an object, come up with some source for the change, and the time it happened
func getUpdateSource(obj v1.Object) (source string, operation string, ts time.Time) {
	// If it has managed fields, return the newest change that updated the spec
	for _, mf := range obj.GetManagedFields() {
		var fields map[string]interface{}
		err := json.Unmarshal(mf.FieldsV1.Raw, &fields)
		if err != nil {
			continue
		}
		if _, found := fields["f:spec"]; found && mf.Time.Time.After(ts) {
			ts = mf.Time.Time
			source = mf.Manager
			operation = string(mf.Operation)
		}
	}
	if !ts.IsZero() {
		return source, operation, ts
	}
	// TODO: try some other ways
	return "unknown", "unknown", ts
}

// If we reach an object with no owner and no recent events, start a new trace.
// Trace ID is a hash of object UID + generation.
func (r *EventWatcher) createTraceFromTopLevelObject(ctx context.Context, obj runtime.Object) (*tracesdk.SpanData, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	updateSource, operation, updateTime := getUpdateSource(m)
	res := r.getResource(source{name: updateSource})

	if updateTime.IsZero() { // We didn't find a time in the object
		updateTime = time.Now() // TODO: can we use the time of the event that triggered this instead?
	}

	attrs := []label.KeyValue{
		label.String("namespace", m.GetNamespace()),
		label.String("name", m.GetName()),
		label.Int64("generation", m.GetGeneration()),
	}

	spanData := &tracesdk.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: objectToTraceID(m),
			SpanID:  objectToSpanID(m),
		},
		SpanKind:   trace.SpanKindInternal,
		Name:       fmt.Sprintf("%s.%s", obj.GetObjectKind().GroupVersionKind().Kind, operation),
		StartTime:  updateTime,
		EndTime:    updateTime,
		Attributes: attrs,
		Resource:   res,
	}

	return spanData, nil
}

func objectToSpanID(m v1.Object) trace.SpanID {
	f := fnv.New64a()
	f.Write([]byte(m.GetUID()))
	binary.Write(f, binary.LittleEndian, m.GetGeneration())
	var h trace.SpanID
	_ = f.Sum(h[:0])
	return h
}

func objectToTraceID(m v1.Object) trace.ID {
	f := fnv.New128a()
	f.Write([]byte(m.GetUID()))
	binary.Write(f, binary.LittleEndian, m.GetGeneration())
	var h trace.ID
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
	return objectReference{
		Kind:      oRef.Kind,
		Namespace: oRef.Namespace,
		Name:      oRef.Name,
	}
}

func refFromOwner(oRef v1.OwnerReference, namespace string) objectReference {
	return objectReference{
		Kind:      oRef.Kind,
		Namespace: namespace,
		Name:      oRef.Name,
	}
}

func refFromObjectMeta(obj runtime.Object, m v1.Object) objectReference {
	return objectReference{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: m.GetNamespace(),
		Name:      m.GetName(),
	}
}
