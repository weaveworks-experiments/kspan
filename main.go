/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/weaveworks-experiments/kspan/controllers/events"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
}

func setupOTLP(ctx context.Context, addr string, headers string, secured bool) (tracesdk.SpanExporter, error) {
	setupLog.Info("Setting up OTLP Exporter", "addr", addr)

	var exp *otlp.Exporter
	var err error

	headersMap := make(map[string]string)
	if headers != "" {
		ha := strings.Split(headers, ",")
		for _, h := range ha {
			parts := strings.Split(h, "=")
			if len(parts) != 2 {
				setupLog.Error(errors.New("Error parsing OTLP header"), "header parts length is not 2", "header", h)
				continue
			}
			headersMap[parts[0]] = parts[1]
		}
	}

	if secured {
		exp, err = otlp.NewExporter(
			ctx,
			otlpgrpc.NewDriver(
				otlpgrpc.WithEndpoint(addr),
				otlpgrpc.WithHeaders(headersMap),
				otlpgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
			),
		)
	} else {
		exp, err = otlp.NewExporter(
			ctx,
			otlpgrpc.NewDriver(
				otlpgrpc.WithEndpoint(addr),
				otlpgrpc.WithHeaders(headersMap),
				otlpgrpc.WithInsecure(),
			),
		)
	}
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return exp, err
}

func main() {
	var metricsAddr string
	var otlpAddr string
	var otlpHeaders string
	var otlpSecured bool
	var captureFile string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&otlpAddr, "otlp-addr", "otlp-collector.default:55680", "Address to send traces to")
	flag.StringVar(&otlpHeaders, "otlp-headers", "", "Add headers key/values pairs to OTLP communication")
	flag.BoolVar(&otlpSecured, "otlp-secured", false, "Use TLS for OTLP export")
	flag.StringVar(&captureFile, "capture-to", "", "Write out all updates received to this file")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	ctx := context.Background()
	spanExporter, err := setupOTLP(ctx, otlpAddr, otlpHeaders, otlpSecured)
	if err != nil {
		setupLog.Error(err, "unable to set up tracing")
		os.Exit(1)
	}
	resourceAttrs, err := resource.New(ctx)
	if err != nil {
		setupLog.Error(err, "unable to initialize builtin resource attributes")
	}

	defer func() {
		err := spanExporter.Shutdown(ctx)
		if err != nil {
			setupLog.Error(err, "unable to gracefully shutdown exporter")
		}
	}()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	var capture io.WriteCloser
	if captureFile != "" {
		capture, err = os.Create(captureFile)
		if err != nil {
			setupLog.Error(err, "unable to open capture file")
			os.Exit(1)
		}
	}

	if err = (&events.EventWatcher{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log,
		Exporter: spanExporter,
		Resource: resourceAttrs,
		Capture:  capture,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Events")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// Close capture file when program shuts down
	if err := mgr.Add(manager.RunnableFunc(func(stop <-chan struct{}) error {
		<-stop
		return capture.Close()
	})); err != nil {
		setupLog.Error(err, "unable to add close function")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
