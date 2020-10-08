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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDirectorAppSpec defines the desired state of KubeDirectorApp.
type KubeDirectorAppSpec struct {
	Label               Label               `json:"label"`
	DistroID            string              `json:"distroID"`
	Version             string              `json:"version"`
	SchemaVersion       int                 `json:"configSchemaVersion"`
	DefaultImageRepoTag *string             `json:"defaultImageRepoTag,omitempty"`
	DefaultSetupPackage SetupPackage        `json:"defaultConfigPackage,omitempty"`
	Services            []Service           `json:"services,omitempty"`
	NodeRoles           []NodeRole          `json:"roles"`
	Config              NodeGroupConfig     `json:"config"`
	DefaultPersistDirs  *[]string           `json:"defaultPersistDirs,omitempty"`
	DefaultEventList    *[]string           `json:"defaultEventList,omitempty"`
	Capabilities        []corev1.Capability `json:"capabilities,omitempty"`
	SystemdRequired     bool                `json:"systemdRequired,omitempty"`
	LogoURL             string              `json:"logoURL,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorApp is the Schema for the kubedirectorapps API.
// +kubebuilder:resource:path=kubedirectorapps,scope=Namespaced
type KubeDirectorApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubeDirectorAppSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorAppList contains a list of KubeDirectorApp.
type KubeDirectorAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDirectorApp `json:"items"`
}

// Label is a short name and long description for the app definition.
type Label struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SetupPackage describes the app setup package to be used. A top-level
// package can be specified, and/or a role-specific package that will override
// any top-level package. Note that there is custom deserialization code for
// this type in decode.go to allow us to distinguish the cases of "unset" from
// "explicitly set null". Therefore "operator-sdk generate crds" cannot be
// used to generate a correct CRD in this case.
type SetupPackage struct {
	IsSet      bool
	IsNull     bool
	PackageURL SetupPackageURL
}

// SetupPackageURL is the URL of the setup package.
type SetupPackageURL struct {
	PackageURL string `json:"packageURL"`
}

// Service describes a network endpoint that should be exposed for external
// access, and/or identified for other use by API clients or consumers
// internal to the virtual cluster (e.g. app setup packages).
type Service struct {
	ID              string          `json:"id"`
	Label           Label           `json:"label,omitempty"`
	Endpoint        ServiceEndpoint `json:"endpoint,omitempty"`
	ExportedService string          `json:"exported_service,omitempty"`
}

// ServiceEndpoint describes the service network address and protocol, and
// whether it should be displayed through a web browser.
type ServiceEndpoint struct {
	URLScheme    string `json:"urlScheme,omitempty"`
	Port         *int32 `json:"port"`
	Path         string `json:"path,omitempty"`
	IsDashboard  bool   `json:"isDashboard,omitempty"`
	HasAuthToken bool   `json:"hasAuthToken,omitempty"`
}

// NodeRole describes a subset of virtual cluster members that will provide
// the same services. At deployment time all role members will receive
// identical resource assignments.
type NodeRole struct {
	ID           string               `json:"id"`
	Cardinality  string               `json:"cardinality"`
	ImageRepoTag *string              `json:"imageRepoTag,omitempty"`
	SetupPackage SetupPackage         `json:"configPackage,omitempty"`
	PersistDirs  *[]string            `json:"persistDirs,omitempty"`
	EventList    *[]string            `json:"eventList,omitempty"`
	MinResources *corev1.ResourceList `json:"minResources,omitempty"`
}

// NodeGroupConfig identifies a set of roles, and the services on those roles.
// The top-level config indicates which roles and services will always be
// active. Implementation of "config choices" will introduce other conditional
// configs.
type NodeGroupConfig struct {
	RoleServices   []RoleService     `json:"roleServices"`
	SelectedRoles  []string          `json:"selectedRoles"`
	ConfigMetadata map[string]string `json:"configMeta,omitempty"`
}

// RoleService associates a service with a role.
type RoleService struct {
	ServiceIDs []string `json:"serviceIDs"`
	RoleID     string   `json:"roleID"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorApp{}, &KubeDirectorAppList{})
}
