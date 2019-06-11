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

package catalog

import (
	"fmt"
	"strconv"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	v1 "k8s.io/api/core/v1"
)

// GetServiceFromID is a utility function that returns the service definition for
// the given service ID, or nil if no such service is defined.
func GetServiceFromID(
	appCR *kdv1.KubeDirectorApp,
	serviceID string,
) *kdv1.Service {

	for _, serviceDef := range appCR.Spec.Services {
		if serviceDef.ID == serviceID {
			return &serviceDef
		}
	}
	return nil
}

// GetAllServiceIDs is a utility function that returns the list of all
// service IDs.
func GetAllServiceIDs(
	appCR *kdv1.KubeDirectorApp,
) []string {

	var services []string
	for _, serviceDef := range appCR.Spec.Services {
		services = append(services, serviceDef.ID)
	}
	return services
}

// GetRoleFromID is a utility function that returns the service definition for
// the given service ID, or nil if no such service is defined.
func GetRoleFromID(
	appCR *kdv1.KubeDirectorApp,
	roleID string,
) *kdv1.NodeRole {

	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == roleID {
			return &nodeRole
		}
	}
	return nil
}

// GetAllRoleIDs is a utility function that returns the list of all node roles
// ID.
func GetAllRoleIDs(
	appCR *kdv1.KubeDirectorApp,
) []string {

	var nodeRoles []string
	for _, nodeRole := range appCR.Spec.NodeRoles {
		nodeRoles = append(nodeRoles, nodeRole.ID)
	}
	return nodeRoles
}

// GetSelectedRoleIDs returns the list of selected roles in the config.
func GetSelectedRoleIDs(
	appCR *kdv1.KubeDirectorApp,
) []string {

	// Will be modified to accept config choices input when that is
	// implemented.

	return appCR.Spec.Config.SelectedRoles
}

// GetRoleCardinality is a utility function that fetches the cardinality value
// for a given app role along with whether it is a scale out cardinality
func GetRoleCardinality(
	appRole *kdv1.NodeRole,
) (int32, bool) {

	var count int
	var isScaleOut = false

	// Check if it is a scaleout cardinality
	if strings.HasSuffix(appRole.Cardinality, "+") {
		count, _ = strconv.Atoi(strings.Trim(appRole.Cardinality, "+"))
		isScaleOut = true
	} else {
		count, _ = strconv.Atoi(appRole.Cardinality)
	}
	return int32(count), isScaleOut
}

// GetRoleMinResources is a utility function that fetching the minimum resources
// for a given app role
func GetRoleMinResources(
	appRole *kdv1.NodeRole,
) *v1.ResourceList {

	return appRole.MinResources
}

// PortsForRole returns list of service port info (id and port num) for a given role.
// This will be used to export those ports as NodePort/LoadBalancer
func PortsForRole(
	cr *kdv1.KubeDirectorCluster,
	role string,
) ([]ServicePortInfo, error) {
	//) ([]int32, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return nil, err
	}

	var result []ServicePortInfo

	// Match the role in the roleService and based on that fetch the service
	// endpoint ports matching the service IDs.
	for _, roleService := range appCR.Spec.Config.RoleServices {
		if roleService.RoleID == role {
			for _, service := range appCR.Spec.Services {
				if shared.StringInList(service.ID, roleService.ServiceIDs) {
					if service.Endpoint.Port != nil {
						servicePortInfo := ServicePortInfo{
							ID:   service.ID,
							Port: *(service.Endpoint.Port),
						}
						result = append(result, servicePortInfo)
					}
				}
			}
			break
		}
	}

	return result, nil
}

// ImageForRole returns the image to be used for pods in a given role.
func ImageForRole(
	cr *kdv1.KubeDirectorCluster,
	role string,
) (string, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return "", err
	}

	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			if nodeRole.ImageRepoTag != nil {
				return *(nodeRole.ImageRepoTag), nil
			}
			// Should never reach here.
			return "", fmt.Errorf(
				"Image repo tag not set for role {%s} in app {%s}",
				role,
				cr.Spec.AppID,
			)
		}
	}

	// Should never reach here.
	return "", fmt.Errorf(
		"Role {%s} not found for app {%s} when searching for image repo tag",
		role,
		cr.Spec.AppID,
	)
}

// AppSetupPackageURL returns the app setup package url for a given role. The
// fact that this function is invoked means that setup package was specified
// either for the node role or the application as a whole.
func AppSetupPackageURL(
	cr *kdv1.KubeDirectorCluster,
	role string,
) (string, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return "", err
	}

	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			setupPackage := nodeRole.SetupPackage

			// setupPackage will always be set because we mutated the spec during
			// validation.
			if setupPackage.IsNull == false {
				return setupPackage.PackageURL.PackageURL, nil
			}

			// No config package for this role.
			return "", nil
		}
	}

	// Should never reach here.
	return "", fmt.Errorf(
		"Role {%s} not found for app {%s} when searching for config package",
		role,
		cr.Spec.AppID,
	)
}

// SystemdRequired checks whether systemctl mounts are required for a given
// app.
func SystemdRequired(
	cr *kdv1.KubeDirectorCluster,
) (bool, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return false, err
	}

	return appCR.Spec.SystemdRequired, nil
}

// AgentRequired checks whether agent installation is required for a given app.
func AgentRequired(
	cr *kdv1.KubeDirectorCluster,
) bool {

	return false // currently, always false
}

// AppCapabilities fetches the required capabilities for a given app
func AppCapabilities(
	cr *kdv1.KubeDirectorCluster,
) ([]v1.Capability, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return []v1.Capability{}, err
	}

	return appCR.Spec.Capabilities, nil
}

// AppPersistDirs fetches the required directories for a given role that
// has be persisted on a PVC.
func AppPersistDirs(
	cr *kdv1.KubeDirectorCluster,
	role string,
) (*[]string, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return nil, err
	}

	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			// Validation hook has already mutated the role's PersistDirs value
			// to match the global default if it was unspecified. If neither
			// were specified then it will be nil, which is an acceptable
			// result for the caller too.
			return nodeRole.PersistDirs, nil
		}
	}

	// Should never reach here.
	return nil, fmt.Errorf(
		"Role {%s} not found for app {%s} when searching for persist dirs",
		role,
		cr.Spec.AppID,
	)
}

// GetApp returns the app type definition for the given virtual cluster. If
// it has already been fetched and cached, return the cached spec. Otherwise
// fetch, cache, and return it.
func GetApp(
	cr *kdv1.KubeDirectorCluster,
) (*kdv1.KubeDirectorApp, error) {

	if cr.AppSpec != nil {
		return cr.AppSpec, nil
	}
	appCR, appErr := observer.GetApp(cr.Namespace, cr.Spec.AppNamespace, cr.Spec.AppID)
	if appErr != nil {
		return nil, fmt.Errorf(
			"failed to fetch CR for the App : %s error %v",
			cr.Spec.AppID,
			appErr,
		)
	}
	cr.AppSpec = appCR
	return appCR, nil
}
