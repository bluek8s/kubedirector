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

package kubedirectorcluster

import (
	"context"
	"fmt"
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new KubeDirectorCluster Controller and adds it to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

var log = logf.Log.WithName("controller_kubedirectorcluster")

// newReconciler returns a new reconcile.Reconciler .
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKubeDirectorCluster{scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kubedirectorcluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KubeDirectorCluster
	err = c.Watch(&source.Kind{Type: &kdv1.KubeDirectorCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKubeDirectorCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKubeDirectorCluster{}

const (
	// Period between the time when the controller requeues a request
	// and it's scheduled again for reconciliation
	reconcilePeriod = 30 * time.Second
)

// ReconcileKubeDirectorCluster reconciles a KubeDirectorCluster object
type ReconcileKubeDirectorCluster struct {
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KubeDirectorCluster object and makes changes based
// on the state read and what is in the KubeDirectorCluster.Spec
//
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKubeDirectorCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KubeDirectorCluster")
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the KubeDirectorCluster instance
	kdCluster := &kdv1.KubeDirectorCluster{}
	err := shared.Client().Get(context.TODO(), request.NamespacedName, kdCluster)
	if err != nil {
		// If the resource is not found, that means all of
		// the finalizers have been removed, and the kubedirectorcluster
		// resource has been deleted, so there is nothing left
		// to do.
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcileResult,
			fmt.Errorf("could not fetch KubeDirectorCluster instance: %s", err)
	}
	err = r.syncCluster(reqLogger, kdCluster)
	return reconcileResult, err
}