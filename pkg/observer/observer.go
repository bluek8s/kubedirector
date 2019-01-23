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

package observer

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetCluster finds the k8s KubeDirectorCluster with the given name in the
// given namespace.
func GetCluster(
	namespace string,
	clusterName string,
) (*kdv1.KubeDirectorCluster, error) {

	result := &kdv1.KubeDirectorCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeDirectorCluster",
			APIVersion: "kubedirector.bluedata.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetStatefulSet finds the k8s StatefulSet with the given name in the given
// namespace.
func GetStatefulSet(
	namespace string,
	statefulSetName string,
) (*appsv1.StatefulSet, error) {

	result := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetService finds the k8s Service with the given name in the given namespace.
func GetService(
	namespace string,
	serviceName string,
) (*v1.Service, error) {

	result := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetPod finds the k8s Pod with the given name in the given namespace.
func GetPod(
	namespace string,
	podName string,
) (*v1.Pod, error) {

	result := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetPVC finds the k8s PersistentVolumeClaim with the given name in the given
// namespace.
func GetPVC(
	namespace string,
	pvcName string,
) (*v1.PersistentVolumeClaim, error) {

	result := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetApp fetches the k8s KubeDirectorApp resource in KubeDirector's namespace.
func GetApp(
	appID string,
) (*kdv1.KubeDirectorApp, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &kdv1.KubeDirectorApp{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeDirectorApp",
			APIVersion: "kubedirector.bluedata.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appID,
			Namespace: kdNamespace,
		},
	}
	err = sdk.Get(result)
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
	result := &v1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      validator,
			Namespace: kdNamespace,
		},
	}
	err = sdk.Get(result)
	return result, err
}

// GetSecret fetches the secret resource in the given namespace.
func GetSecret(
	secretName string,
	namespace string,
) (*v1.Secret, error) {

	result := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}
	err := sdk.Get(result)
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
	result := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: kdNamespace,
		},
	}
	err = sdk.Get(result)
	return result, err
}

// GetKDConfig fetches kubedirector config CR in KubeDirector's namespace.
func GetKDConfig(
	kdConfigName string,
) (*kdv1.KubeDirectorConfig, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &kdv1.KubeDirectorConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeDirectorConfig",
			APIVersion: "kubedirector.bluedata.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kdConfigName,
			Namespace: kdNamespace,
		},
	}
	err = sdk.Get(result)
	return result, err
}

// GetStorageClass fetches the storage class resource with a given name.
func GetStorageClass(
	storageClassName string,
) (*storagev1.StorageClass, error) {

	result := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClassName,
			// Namespace does not matter for this query; leave blank.
		},
	}
	err := sdk.Get(result)
	return result, err
}

// GetStorageClassList fetches all storage class resources.
func GetStorageClassList() ([]storagev1.StorageClass, error) {

	result := &storagev1.StorageClassList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
	}
	// Namespace does not matter for this query; leave blank.
	namespace := ""
	err := sdk.List(namespace, result)
	return result.Items, err
}
