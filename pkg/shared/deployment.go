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

package shared

import (
	"context"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// GetDeployment fetches the deployment resource in KubeDirector's namespace.
func GetDeployment(
	deploymentName string,
) (*appsv1.Deployment, error) {

	kdNamespace, err := GetKubeDirectorNamespace()
	if err != nil {
		return nil, err
	}
	result := &appsv1.Deployment{}
	err = Client().Get(
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
