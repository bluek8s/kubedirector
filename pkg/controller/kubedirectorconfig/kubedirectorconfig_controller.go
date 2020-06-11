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

package kubedirectorconfig

import (
	"context"
	"fmt"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/shared"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kubedirectorconfig")

// Add creates a new KubeDirectorConfig Controller and adds it to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager
// is Started.
func Add(
	mgr manager.Manager,
) error {

	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(
	mgr manager.Manager,
) reconcile.Reconciler {

	return &ReconcileKubeDirectorConfig{scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(
	mgr manager.Manager,
	r reconcile.Reconciler,
) error {

	// Create a new controller
	c, err := controller.New("kubedirectorconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KubeDirectorConfig.
	err = c.Watch(&source.Kind{Type: &kdv1.KubeDirectorConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKubeDirectorConfig implements
// reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileKubeDirectorConfig{}

const (
	// Period between the time when the controller requeues a request and
	// it's scheduled again for reconciliation. Zero means don't poll.
	// For now we don't need polling on this CR. Some anticipated features
	// will need polling at which point we can change this.
	//reconcilePeriod = 30 * time.Second
	reconcilePeriod = 0
)

// ReconcileKubeDirectorConfig reconciles a KubeDirectorConfig object.
type ReconcileKubeDirectorConfig struct {
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KubeDirectorConfig object
// and makes changes based on the state read and what is in the
// KubeDirectorConfig.Spec.
// Note:
// The Controller will requeue the Request to be processed again if the
// returned error is non-nil or Result.Requeue is true, otherwise upon
// completion it will remove the work from the queue.
func (r *ReconcileKubeDirectorConfig) Reconcile(
	request reconcile.Request,
) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the KubeDirectorConfig instance.
	cr := &kdv1.KubeDirectorConfig{}
	err := shared.Get(context.TODO(), request.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after
			// reconcile request. Owned objects are automatically garbage
			// collected. For additional cleanup logic use finalizers.
			// Return and don't requeue.
			shared.RemoveGlobalConfig()
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult,
			fmt.Errorf("could not fetch KubeDirectorConfig instance: %s", err)
	}

	err = r.syncConfig(reqLogger, cr)
	return reconcileResult, err
}
