// Copyright 2019 Hewlett Packard Enterprise Development LP

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/bluek8s/kubedirector/pkg/apis"
	"github.com/bluek8s/kubedirector/pkg/controller"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/bluek8s/kubedirector/pkg/validator"
	"github.com/bluek8s/kubedirector/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("kubedirector")

func printVersion() {

	log.Info(fmt.Sprintf("KubeDirector Version: %v", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {

	// Add the zap logger flag set to the CLI. The flag set must be added
	// before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime).
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap flags are
	// configured (or if the zap flag set is not being used), this defaults to
	// a production zap logger.
	//
	// The logger instantiated here can be changed to any logger implementing
	// the logr.Logger interface. This logger will be propagated through the
	// whole operator, generating uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	// Create the overall controller-runtime manager. Note that it will watch
	// all namespaces because of the specified emptystring for Namespace.
	// (We'll reject KubeDirectorConfig requests in the validator when the
	// namespace isn't the KubeDirector namespace.)
	// Leader election configured here in order to do lease-based leader
	// acqusition; as opposed to "leader for life" style which depends on
	// timely pod eviction of dead pods (which may not happen at all,
	// depending on eviction settings and overall cluster config).
	mgr, mgrErr := manager.New(shared.Config(), manager.Options{
		Namespace:          "",
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		LeaderElection:     true,
		LeaderElectionID:   "kubedirector-lock",
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if mgrErr != nil {
		log.Error(mgrErr, "failed to create manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if schemeErr := apis.AddToScheme(mgr.GetScheme()); schemeErr != nil {
		log.Error(schemeErr, "failed to add KubeDirector CRs to scheme")
		os.Exit(1)
	}

	// Setup all Controllers
	if controllerErr := controller.AddToManager(mgr); controllerErr != nil {
		log.Error(controllerErr, "failed to add controllers to manager")
		os.Exit(1)
	}

	// Add the Metrics Service
	// XXX Commenting out until we can test properly.
	//	addMetrics(context.TODO(), shared.Config(), "")

	// See https://github.com/bluek8s/kubedirector/issues/173
	// Since we are not using the manager's webhook framework and are
	// setting up our own validation server we need to use a temporary
	// client, initialized in the shared package, to do all the K8s
	// CRUD operations required to do that because the manager's client
	// won't work before mgr.Start() is called and the the manager's
	// cache is initialized. Once the cache is initialized we can start
	// using the manager's split (caching) client.
	stopCh := signals.SetupSignalHandler()
	go func() {
		log.Info("Waiting for client cache sync")
		if mgr.GetCache().WaitForCacheSync(stopCh) {
			log.Info("Client cache sync successful")
			shared.SetClient(mgr.GetClient())
		} else {
			log.Error(errors.New("Client cache sync failed"), "")
		}
	}()

	// Fetch a reference to the KubeDirector Deployment object
	ownerReference, ownerReferenceErr := observer.GetKubeDirectorReference()
	if ownerReferenceErr != nil {
		log.Error(ownerReferenceErr, "failed to get a reference to the KubeDirector deployment object")
		os.Exit(1)
	}

	validatorErr := validator.InitValidationServer(*ownerReference)
	if validatorErr != nil {
		log.Error(validatorErr, "failed to initialize validation server")
		os.Exit(1)
	}

	go func() {
		log.Info("Starting admission validation server")
		validator.StartValidationServer()
	}()

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if cmdErr := mgr.Start(stopCh); cmdErr != nil {
		log.Error(cmdErr, "Manager exited non-zero")
		os.Exit(1)
	}
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(
	ctx context.Context,
	cfg *rest.Config,
	namespace string,
) {

	if err := serveCRMetrics(cfg); err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(
	cfg *rest.Config,
) error {

	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
