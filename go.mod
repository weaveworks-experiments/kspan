module github.com/bboreham/kspan

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.0.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.0+incompatible // indirect
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/structured-merge-diff v0.0.0-20190525122527-15d366b2352e // indirect
)

replace sigs.k8s.io/controller-runtime => github.com/bboreham/controller-runtime v0.5.12-0.20200930113325-b26941b0dd19
