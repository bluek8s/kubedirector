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
	"encoding/json"
	"strconv"
	"sync"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/core/v1"
)

// allServiceRefkeys is a subroutine of getServices, used to generate a
// description of a service's associated roles in the format expected by the
// app setup Python packages.
func allServiceRefkeys(
	roleNames []string,
	serviceName string,
) refkeysMap {

	result := make(refkeysMap)
	for _, r := range roleNames {
		result[r] = refkeys{
			BdvlibRefKey: []string{"nodegroups", "1", "roles", r, "services", serviceName},
		}
	}
	return result
}

// getServices is a subroutine of clusterBaseConfig, used to generate a
// description of all active services and their associated roles in the
// format expected by the app setup Python packages.
func getServices(
	appCR *kdv1.KubeDirectorApp,
	membersForRole map[string][]*kdv1.MemberStatus,
) map[string]ngRefkeysMap {

	result := make(map[string]ngRefkeysMap)

	for _, service := range appCR.Spec.Services {
		var activeRoleNames []string
		for _, roleService := range appCR.Spec.Config.RoleServices {
			if shared.StringInList(service.ID, roleService.ServiceIDs) {
				if _, ok := membersForRole[roleService.RoleID]; ok {
					activeRoleNames = append(activeRoleNames, roleService.RoleID)
				}
			}
		}
		if len(activeRoleNames) > 0 {
			result[service.ID] = ngRefkeysMap{
				"1": allServiceRefkeys(activeRoleNames, service.ID),
			}
		}
	}

	return result
}

// servicesForRole generates a map of service ID to internal service
// representation, for all services active in the given role.
func servicesForRole(
	appCR *kdv1.KubeDirectorApp,
	roleName string,
	members []*kdv1.MemberStatus,
) map[string]service {

	result := make(map[string]service)

	for _, roleService := range appCR.Spec.Config.RoleServices {
		if roleService.RoleID == roleName {
			for _, serviceID := range roleService.ServiceIDs {
				serviceDef := GetServiceFromID(appCR, serviceID)
				var endpoints []string
				if serviceDef.Endpoint.Port != nil {
					for _, m := range members {
						nodeName := m.Pod
						endpoint := serviceDef.Endpoint.URLScheme
						endpoint += "://" + nodeName
						endpoint += ":" + strconv.Itoa(int(*(serviceDef.Endpoint.Port)))
						endpoints = append(endpoints, endpoint)
					}
				}
				s := service{
					Qualifiers: []string{}, // currently, always empty
					Name:       serviceDef.Label.Name,
					Id:         serviceDef.ID,
					Hostnames: refkeys{
						BdvlibRefKey: []string{"nodegroups", "1", "roles", roleName, "hostnames"},
					},
					GlobalId: "1_" + roleName + "_" + serviceDef.ID,
					Fqdns: refkeys{
						BdvlibRefKey: []string{"nodegroups", "1", "roles", roleName, "fqdns"},
					},
					ExportedService: "", // currently, always empty
					Endpoints:       endpoints,
				}
				result[serviceDef.ID] = s
			}
		}
	}

	return result
}

