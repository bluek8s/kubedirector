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
	"context"
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetCluster finds the k8s KubeDirectorCluster with the given name in the
// given namespace.
func GetCluster(
	namespace string,
	clusterName string,
	client k8sclient.Client,
) (*kdv1.KubeDirectorCluster, error) {

	result := &kdv1.KubeDirectorCluster{}
	err := client.Get(
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
	client k8sclient.Client,
) (*appsv1.StatefulSet, error) {

	result := &appsv1.StatefulSet{}
	err := client.Get(
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
	client k8sclient.Client,
) (*v1.Service, error) {

	result := &v1.Service{}
	err := client.Get(
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
	client k8sclient.Client,
) (*v1.Pod, error) {

	result := &v1.Pod{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: podName},
		result,
	)
	return result, err
}

// GetPVC finds the k8s PersistentVolumeClaim with the given name in the given
// namespace.
func GetPVC(
	namespace string,
	pvcName string,
	client k8sclient.Client,
) (*v1.PersistentVolumeClaim, error) {

	result := &v1.PersistentVolumeClaim{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: pvcName},
		result,
	)
	return result, err
}

// GetApp fetches the k8s KubeDirectorApp resource in KubeDirector's namespace.
func GetApp(
	clusterNamespace string,
	appID string,
	client k8sclient.Client,
) (*kdv1.KubeDirectorApp, error) {

	appSpec := &kdv1.KubeDirectorApp{}

	// Check to see if this app exists in the cluster namespace
	appErr := client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: clusterNamespace, Name: appID},
		appSpec,
	)

	if appErr == nil {
		return appSpec, appErr
	}

	// Check to see if it is present in our namespace
	kdNamespace, nsErr := shared.GetKubeDirectorNamespace()
	if nsErr != nil {
		return nil, nsErr
	}

	appErr = client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: appID},
		appSpec,
	)
	return appSpec, appErr
}

// GetValidatorWebhook fetches the webhook validator resource in
// KubeDirector's namespace.
func GetValidatorWebhook(
	validator string,
	client k8sclient.Client,
) (*v1beta1.MutatingWebhookConfiguration, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &v1beta1.MutatingWebhookConfiguration{}
	err = client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: validator},
		result,
	)
	return result, err
}

// GetSecret fetches the secret resource in the given namespace.
func GetSecret(
	secretName string,
	namespace string,
	client k8sclient.Client,
) (*v1.Secret, error) {

	result := &v1.Secret{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: namespace, Name: secretName},
		result,
	)
	return result, err
}

// GetDeployment fetches the deployment resource in KubeDirector's namespace.
func GetDeployment(
	deploymentName string,
	client k8sclient.Client,
) (*appsv1.Deployment, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &appsv1.Deployment{}
	err = client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: deploymentName},
		result,
	)
	return result, err
}

// GetKDConfig fetches kubedirector config CR in KubeDirector's namespace.
func GetKDConfig(
	kdConfigName string,
	client k8sclient.Client,
) (*kdv1.KubeDirectorConfig, error) {

	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}

	result := &kdv1.KubeDirectorConfig{}
	err = client.Get(
		context.TODO(),
		types.NamespacedName{Namespace: kdNamespace, Name: kdConfigName},
		result,
	)
	return result, err
}

// GetStorageClass fetches the storage class resource with a given name.
func GetStorageClass(
	storageClassName string,
	client k8sclient.Client,
) (*storagev1.StorageClass, error) {

	result := &storagev1.StorageClass{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{Name: storageClassName},
		result,
	)
	return result, err
}

// GetDefaultStorageClass returns the default storage class, if any, as
// defined by k8s.
func GetDefaultStorageClass(client k8sclient.Client) (*storagev1.StorageClass, error) {

	// Namespace does not matter for this query; leave blank.
	result := &storagev1.StorageClassList{}
	err := client.List(context.TODO(), &k8sclient.ListOptions{}, result)
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
