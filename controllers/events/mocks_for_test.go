package events

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Initialize an EventWatcher, context and logger ready for testing
func newTestEventWatcher(initObjs ...runtime.Object) (context.Context, *EventWatcher, *fakeExporter, logr.Logger) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	log := zap.New(zap.UseDevMode(true))

	fakeClient := fake.NewFakeClientWithScheme(scheme, initObjs...)
	exporter := newFakeExporter()

	r := &EventWatcher{
		Client:   fakeClient,
		Log:      log,
		Exporter: exporter,
	}

	r.initialize(scheme)

	return ctx, r, exporter, log
}

func newFakeExporter() *fakeExporter {
	return &fakeExporter{}
}

// records spans sent to it, for testing purposes
type fakeExporter struct {
	SpanSnapshot []*tracesdk.SpanSnapshot
}

func (f *fakeExporter) dump() []string {
	f.sort()
	spanMap := make(map[trace.SpanID]int)
	for i, d := range f.SpanSnapshot {
		spanMap[d.SpanContext.SpanID()] = i
	}
	var ret []string
	for i, d := range f.SpanSnapshot {
		parent, found := spanMap[d.ParentSpanID]
		var parentStr string
		if found {
			parentStr = fmt.Sprintf(" (%d)", parent)
		}
		message := attributeValue(d.Attributes, attribute.Key("message"))
		resourceName := attributeValue(d.Resource.Attributes(), semconv.ServiceNameKey)
		ret = append(ret, fmt.Sprintf("%d: %s %s%s %s", i, resourceName, d.Name, parentStr, message))
	}
	return ret
}

// ExportSpans implements trace.SpanExporter
func (f *fakeExporter) ExportSpans(ctx context.Context, SpanSnapshot []*tracesdk.SpanSnapshot) error {
	f.SpanSnapshot = append(f.SpanSnapshot, SpanSnapshot...)
	return nil
}

// Shutdown implements trace.SpanExporter
func (f *fakeExporter) Shutdown(ctx context.Context) error {
	return nil
}

func attributeValue(attributes []attribute.KeyValue, key attribute.Key) string {
	for _, lbl := range attributes {
		if lbl.Key == key {
			return lbl.Value.AsString()
		}
	}
	return ""
}

// Sort the captured spans so they are in a predictable order to check expected output.
// Use depth-first search, with edges ordered by start-time where those differ.
func (f *fakeExporter) sort() {
	sort.Stable(SortableSpans(f.SpanSnapshot))

	// Make a map from span-id to index in the set
	spanMap := make(map[trace.SpanID]int)
	for i, d := range f.SpanSnapshot {
		spanMap[d.SpanContext.SpanID()] = i
	}

	// Prepare vertexes for depth-first sort
	v := make([]*vertex, len(f.SpanSnapshot))
	for i := range f.SpanSnapshot {
		v[i] = &vertex{value: i}
	}
	topSpan := -1
	for i, s := range f.SpanSnapshot {
		if s.ParentSpanID.IsValid() {
			p := spanMap[s.ParentSpanID]
			v[p].connect(v[i])
		} else {
			if topSpan != -1 {
				panic("More than one top span")
			}
			topSpan = i
		}
	}

	sortedSpans := make([]*tracesdk.SpanSnapshot, 0, len(f.SpanSnapshot))
	t := dfs{
		visit: func(v *vertex) {
			sortedSpans = append(sortedSpans, f.SpanSnapshot[v.value])
		},
	}
	t.walk(v[topSpan])

	f.SpanSnapshot = sortedSpans
}

// SortableSpans attaches the methods of sort.Interface to []*tracesdk.SpanSnapshot, sorting by start time.
type SortableSpans []*tracesdk.SpanSnapshot

func (x SortableSpans) Len() int           { return len(x) }
func (x SortableSpans) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x SortableSpans) Less(i, j int) bool { return x[i].StartTime.Before(x[j].StartTime) }

// Single-use DFS implementation.
// Not using one from a library; see https://github.com/gonum/gonum/issues/1595
type vertex struct {
	visited    bool
	value      int
	neighbours []*vertex
}

func (v *vertex) connect(vertex *vertex) {
	v.neighbours = append(v.neighbours, vertex)
}

type dfs struct {
	visit func(*vertex)
}

func (d *dfs) walk(vertex *vertex) {
	if vertex.visited {
		return
	}
	vertex.visited = true
	d.visit(vertex)
	for _, v := range vertex.neighbours {
		d.walk(v)
	}
}
