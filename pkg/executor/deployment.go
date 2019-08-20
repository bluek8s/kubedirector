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

package executor

import (
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
