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

package catalog

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	// ConfigMapType is a label placed on desired comfig maps that
	// we want to watch and propogate inside containers
	configMapType = shared.KdDomainBase + "/cmType"
	// ServiceTokenAnnotation auth token for service
	serviceAuthToken = shared.KdDomainBase + "/kd-auth-token"
	// SecretType is a label placed on desired secret that
	// we want to watch and propogate inside containers
	secretType = shared.KdDomainBase + "/secretType"
)

// allServiceRefkeys is a subroutine of getServices, used to generate a
// description of a service's associated roles in the format expected by the
// app setup Python packages.
func allServiceRefkeys(
	roleNames []string,
	serviceName string,
	connectedClusterName string,
) refkeysMap {

	result := make(refkeysMap)
	for _, r := range roleNames {
		var refKeyList []string
		if connectedClusterName != "" {
			refKeyList = []string{"connections", "clusters", connectedClusterName}
		}
		refKeyList = append(refKeyList, "nodegroups", "1", "roles", r, "services", serviceName)
		result[r] = refkeys{
			BdvlibRefKey: refKeyList,
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
	connectedClusterName string,
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
				"1": allServiceRefkeys(activeRoleNames, service.ID, connectedClusterName),
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
	connectedClusterName string,
	domain string,
) map[string]service {

	result := make(map[string]service)
	for _, roleService := range appCR.Spec.Config.RoleServices {
		if roleService.RoleID == roleName {
			for _, serviceID := range roleService.ServiceIDs {
				var serviceToken string
				serviceDef := GetServiceFromID(appCR, serviceID)
				var endpoints []string
				if serviceDef.Endpoint.Port != nil {
					for _, m := range members {
						nodeName := m.Pod
						endpoint := serviceDef.Endpoint.URLScheme
						endpoint += "://" + nodeName + "." + domain
						endpoint += ":" + strconv.Itoa(int(*(serviceDef.Endpoint.Port)))
						endpoints = append(endpoints, endpoint)
						if serviceDef.Endpoint.HasAuthToken {
							if len(m.AuthToken) == 0 {
								checksum := md5.Sum([]byte(uuid.New().String()))
								m.AuthToken = hex.EncodeToString(checksum[:])
							}
							serviceToken = m.AuthToken
							wait := time.Second
							maxWait := 4096 * time.Second
							for {
								if wait > maxWait {
									break
								}
								k8sService, err := observer.GetService(appCR.Namespace, m.Service)
								if err == nil {
									k8sService.Annotations[serviceAuthToken] = serviceToken
									if shared.Update(context.TODO(), k8sService) == nil {
										break
									}
								}
								time.Sleep(wait)
								wait = wait * 2
							}
						}
					}
				}
				s := service{
					Qualifiers: []string{}, // currently, always empty
					Name:       serviceDef.Label.Name,
					ID:         serviceDef.ID,
					Hostnames: refkeys{
						BdvlibRefKey: []string{"nodegroups", "1", "roles", roleName, "hostnames"},
					},
					GlobalID: "1_" + roleName + "_" + serviceDef.ID,
					FQDNs: refkeys{
						BdvlibRefKey: []string{"nodegroups", "1", "roles", roleName, "fqdns"},
					},
					ExportedService: serviceDef.ExportedService,
					Endpoints:       endpoints,
					AuthToken:       serviceToken,
				}
				if connectedClusterName != "" {
					s.Hostnames.BdvlibRefKey = append(
						[]string{"connections", "clusters", connectedClusterName},
						s.Hostnames.BdvlibRefKey...,
					)
					s.FQDNs.BdvlibRefKey = append(
						[]string{"connections", "clusters", connectedClusterName},
						s.FQDNs.BdvlibRefKey...,
					)
				}
				result[serviceDef.ID] = s
			}
		}
	}

	return result
}

// genConfigConnections will look at the cluster spec
// and generates a map of configmap type and corresponding
// configmaps to be connected to the given cluster
func genConfigConnections(
	cr *kdv1.KubeDirectorCluster,
) (map[string][]map[string]map[string]string, error) {

	// Many connected configmaps can be of a given type, hence
	// create a map of cmType:list of configmaps, where
	// every configmap is a map of string and string
	kdcm := make(map[string][]map[string]map[string]string)
	for _, connectedCmName := range cr.Spec.Connections.ConfigMaps {
		cm, err := observer.GetConfigMap(cr.Namespace, connectedCmName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			} else {
				return nil, err
			}
		}
		if kdConfigMapType, ok := cm.Labels[configMapType]; ok {
			cmMap := make(map[string]map[string]string)
			cmMap["metadata"] = map[string]string{"name": cm.Name}
			cmMap["data"] = cm.Data
			cmMap["labels"] = cm.Labels
			cmMap["annotations"] = cm.Annotations
			if mapList, ok := kdcm[kdConfigMapType]; ok {
				kdcm[kdConfigMapType] = append(mapList, cmMap)
			} else {
				typeMaps := make([]map[string]map[string]string, 0)
				kdcm[kdConfigMapType] = append(typeMaps, cmMap)
			}
		}
	}
	return kdcm, nil
}

// genSecretConnections will look at the cluster spec
// and generates a map of secret type and corresponding
// secrets to be connected to the given cluster
func genSecretConnections(
	cr *kdv1.KubeDirectorCluster,
) (map[string][]map[string]map[string][]byte, error) {

	// Many connected secrets can be of a given type, hence
	// create a map of secretType:list of secrets, where
	// every secret is a map of string and byte array
	kdsecret := make(map[string][]map[string]map[string][]byte)
	for _, connectedsecretName := range cr.Spec.Connections.Secrets {
		sec, err := observer.GetSecret(cr.Namespace, connectedsecretName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			} else {
				return nil, err
			}
		}
		if kdSecretType, ok := sec.Labels[secretType]; ok {
			xlateMap := func(valueMap map[string]string) map[string][]byte {
				convMap := make(map[string][]byte)
				for k, v := range valueMap {
					convMap[k] = []byte(v)
				}
				return convMap
			}
			secretMap := make(map[string]map[string][]byte)
			secretMap["metadata"] = map[string][]byte{"name": []byte(sec.Name)}
			secretMap["data"] = sec.Data
			secretMap["labels"] = xlateMap(sec.Labels)
			secretMap["annotations"] = xlateMap(sec.Annotations)
			if secretList, ok := kdsecret[kdSecretType]; ok {
				kdsecret[kdSecretType] = append(secretList, secretMap)
			} else {
				typeSecrets := make([]map[string]map[string][]byte, 0)
				kdsecret[kdSecretType] = append(typeSecrets, secretMap)
			}
		}
	}
	return kdsecret, nil
}

