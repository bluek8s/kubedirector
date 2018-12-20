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
	"k8s.io/api/core/v1"
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

	var repoTag string
	repoTag = ""

	// Check to see if there is a role-specific image.
	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			repoTag = nodeRole.Image.RepoTag
			break
		}
	}

	return repoTag, nil
}

// AppSetupPackageUrl returns the app setup package url for a given role. The
// fact that this function is invoked means that setup package was specified
// either for the node role or the application as a whole.
func AppSetupPackageUrl(
	cr *kdv1.KubeDirectorCluster,
	role string,
) (string, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return "", err
	}

	var appConfigURL string
	appConfigURL = ""

	// Check to see if there is a role-specific setup package.
	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			setupPackage := nodeRole.SetupPackage

			if (setupPackage.IsSet == true) && (setupPackage.IsNull == false) {
				appConfigURL = setupPackage.PackageURL
			}

			break
		}
	}

	return appConfigURL, nil
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
// has be persisted on a PVC. If the role doesn't have an explicit list, use
// the top level list.
func AppPersistDirs(
	cr *kdv1.KubeDirectorCluster,
	role string,
) (*[]string, error) {

	var appPersistDirs *[]string
	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return nil, err
	}

	// Check to see if there is a role-specific setup package.
	for _, nodeRole := range appCR.Spec.NodeRoles {
		if nodeRole.ID == role {
			appPersistDirs = nodeRole.PersistDirs
			break
		}
	}

	// If role-specific persist_dirs is not present, use the main one.
	if appPersistDirs == nil {
		appPersistDirs = cr.AppSpec.Spec.PersistDirs
	}

	return appPersistDirs, nil
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
	appCR, appErr := observer.GetApp(cr.Spec.AppID)
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
