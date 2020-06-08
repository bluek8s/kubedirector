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

package observer

import (
	"context"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// GetCluster finds the k8s KubeDirectorCluster with the given name in the
// given namespace.
func GetCluster(
	namespace string,
	clusterName string,
) (*kdv1.KubeDirectorCluster, error) {

	result := &kdv1.KubeDirectorCluster{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: clusterName},
		result,
	)
	return result, err
}

// GetStatefulSet finds the k8s StatefulSet with the given name in the given
// namespace.
func GetStatefulSet(
	namespace string,
	statefulSetName string,
) (*appsv1.StatefulSet, error) {

	result := &appsv1.StatefulSet{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: statefulSetName},
		result,
	)
	return result, err
}

// GetService finds the k8s Service with the given name in the given namespace.
func GetService(
	namespace string,
	serviceName string,
) (*corev1.Service, error) {

	result := &corev1.Service{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: serviceName},
		result,
	)
	return result, err
}

// GetPod finds the k8s Pod with the given name in the given namespace.
func GetPod(
	namespace string,
	podName string,
) (*corev1.Pod, error) {

	result := &corev1.Pod{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: podName},
		result,
	)
	return result, err
}

// GetConfigMap finds the k8s ConfigMap with the given name in the given namespace.
func GetConfigMap(
	namespace string,
	cmName string,
) (*corev1.ConfigMap, error) {

	result := &corev1.ConfigMap{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: cmName},
		result,
	)
	return result, err
}

// GetSecret finds the k8s Secret with the given name in the given namespace.
func GetSecret(
	namespace string,
	secretName string,
) (*corev1.Secret, error) {

	result := &corev1.Secret{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: secretName},
		result,
	)
	return result, err
}

// GetPVC finds the k8s PersistentVolumeClaim with the given name in the given
// namespace.
func GetPVC(
	namespace string,
	pvcName string,
) (*corev1.PersistentVolumeClaim, error) {

	result := &corev1.PersistentVolumeClaim{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: pvcName},
		result,
	)
	return result, err
}

// GetApp fetches the k8s KubeDirectorApp resource with the given name in
// the given namespace.
func GetApp(
	namespace string,
	appName string,
) (*kdv1.KubeDirectorApp, error) {

	result := &kdv1.KubeDirectorApp{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: appName},
		result,
	)

	return result, err
}

// GetValidatorWebhook fetches the webhook validator resource in
// KubeDirector's namespace.
func GetValidatorWebhook(
	validator string,
) (*v1beta1.MutatingWebhookConfiguration, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &v1beta1.MutatingWebhookConfiguration{}
	err = shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: validator},
		result,
	)
	return result, err
}

// GetDeployment fetches the deployment resource in KubeDirector's namespace.
func GetDeployment(
	deploymentName string,
) (*appsv1.Deployment, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &appsv1.Deployment{}
	err = shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: deploymentName},
		result,
	)
	return result, err
}

// GetKubeDirectorReference is a utility function to fetch a reference
// to the kubedirector deployment object
func GetKubeDirectorReference() (*metav1.OwnerReference, error) {

	// Fetch our deployment object
	kdName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}

	kd, err := GetDeployment(kdName)
	if err != nil {
		return nil, err
	}

	return metav1.NewControllerRef(kd, schema.GroupVersionKind{
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "Deployment",
	}), nil
}

// GetKDConfig fetches kubedirector config CR in KubeDirector's namespace.
func GetKDConfig(
	kdConfigName string,
) (*kdv1.KubeDirectorConfig, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}

	result := &kdv1.KubeDirectorConfig{}
	err = shared.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: kdConfigName},
		result,
	)
	return result, err
}

// GetStorageClass fetches the storage class resource with a given name.
func GetStorageClass(
	storageClassName string,
) (*storagev1.StorageClass, error) {

	result := &storagev1.StorageClass{}
	err := shared.Get(
		context.TODO(),
		types.NamespacedName{Name: storageClassName},
		result,
	)
	return result, err
}

// GetDefaultStorageClass returns the default storage class, if any, as
// defined by k8s.
func GetDefaultStorageClass() (*storagev1.StorageClass, error) {

	// Namespace does not matter for this query; leave blank.
	result := &storagev1.StorageClassList{}
	err := shared.List(context.TODO(), result)
	if err != nil {
		return nil, err
	}
	numClasses := len(result.Items)
	for i := 0; i < numClasses; i++ {
		sc := &(result.Items[i])
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc, nil
		}
		if sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
			return sc, nil
		}
	}
	return nil, nil
}
