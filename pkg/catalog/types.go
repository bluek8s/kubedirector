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

// configmeta is a representation of a virtual cluster config, based on both
// the app type definition and the deploy-time spec provided in the cluster
// CR. It is arranged in a format to be consumed by the app setup Python
// packages.
type configmeta struct {
	Version     string                  `json:"version"`
	Services    map[string]ngRefkeysMap `json:"services"`
	Nodegroups  map[string]nodegroup    `json:"nodegroups"`
	Distros     map[string]refkeysMap   `json:"distros"`
	Cluster     cluster                 `json:"cluster"`
	Node        *node                   `json:"node"`
	Connections connections             `json:"connections"`
}

type ngRefkeysMap map[string]refkeysMap

type refkeysMap map[string]refkeys

type refkeys struct {
	BdvlibRefKey []string `json:"bdvlibrefkey"`
}

type nodegroup struct {
	Roles               map[string]role   `json:"roles"`
	DistroID            string            `json:"distro_id"`
	CatalogEntryVersion string            `json:"catalog_entry_version"`
	ConfigMeta          map[string]string `json:"config_metadata"`
}

type connections struct {
	Clusters   map[string]configmeta                     `json:"clusters"`
	ConfigMaps map[string][]map[string]map[string]string `json:"configmaps"`
	Secrets    map[string][]map[string]map[string][]byte `json:"secrets"`
}

type cluster struct {
	Name       string             `json:"name"`
	Isolated   bool               `json:"isolated"`
	ID         string             `json:"id"`
	ConfigMeta map[string]refkeys `json:"config_metadata"`
}

type node struct {
	RoleID      string     `json:"role_id"`
	NodegroupID string     `json:"nodegroup_id"`
	ID          string     `json:"id"`
	Hostname    string     `json:"hostname"`
	FQDN        string     `json:"fqdn"`
	Domain      string     `json:"domain"`
	DistroID    string     `json:"distro_id"`
	DependsOn   refkeysMap `json:"depends_on"`
}

type role struct {
	Services     map[string]service `json:"services"`
	NodeIDs      []string           `json:"node_ids"`
	Hostnames    []string           `json:"hostnames"`
	FQDNs        []string           `json:"fqdns"`
	FQDNMappings map[string]string  `json:"fqdn_mappings"`
	Flavor       flavor             `json:"flavor"`
}

type service struct {
	Qualifiers      []string `json:"qualifiers"`
	Name            string   `json:"name"`
	ID              string   `json:"id"`
	Hostnames       refkeys  `json:"hostnames"`
	GlobalID        string   `json:"global_id"`
	FQDNs           refkeys  `json:"fqdns"`
	ExportedService string   `json:"exported_service"`
	Endpoints       []string `json:"endpoints"`
	AuthToken       string   `json:"authToken"`
}

type flavor struct {
	Storage     string `json:"storage"`
	Name        string `json:"name"`
	Memory      string `json:"memory"`
	Description string `json:"description"`
	Cores       string `json:"cores"`
}

// ServicePortInfo - A mapping between a Service Port ID and the port number
type ServicePortInfo struct {
	ID        string
	Port      int32
	URLScheme string
}
