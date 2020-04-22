// Copyright 2020 Hewlett Packard Enterprise Development LP

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmap

import (
	"context"
	"fmt"

	"github.com/bluek8s/kubedirector/pkg/shared"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_configmap")

// Add creates a new configmap Controller and adds it to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager
// is Started.
func Add(mgr manager.Manager) error {

	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {

	return &ReconcileConfigMap{scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(
	mgr manager.Manager,
	r reconcile.Reconciler,
) error {

	// Create a new controller
	c, err := controller.New("configmap-controller", mgr, controller.Options{MaxConcurrentReconciles: 10, Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource configmap.
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKubeDirectorConfigMap implements
// reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileConfigMap{}

const (
	// We do not need polling for configmaps
	reconcilePeriod = 0
)

// ReconcileConfigMap reconciles a KubeDirectorConfig object.
type ReconcileConfigMap struct {
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ConfigMap object
// and makes changes based on the state read and what is in the
// ConfigMap.Spec.
// Note:
// The Controller will requeue the Request to be processed again if the
// returned error is non-nil or Result.Requeue is true, otherwise upon
// completion it will remove the work from the queue.
func (r *ReconcileConfigMap) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the configmap instance.
	configmap := &corev1.ConfigMap{}
	err := shared.Get(context.TODO(), request.NamespacedName, configmap)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult,
			fmt.Errorf("could not fetch ConfigMap instance: %s", err)
	}

	err = r.syncConfigMap(reqLogger, configmap)
	return reconcileResult, err
}
