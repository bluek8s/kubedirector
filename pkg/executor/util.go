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

package executor

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ownerReferences creates an owner reference spec that identifies the
// custom resource as the owner.
func ownerReferences(
	cr shared.KubeDirectorObject,
) []metav1.OwnerReference {

	return []metav1.OwnerReference{
		*metav1.NewControllerRef(cr, schema.GroupVersionKind{
			Group:   kdv1.SchemeGroupVersion.Group,
			Version: kdv1.SchemeGroupVersion.Version,
			Kind:    cr.GetObjectKind().GroupVersionKind().Kind,
		}),
	}
}

// labelsForRole generates a set of resource labels appropriate for the
// given role.
func labelsForRole(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) map[string]string {

	result := map[string]string{
		clusterLabel:         cr.Name,
		clusterRoleLabel:     role.Name,
		headlessServiceLabel: headlessServiceName + "-" + cr.Name,
	}
	for name, value := range role.Labels {
		result[name] = value
	}
	return result
}

// labelsForService generates a set of resource labels appropriate for the
// services created for a cluster
func labelsForService(
	cr *kdv1.KubeDirectorCluster,
) map[string]string {

	return map[string]string{
		clusterLabel: cr.Name,
	}
}

// labelsForPod generates a set of resource labels appropriate for a pod in
// the given role.
func labelsForPod(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	podName string,
) map[string]string {

	podLabels := labelsForRole(cr, role)
	podLabels[statefulSetPodLabel] = podName

	return podLabels
}
