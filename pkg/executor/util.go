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
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
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

// annotationsForCluster generates a set of resource labels appropriate for
// any component of this KDCluster.
func annotationsForCluster(
	cr *kdv1.KubeDirectorCluster,
) map[string]string {

	var result map[string]string
	appCR, err := catalog.GetApp(cr)
	if err == nil {
		result = map[string]string{
			ClusterAppAnnotation: appCR.Spec.Label.Name,
		}
	} else {
		result = map[string]string{}
	}
	return result
}

// labelsForCluster generates a set of resource labels appropriate for any
// component of this KDCluster.
func labelsForCluster(
	cr *kdv1.KubeDirectorCluster,
) map[string]string {

	result := map[string]string{
		ClusterLabel:           cr.Name,
		ClusterAppLabel:        cr.Spec.AppID,
		ClusterAppCatalogLabel: *(cr.Spec.AppCatalog),
	}
	return result
}

// labelsForRole generates a set of resource labels appropriate for the
// given role. These will be propagated to the statefulset, pods, and
// services related to that role.
func labelsForRole(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) map[string]string {

	result := labelsForCluster(cr)
	result[ClusterRoleLabel] = role.Name
	return result
}

// labelsForStatefulSet generates a set of resource labels appropriate for a
// statefulset in the given role.
func labelsForStatefulSet(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) map[string]string {

	result := labelsForRole(cr, role)
	result[HeadlessServiceLabel] = cr.Name
	return result
}

// labelsForPod generates a set of resource labels appropriate for a pod in
// the given role. This includes any user-requested labels.
func labelsForPod(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) map[string]string {

	result := labelsForStatefulSet(cr, role)
	for name, value := range role.PodLabels {
		result[name] = value
	}
	return result
}

// labelsForService generates a set of resource labels appropriate for the
// services created for a cluster. This includes any user-requested labels.
// role may be nil if this is the headless service.
func labelsForService(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
) map[string]string {

	var result map[string]string
	if role == nil {
		result = labelsForCluster(cr)
	} else {
		result = labelsForRole(cr, role)
		for name, value := range role.ServiceLabels {
			result[name] = value
		}
	}
	return result
}

// createPortNameForService creates the port name for a service endpoint.
// It prefixes the ID with the lowercased URL scheme if given; otherwise
// prefixing with "generic-".
func createPortNameForService(
	portInfo catalog.ServicePortInfo,
) string {

	if portInfo.URLScheme == "" {
		return "generic-" + portInfo.ID
	}
	return strings.ToLower(portInfo.URLScheme) + "-" + portInfo.ID
}
