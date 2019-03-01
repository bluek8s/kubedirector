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
	"runtime"
	"time"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"github.com/bluek8s/kubedirector/pkg/validator"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}
	// Fetch our deployment object
	kdName, err := k8sutil.GetOperatorName()
	if err != nil {
		logrus.Fatalf("failed to get kubedirector deployment name: %v", err)
	}
	kd, err := observer.GetDeployment(kdName)
	if err != nil {
		logrus.Fatalf("failed to get kubedirector deployment object: %v", err)
	}

	err = validator.InitValidationServer(*metav1.NewControllerRef(kd, kd.GroupVersionKind()))
	if err != nil {
		logrus.Fatalf("failed to initialize validation server: %v", err)
	}

	handler := reconciler.NewHandler()

	go func() {
		logrus.Infof("Starting admission validation server")
		validator.StartValidationServer(handler)
	}()

	type watchInfo struct {
		kind         string
		resyncPeriod time.Duration
	}

	// Add all CR kinds that we want to watch.
	watchParams := []watchInfo{
		{
			kind: "KubeDirectorCluster",
			// The resync period essentially determines how granularly we can detect
			// the completion of cluster config changes. Making this too small can
			// actually be bad in that there is benefit to batch-resolving changes,
			// within KubeDirector but also especially with the cluster's app config
			// scripts.
			resyncPeriod: time.Duration(30) * time.Second,
		},
		{
			kind:         "KubeDirectorConfig",
			resyncPeriod: 0,
		},
	}

	resource := "kubedirector.bluedata.io/v1alpha1"
	for _, w := range watchParams {
		logrus.Infof("Watching %s, %s, %s, %d", resource, w.kind, namespace, w.resyncPeriod)
		sdk.Watch(resource, w.kind, namespace, w.resyncPeriod)
	}
	sdk.Handle(handler)
	sdk.Run(context.TODO())
}
