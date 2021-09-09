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

// Service names size have a limitation of max 63 characters. The service
// names are derived from statefulset names that have a 5 character UID
// appended towards the end. While calculating the max prefix size for the
// service names, the 5 digit UID and the 4 digit maxKDMember size (1000)
// should be accounted for.
// Also, as part of stateful pod creating a 10 digit hash value is added
// to the controller revision hash label which needs to be accounted for
// while calculating the prefix size.
// Naming scheme for the service is as follows: prefix + UID + member index
// Naming scheme for the label is as follows: prefix + UID + hash value
// Since, the max member size currently is restricted to be 4 characters, take
// the max of hash value digits and member size digits which is 10.
// Prefix calculation is done as following = 63 - 10 - 5 - 2 ('-' characters) = 46.
const nameLengthLimit = 46

// ownerReferences creates an owner reference spec that identifies the
// custom resource as the owner.
func ownerReferences(
	cr shared.KubeDirectorObject,
) []metav1.OwnerReference {

	// IF THIS IS EVER CHANGED TO RETURN MORE THAN ONE REFERENCE for some
	// reason, then ownerReferencesPresent below will also need to be
	// changed.
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(cr, schema.GroupVersionKind{
			Group:   kdv1.SchemeGroupVersion.Group,
			Version: kdv1.SchemeGroupVersion.Version,
			Kind:    cr.GetObjectKind().GroupVersionKind().Kind,
		}),
	}
}

// ownerReferencesPresent determines whether the desired references (from
// the ownerReferences func) are present in the CR.
func ownerReferencesPresent(
	cr shared.KubeDirectorObject,
	currentRefs []metav1.OwnerReference,
) bool {

	// As mentioned above, for simplicity we leverage the fact that
	// we only require one owner reference. Also we probably don't need/want
	// to do an entire struct compare; only the fields we really care about.
	desiredRef := &(ownerReferences(cr)[0])
	for _, ref := range currentRefs {
		if (ref.APIVersion == desiredRef.APIVersion) &&
			(ref.Kind == desiredRef.Kind) &&
			(ref.Name == desiredRef.Name) &&
			(ref.UID == desiredRef.UID) &&
			(ref.Controller != nil) &&
			(*ref.Controller == true) {
			return true
		}
	}
	return false
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

// MungObjectName is a utility function that truncates the object names
// to be below nameLengthLimit threshold set for the CrNameRole naming scheme.
// The function also replaces '.' (dot) and '_' (underscore) characters with a
// '-' (dash).
func MungObjectName(
	name string,
) string {
	length := len(name)
	var modName string

	if length == 0 {
		return name
	}

	for i := 0; i < length && i < nameLengthLimit; i++ {
		if name[i] == '.' || name[i] == '_' {
			if i != nameLengthLimit-1 {
				modName += string('-')
			}
		} else {
			modName += strings.ToLower(string(name[i]))
		}
	}

	return modName
}
