package events

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/semconv"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	r.initialize()

	return ctx, r, exporter, log
}

func newFakeExporter() *fakeExporter {
	return &fakeExporter{}
}

// records spans sent to it, for testing purposes
type fakeExporter struct {
	spanData []*tracesdk.SpanData
}

func (f *fakeExporter) reset() {
	f.spanData = nil
}

func (f *fakeExporter) dump() []string {
	spanMap := make(map[trace.SpanID]int)
	for i, d := range f.spanData {
		spanMap[d.SpanContext.SpanID] = i
	}
	var ret []string
	for i, d := range f.spanData {
		parent, found := spanMap[d.ParentSpanID]
		var parentStr string
		if found {
			parentStr = fmt.Sprintf(" (%d)", parent)
		}
		message := labelValue(d.Attributes, label.Key("message"))
		resourceName := labelValue(d.Resource.Attributes(), semconv.ServiceNameKey)
		ret = append(ret, fmt.Sprintf("%d: %s %s%s %s", i, resourceName, d.Name, parentStr, message))
	}
	return ret
}

// ExportSpans implements trace.SpanExporter
func (f *fakeExporter) ExportSpans(ctx context.Context, spanData []*tracesdk.SpanData) error {
	f.spanData = append(f.spanData, spanData...)
	return nil
}

// Shutdown implements trace.SpanExporter
func (f *fakeExporter) Shutdown(ctx context.Context) error {
	return nil
}

func labelValue(labels []label.KeyValue, key label.Key) string {
	for _, lbl := range labels {
		if lbl.Key == key {
			return lbl.Value.AsString()
		}
	}
	return ""
}
