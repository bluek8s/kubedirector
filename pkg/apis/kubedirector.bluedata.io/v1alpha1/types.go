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

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorClusterList is the top-level list type for virtual cluster CRs.
type KubeDirectorClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KubeDirectorCluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorCluster represents a single virtual cluster. This cluster
// will be implemented by KubeDirector using native k8s objects. Note that
// the AppSpec field is only used internally in KubeDirector; that field is
// not persisted to k8s.
type KubeDirectorCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec      `json:"spec"`
	Status            *ClusterStatus   `json:"status,omitempty"`
	AppSpec           *KubeDirectorApp `json:"-"`
}

// ClusterSpec is the spec provided for a virtual cluster. AppID references
// a KubeDirectorApp CR. ServiceType indicates whether to use NodePort or
// LoadBalancer services. The Roles field describes the requested cluster roles,
// each of which will be implemented (by KubeDirector) using a StatefulSet.
type ClusterSpec struct {
	AppID       string  `json:"app"`
	ServiceType *string `json:"serviceType"`
	Roles       []Role  `json:"roles"`
}

// EnvVar specifies environment variables for the start script in a
// container
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Role describes a subset of the virtual cluster members that shares a common
// image, resource requirements, persistent storage definition, and (as
// defined by the cluster's KubeDirectorApp) set of service endpoints.
type Role struct {
	Name      string                  `json:"id"`
	Members   *int32                  `json:"members"`
	Resources v1.ResourceRequirements `json:"resources"`
	Storage   ClusterStorage          `json:"storage,omitempty"`
	EnvVars   []v1.EnvVar             `json:"env,omitempty"`
}

// ClusterStorage defines the persistent storage size/type, if any, to be used
// for certain specified directories of each container filesystem in a role.
type ClusterStorage struct {
	Size         string  `json:"size"`
	StorageClass *string `json:"storageClassName"`
}

// ClusterStatus is the overall status object for a virtual cluster. It
// identifies which native k8s objects make up the cluster, and broadly
// indicates ongoing operations of cluster creation or reconfiguration.
type ClusterStatus struct {
	State          string       `json:"state"`
	GenerationUid  string       `json:"generation_uid"`
	ClusterService string       `json:"cluster_service"`
	LastNodeId     int64        `json:"last_node_id"`
	Roles          []RoleStatus `json:"roles"`
}

// RoleStatus describes the component objects of a virtual cluster role.
type RoleStatus struct {
	Name        string         `json:"id"`
	StatefulSet string         `json:"stateful_set"`
	Members     []MemberStatus `json:"members"`
}

// MemberStatus describes the component objects of a virtual cluster member.
type MemberStatus struct {
	Pod     string `json:"pod"`
	Service string `json:"service"`
	PVC     string `json:"pvc,omitempty"`
	State   string `json:"state"`
	NodeId  int64  `json:"node_id"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorAppList is the top-level list type for app definition CRs.
type KubeDirectorAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KubeDirectorApp `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorApp represents a single app definition. This will be
// referenced by KubeDirectorCluster CRs, and then used by KupeDirector to
// determine how to deploy and manage the virtual cluster.
type KubeDirectorApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppSpec `json:"spec"`
}

// AppSpec is the spec provided for an app definition.
type AppSpec struct {
	Label           Label           `json:"label"`
	DistroID        string          `json:"distro_id"`
	Version         string          `json:"version"`
	SchemaVersion   int             `json:"schema_version"`
	Image           Image           `json:"image_repo_tag,omitempty"`
	SetupPackage    SetupPackage    `json:"config_package,omitempty"`
	Services        []Service       `json:"services"`
	NodeRoles       []NodeRole      `json:"roles"`
	Config          NodeGroupConfig `json:"config"`
	PersistDirs     *[]string       `json:"persist_dirs"`
	Capabilities    []v1.Capability `json:"capabilities"`
	SystemdRequired bool            `json:"systemdRequired"`
}

// Label is a short name and long description for the app definition.
type Label struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Image is the Docker container image to be used. A top-level image can be
// specified, and/or a role-specific image that will override any top-level
// image.
type Image struct {
	IsSet   bool
	IsNull  bool
	RepoTag string
}

// SetupPackage describes the app setup package to be used. A top-level
// package can be specified, and/or a role-specific package that will override
// any top-level package.
type SetupPackage struct {
	IsSet      bool
	IsNull     bool
	PackageURL SetupPacakgeURL
}

// SetupPacakgeURL is the URL of the setup package.
type SetupPacakgeURL struct {
	PackageURL string `json:"package_url"`
}

// Service describes a network endpoint that should be exposed for external
// access, and/or identified for other use by API clients or consumers
// internal to the virtual cluster (e.g. app setup packages).
type Service struct {
	ID       string          `json:"id"`
	Label    Label           `json:"label,omitempty"`
	Endpoint ServiceEndpoint `json:"endpoint,omitempty"`
}

// ServiceEndpoint describes the service network address and protocol, and
// whether it should be displayed through a web browser.
type ServiceEndpoint struct {
	URLScheme   string `json:"url_scheme,omitempty"`
	Port        *int32 `json:"port"`
	Path        string `json:"path,omitempty"`
	IsDashboard bool   `json:"is_dashboard,omitempty"`
}

// NodeRole describes a subset of virtual cluster members that will provide
// the same services. At deployment time all role members will receive
// identical resource assignments.
type NodeRole struct {
	ID           string       `json:"id"`
	Cardinality  string       `json:"cardinality"`
	Image        Image        `json:"image_repo_tag,omitempty"`
	SetupPackage SetupPackage `json:"config_package,omitempty"`
	PersistDirs  *[]string    `json:"persist_dirs"`
}

// NodeGroupConfig identifies a set of roles, and the services on those roles.
// The top-level config indicates which roles and services will always be
// active. Implementation of "config choices" will introduce other conditional
// configs.
type NodeGroupConfig struct {
	RoleServices   []RoleService     `json:"role_services"`
	SelectedRoles  []string          `json:"selected_roles"`
	ConfigMetadata map[string]string `json:"config_meta"`
}

// RoleService associates a service with a role.
type RoleService struct {
	ServiceIDs []string `json:"service_ids"`
	RoleID     string   `json:"role_id"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfigList is the top-level list type for global config CRs
type KubeDirectorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KubeDirectorConfig `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfig represents single global config. This will be referenced
// by kubediector when processing cluster CRs and app CRs.
type KubeDirectorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ConfigSpec `json:"spec"`
}

// ConfigSpec is the spec provided for an app definition.
type ConfigSpec struct {
	StorageClass         *string `json:"defaultStorageClassName,omitempty"`
	ServiceType          *string `json:"defaultServiceType,omitempty"`
	NativeSystemdSupport bool    `json:"nativeSystemdSupport"`
}
