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
	"log"
	"os"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/propagators"
	tracesdk "go.opentelemetry.io/otel/sdk/export/trace"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/bboreham/kspan/controllers/events"
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

func setupOTLP(collectorAddr, serviceName string) (tracesdk.SpanExporter, error) {
	exp, err := otlp.NewExporter(
		otlp.WithInsecure(),
		otlp.WithAddress(collectorAddr),
	)
	if err != nil {
		return nil, err
	}
	global.SetTextMapPropagator(propagators.TraceContext{})
	return exp, err
}

func main() {
	var metricsAddr string
	var collectorAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&collectorAddr, "otlp-addr", "otlp-collector.default:55680", "Address to send traces to")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	spanExporter, err := setupOTLP(collectorAddr, "events")
	if err != nil {
		setupLog.Error(err, "unable to set up tracing")
		os.Exit(1)
	}
	defer spanExporter.Shutdown(context.Background())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&events.EventWatcher{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log,
		Exporter: spanExporter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Events")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