// nodegroups generates a map of nodegroup ID to internal nodegroup
// representation. Note that KubeDirector currently only allows/manages one
// nodegroup per virtual cluster, so this will always be a map that has a
// single key of "1".
func nodegroups(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	membersForRole map[string][]*kdv1.MemberStatus,
	domain string,
) map[string]nodegroup {

	roles := make(map[string]role)
	for _, roleSpec := range cr.Spec.Roles {
		roleName := roleSpec.Name
		members := membersForRole[roleName]

		var fqdns []string
		var nodeIds []string
		fqdnMappings := make(map[string]string)
		for _, m := range members {
			nodeName := m.Pod
			// ConfigCli expects this to be a string.
			nodeIdStr := strconv.FormatInt(m.NodeId, 10)

			f := nodeName + "." + domain
			fqdnMappings[f] = nodeIdStr

			fqdns = append(fqdns, f)
			nodeIds = append(nodeIds, nodeIdStr)
		}
		memoryQuant := roleSpec.Resources.Limits[v1.ResourceMemory]
		memoryMb := memoryQuant.Value() / (1024 * 1024)
		coresQuant := roleSpec.Resources.Limits[v1.ResourceCPU]
		roleFlavor := flavor{
			Storage:     "n/a",
			Name:        "n/a",
			Memory:      strconv.FormatInt(memoryMb, 10),
			Description: "n/a",
			Cores:       strconv.FormatInt(coresQuant.Value(), 10), // rounds up
		}
		roles[roleName] = role{
			Services:     servicesForRole(appCR, roleName, members),
			NodeIds:      nodeIds,
			Hostnames:    fqdns,
			Fqdns:        fqdns,
			FqdnMappings: fqdnMappings,
			Flavor:       roleFlavor,
		}
	}
	return map[string]nodegroup{
		"1": nodegroup{
			Roles:               roles,
			DistroId:            appCR.Spec.DistroID,
			CatalogEntryVersion: appCR.Spec.Version,
			ConfigMeta:          appCR.Spec.Config.ConfigMetadata,
		},
	}
}

// clusterBaseConfig generates the portion of the config metadata that does
// not vary from member to member.
func clusterBaseConfig(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	membersForRole map[string][]*kdv1.MemberStatus,
	domain string,
) *configmeta {

	return &configmeta{
		Version:    strconv.Itoa(appCR.Spec.JSONSetupPackage.SetupPackage.ConfigAPIVersion),
		Services:   getServices(appCR, membersForRole),
		Nodegroups: nodegroups(cr, appCR, membersForRole, domain),
		Distros: map[string]refkeysMap{
			appCR.Spec.DistroID: refkeysMap{
				"1": refkeys{
					BdvlibRefKey: []string{"nodegroups", "1"},
				},
			},
		},
		Cluster: cluster{
			Name:     cr.Name,
			Isolated: false, // currently, always false
			Id:       string(cr.UID),
			ConfigMeta: map[string]refkeys{
				"1": refkeys{
					BdvlibRefKey: []string{"nodegroups", "1", "config_metadata"},
				},
			},
		},
	}
}

// ConfigmetaGenerator returns a function that generates metadata which will be
// consumed by the app setup Python packages inside a specific cluster member.
// This metadata is prepared based on the app type definition that is
// referenced in the virtual cluster spec.
func ConfigmetaGenerator(
	cr *kdv1.KubeDirectorCluster,
	membersForRole map[string][]*kdv1.MemberStatus,
) (func(string) string, error) {

	// Fetch the app type definition if we haven't yet cached it in this
	// handler pass.
	appCR, err := GetApp(cr)
	if err != nil {
		return nil, err
	}

	// It's tempting to do this part of the metadata creation lazily in the
	// returned function, since we won't always actually need to call the
	// function. However it's really handy to know up front if any errors
	// would be generated.
	domain := cr.Status.ClusterService + "." + cr.Namespace + shared.DomainBase
	perNodeConfig := make(map[string]*node)
	c := clusterBaseConfig(cr, appCR, membersForRole, domain)
	for roleName, members := range membersForRole {
		for _, member := range members {
			memberName := member.Pod
			perNodeConfig[memberName] = &node{
				RoleId:      roleName,
				NodegroupId: "1",
				Id:          strconv.FormatInt(member.NodeId, 10),
				Hostname:    memberName + "." + domain,
				Fqdn:        memberName + "." + domain,
				Domain:      domain,
				DistroId:    appCR.Spec.DistroID,
				DependsOn:   make(refkeysMap), // currently, always empty
			}
		}
	}

	var mux sync.Mutex

	return func(n string) string {
		mux.Lock()
		c.Node = perNodeConfig[n]
		jsonConfig, _ := json.Marshal(c)
		mux.Unlock()
		return string(jsonConfig)
	}, nil
}
