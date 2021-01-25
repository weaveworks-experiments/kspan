module github.com/bboreham/kspan

go 1.13

require (
	github.com/HdrHistogram/hdrhistogram-go v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/onsi/gomega v1.10.1
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	go.opentelemetry.io/otel v0.13.0
	go.opentelemetry.io/otel/exporters/otlp v0.13.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk v0.13.0
	google.golang.org/grpc v1.32.0
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/yaml v1.1.0
)

replace sigs.k8s.io/controller-runtime => github.com/bboreham/controller-runtime v0.5.12-0.20200930113325-b26941b0dd19
