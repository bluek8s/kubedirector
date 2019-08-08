// Copyright 2018 BlueData Software, Inc.

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
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/bluek8s/kubedirector/pkg/validator"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"runtime"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/bluek8s/kubedirector/pkg/apis"
	"github.com/bluek8s/kubedirector/pkg/controller"
	"github.com/bluek8s/kubedirector/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 60000
)
var log = logf.Log.WithName("kubedirector")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Operator-sdk Version: %v", sdkVersion.Version))
	log.Info(fmt.Sprintf("KubeDirector Version: %v", version.Version))
}

func main() {

	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Become the leader before proceeding
	ctx := context.TODO()
	err = leader.Become(ctx, "kubedirector-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(shared.Config(), manager.Options{
		Namespace:          namespace,

		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "failed to add KubeDirector CRs to scheme")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create Service object to expose the metrics port.
	_, err = metrics.ExposeMetricsPort(ctx, metricsPort)
	if err != nil {
		log.Info(err.Error())
	}

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

	// Fetch our deployment object
	kdName, err := k8sutil.GetOperatorName()
	if err != nil {
		log.Error(err, "failed to get kubedirector deployment name")
		os.Exit(1)
	}

	kd, err := observer.GetDeployment(kdName)
	if err != nil {
		log.Error(err, "failed to get kubedirector deployment object")
		os.Exit(1)
	}

	err = validator.InitValidationServer(
		*metav1.NewControllerRef(
			kd,
			schema.GroupVersionKind{
				Group:   appsv1.SchemeGroupVersion.Group,
				Version: appsv1.SchemeGroupVersion.Version,
				Kind:    "Deployment",
			}),
	)
	if err != nil {
		log.Error(err, "failed to initialize validation server")
		os.Exit(1)
	}

	go func() {
		log.Info("Starting admission validation server")
		validator.StartValidationServer()
	}()

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(stopCh); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
