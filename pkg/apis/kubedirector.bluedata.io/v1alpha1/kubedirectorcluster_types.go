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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDirectorClusterSpec defines the desired state of KubeDirectorCluster.
// AppID references a KubeDirectorApp CR. ServiceType indicates whether to
// use NodePort or LoadBalancer services. The Roles field describes the
// requested cluster roles, each of which will be implemented (by KubeDirector)
// using a StatefulSet.
// +k8s:openapi-gen=true
type KubeDirectorClusterSpec struct {
	AppID         string    `json:"app"`
	AppCatalog    *string   `json:"appCatalog"`
	ServiceType   *string   `json:"serviceType"`
	Roles         []Role    `json:"roles"`
	DefaultSecret *KDSecret `json:"defaultSecret"`
}

// KubeDirectorClusterStatus defines the observed state of KubeDirectorCluster.
// It identifies which native k8s objects make up the cluster, and broadly
// indicates ongoing operations of cluster creation or reconfiguration.
// +k8s:openapi-gen=true
type KubeDirectorClusterStatus struct {
	State          string       `json:"state"`
	GenerationUID  string       `json:"generationUID"`
	ClusterService string       `json:"clusterService"`
	LastNodeID     int64        `json:"lastNodeID"`
	Roles          []RoleStatus `json:"roles"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorCluster represents a single virtual cluster. This cluster
// will be implemented by KubeDirector using native k8s objects. Note that
// the AppSpec field is only used internally in KubeDirector; that field is
// not persisted to k8s.
// +k8s:openapi-gen=true
type KubeDirectorCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    KubeDirectorClusterSpec    `json:"spec,omitempty"`
	Status  *KubeDirectorClusterStatus `json:"status,omitempty"`
	AppSpec *KubeDirectorApp           `json:"-"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorClusterList is the top-level list type for virtual cluster CRs.
type KubeDirectorClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDirectorCluster `json:"items"`
}

// KDSecret describes a secret object intended to be mounted inside a container.
type KDSecret struct {
	Name        string `json:"name"`
	DefaultMode *int32 `json:"defaultMode,omitempty"`
	MountPath   string `json:"mountPath"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
}

// EnvVar specifies environment variables for the start script in a container
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FilePermissions specifies file mode along with user/group
// information for the file
type FilePermissions struct {
	FileMode  *string `json:"fileMode,omitempty"`
	FileOwner *string `json:"fileOwner,omitempty"`
	FileGroup *string `json:"fileGroup,omitempty"`
}

// FileInjections specifies file injection spec, including
// file permissions on the destination file
type FileInjections struct {
	SrcURL      string           `json:"srcURL"`
	DestDir     string           `json:"destDir"`
	Permissions *FilePermissions `json:"permissions,omitempty"`
}

// Role describes a subset of the virtual cluster members that shares a common
// image, resource requirements, persistent storage definition, and (as
// defined by the cluster's KubeDirectorApp) set of service endpoints.
type Role struct {
	Name           string                      `json:"id"`
	Members        *int32                      `json:"members"`
	Resources      corev1.ResourceRequirements `json:"resources"`
	Storage        *ClusterStorage             `json:"storage,omitempty"`
	EnvVars        []corev1.EnvVar             `json:"env,omitempty"`
	FileInjections []FileInjections            `json:"fileInjections,omitempty"`
	Secret         *KDSecret                   `json:"secret"`
}

// ClusterStorage defines the persistent storage size/type, if any, to be used
// for certain specified directories of each container filesystem in a role.
type ClusterStorage struct {
	Size         string  `json:"size"`
	StorageClass *string `json:"storageClassName"`
}

// RoleStatus describes the component objects of a virtual cluster role.
type RoleStatus struct {
	Name        string         `json:"id"`
	StatefulSet string         `json:"statefulSet"`
	Members     []MemberStatus `json:"members"`
}

// MemberStatus describes the component objects of a virtual cluster member.
type MemberStatus struct {
	Pod     string `json:"pod"`
	Service string `json:"service"`
	PVC     string `json:"pvc,omitempty"`
	State   string `json:"state"`
	NodeID  int64  `json:"nodeID"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorCluster{}, &KubeDirectorClusterList{})
}