// genClusterConnections generates a map of running clusters that are to be connected
// to this cluster
func genClusterConnections(
	cr *kdv1.KubeDirectorCluster,
) (map[string]configmeta, error) {

	toConnectMeta := make(map[string]configmeta)
	for _, clusterName := range cr.Spec.Connections.Clusters {
		// Fetch the cluster object
		clusterToConnect, connectedErr := observer.GetCluster(cr.Namespace, clusterName)
		if connectedErr != nil {
			if errors.IsNotFound(connectedErr) {
				continue
			} else {
				return nil, connectedErr
			}
		}
		appForclusterToConnect, connectedAppErr := observer.GetApp(clusterToConnect.Namespace, clusterToConnect.Spec.AppID)
		if connectedAppErr != nil {
			if errors.IsNotFound(connectedAppErr) {
				continue
			} else {
				return nil, connectedAppErr
			}
		}
		domain := clusterToConnect.Status.ClusterService + "." + clusterToConnect.Namespace + shared.GetSvcClusterDomainBase()
		membersForRole := make(map[string][]*kdv1.MemberStatus)
		for _, roleInfo := range clusterToConnect.Status.Roles {
			var membersStatus []*kdv1.MemberStatus
			for _, members := range roleInfo.Members {
				membersStatus = append(
					membersStatus,
					&members,
				)
			}
			membersForRole[roleInfo.Name] = membersStatus
		}

		toConnectMeta[clusterName] = configmeta{
			Version:    strconv.Itoa(appForclusterToConnect.Spec.SchemaVersion),
			Services:   getServices(appForclusterToConnect, membersForRole, clusterName),
			Nodegroups: nodegroups(clusterToConnect, appForclusterToConnect, membersForRole, domain),
			Distros: map[string]refkeysMap{
				appForclusterToConnect.Spec.DistroID: refkeysMap{
					"1": refkeys{
						BdvlibRefKey: []string{"connections", "clusters", clusterName, "nodegroups", "1"},
					},
				},
			},
			Cluster: cluster{
				Name:     clusterName,
				Isolated: false, // currently, always false
				ID:       string(clusterToConnect.UID),
				ConfigMeta: map[string]refkeys{
					"1": refkeys{
						BdvlibRefKey: []string{"nodegroups", "1", "config_metadata"},
					},
				},
			},
		}
	}

	return toConnectMeta, nil
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

		if members == nil {
			continue
		}

		var fqdns []string
		var nodeIds []string
		fqdnMappings := make(map[string]string)
		for _, m := range members {
			nodeName := m.Pod
			// ConfigCli expects this to be a string.
			nodeIDStr := strconv.FormatInt(m.NodeID, 10)

			f := nodeName + "." + domain
			fqdnMappings[f] = nodeIDStr

			fqdns = append(fqdns, f)
			nodeIds = append(nodeIds, nodeIDStr)
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
			Services:     servicesForRole(appCR, roleName, members, "", domain),
			NodeIDs:      nodeIds,
			Hostnames:    fqdns,
			FQDNs:        fqdns,
			FQDNMappings: fqdnMappings,
			Flavor:       roleFlavor,
		}
	}
	return map[string]nodegroup{
		"1": nodegroup{
			Roles:               roles,
			DistroID:            appCR.Spec.DistroID,
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
) (*configmeta, error) {

	clustersMeta, conErr := genClusterConnections(cr)
	if conErr != nil {
		return nil, conErr
	}
	kdConfigMaps, cmErr := genConfigConnections(cr)
	if cmErr != nil {
		return nil, cmErr
	}

	kdSecrets, secErr := genSecretConnections(cr)
	if secErr != nil {
		return nil, secErr
	}

	return &configmeta{
		Version:    strconv.Itoa(appCR.Spec.SchemaVersion),
		Services:   getServices(appCR, membersForRole, ""),
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
			ID:       string(cr.UID),
			ConfigMeta: map[string]refkeys{
				"1": refkeys{
					BdvlibRefKey: []string{"nodegroups", "1", "config_metadata"},
				},
			},
		},
		Connections: connections{
			Clusters:   clustersMeta,
			ConfigMaps: kdConfigMaps,
			Secrets:    kdSecrets,
		},
	}, nil
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
	domain := cr.Status.ClusterService + "." + cr.Namespace + shared.GetSvcClusterDomainBase()
	perNodeConfig := make(map[string]*node)
	c, err := clusterBaseConfig(cr, appCR, membersForRole, domain)
	if err != nil {
		return nil, err
	}
	for roleName, members := range membersForRole {
		for _, member := range members {
			memberName := member.Pod
			perNodeConfig[memberName] = &node{
				RoleID:      roleName,
				NodegroupID: "1",
				ID:          strconv.FormatInt(member.NodeID, 10),
				Hostname:    memberName + "." + domain,
				FQDN:        memberName + "." + domain,
				Domain:      domain,
				DistroID:    appCR.Spec.DistroID,
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
