package events

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/trace"
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

func (r objectReference) Blank() bool {
	return r == objectReference{}
}

// Given an object, come up with some source for the change, and the time it happened
func getUpdateSource(obj v1.Object, subFields ...string) (source string, operation string, ts time.Time) {
	// If it has managed fields, return the newest change that updated the spec
	for _, mf := range obj.GetManagedFields() {
		var fields map[string]interface{}
		err := json.Unmarshal(mf.FieldsV1.Raw, &fields)
		if err != nil {
			continue
		}
		if _, found, _ := unstructured.NestedFieldNoCopy(fields, subFields...); found && mf.Time.Time.After(ts) {
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
func (r *EventWatcher) createTraceFromTopLevelObject(ctx context.Context, obj runtime.Object, eventTime time.Time) (*tracesdk.SpanSnapshot, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	updateSource, operation, updateTime := getUpdateSource(m, "f:spec")
	res := r.getResource(source{name: updateSource})

	if updateTime.IsZero() { // We didn't find a time in the object
		updateTime = eventTime
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	attrs := []attribute.KeyValue{
		attribute.String("k8s.namespace.name", m.GetNamespace()),
		attribute.String("k8s."+strings.ToLower(kind)+".name", m.GetName()),
		attribute.String("kind", kind),
		attribute.Int64("generation", m.GetGeneration()),
	}

	spanData := &tracesdk.SpanSnapshot{
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: objectToTraceID(m),
			SpanID:  objectToSpanID(m),
		}),
		SpanKind:   trace.SpanKindInternal,
		Name:       fmt.Sprintf("%s.%s", obj.GetObjectKind().GroupVersionKind().Kind, operation),
		StartTime:  updateTime,
		EndTime:    updateTime,
		Attributes: attrs,
		Resource:   res,
	}

	return spanData, nil
}

// create a spanID that will be consistent for a particular object
func objectToSpanID(m v1.Object) trace.SpanID {
	f := fnv.New64a()
	_, _ = f.Write([]byte(m.GetUID()))
	_ = binary.Write(f, binary.LittleEndian, m.GetGeneration())
	var h trace.SpanID
	_ = f.Sum(h[:0])
	return h
}

// we include the generation, so changes to the spec will start a new trace.
func objectToTraceID(m v1.Object) trace.TraceID {
	f := fnv.New128a()
	_, _ = f.Write([]byte(m.GetUID()))
	_ = binary.Write(f, binary.LittleEndian, m.GetGeneration())
	var h trace.TraceID
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
	return obj, errors.Wrap(err, "unable to get object")
}

// Canonicalise strings via lowercase - Kubernetes is case-independent.
// (we don't canonicalise Kind)
func lc(s string) string {
	return strings.ToLower(s)
}

func refFromObjRef(oRef corev1.ObjectReference) objectReference {
	return objectReference{
		Kind:      oRef.Kind,
		Namespace: lc(oRef.Namespace),
		Name:      lc(oRef.Name),
	}
}

func refFromOwner(oRef v1.OwnerReference, namespace string) objectReference {
	return objectReference{
		Kind:      oRef.Kind,
		Namespace: lc(namespace),
		Name:      lc(oRef.Name),
	}
}

func refFromObject(obj v1.Object) objectReference {
	ty := obj.(v1.Type)
	return objectReference{
		Kind:      ty.GetKind(),
		Namespace: lc(obj.GetNamespace()),
		Name:      lc(obj.GetName()),
	}
}
