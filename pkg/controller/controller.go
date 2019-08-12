package controller

import (
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

// GetKubeDirectorReference is a utility function to fetch a reference
// to the kubedirector deployment object
func GetKubeDirectorReference(
	log logr.Logger,
) (*metav1.OwnerReference, error) {

	// Fetch our deployment object
	kdName, err := k8sutil.GetOperatorName()
	if err != nil {
		log.Error(err, "failed to get kubedirector deployment name")
		return nil, err
	}

	kd, err := observer.GetDeployment(kdName)
	if err != nil {
		log.Error(err, "failed to get kubedirector deployment object")
		return nil, err
	}

	return metav1.NewControllerRef(kd, schema.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "Deployment",
	}), nil
}
