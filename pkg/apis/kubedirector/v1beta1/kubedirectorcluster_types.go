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

const (
	// UID represents the old naming scheme where object names were generated
	// with unique UID extensions.
	UID string = "UID"

	// CrNameRole represents the new naming scheme based on cluster name and
	// respective role name.
	CrNameRole string = "CrNameRole"
)

// KubeDirectorClusterSpec defines the desired state of KubeDirectorCluster.
// AppID references a KubeDirectorApp CR. ServiceType indicates whether to
// use NodePort or LoadBalancer services. The Roles field describes the
// requested cluster roles, each of which will be implemented (by KubeDirector)
// using a StatefulSet.
type KubeDirectorClusterSpec struct {
	AppID         string      `json:"app"`
	AppCatalog    *string     `json:"appCatalog,omitempty"`
	ServiceType   *string     `json:"serviceType,omitempty"`
	Roles         []Role      `json:"roles"`
	DefaultSecret *KDSecret   `json:"defaultSecret,omitempty"`
	Connections   Connections `json:"connections"`
	NamingScheme  *string     `json:"namingScheme,omitempty"`
}

// Connections specifies list of cluster objects and configmaps objects that has
// be connected to the cluster.
type Connections struct {
	Clusters   []string `json:"clusters,omitempty"`
	ConfigMaps []string `json:"configmaps,omitempty"`
	Secrets    []string `json:"secrets,omitempty"`
}

// KubeDirectorClusterStatus defines the observed state of KubeDirectorCluster.
// It identifies which native k8s objects make up the cluster, and broadly
// indicates ongoing operations of cluster creation or reconfiguration.
type KubeDirectorClusterStatus struct {
	State                   string           `json:"state"`
	RestoreProgress         *RestoreProgress `json:"restoreProgress,omitempty"`
	MemberStateRollup       StateRollup      `json:"memberStateRollup"`
	GenerationUID           string           `json:"generationUID"`
	SpecGenerationToProcess *int64           `json:"specGenerationToProcess,omitempty"`
	ClusterService          string           `json:"clusterService"`
	LastNodeID              int64            `json:"lastNodeID"`
	Roles                   []RoleStatus     `json:"roles"`
	LastConnectionHash      string           `json:"lastConnectionHash"`
	AppID                   *string          `json:"app"`
	UpgradeInfo             *UpgradeInfo     `json:"upgradeInfo,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorCluster is the Schema for the kubedirectorclusters API.
// This object represents a single virtual cluster. This cluster will be
// implemented by KubeDirector using native k8s objects. Note that the AppSpec
// field is only used internally in KubeDirector; that field is not persisted
// to k8s.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubedirectorclusters,scope=Namespaced
type KubeDirectorCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubeDirectorClusterSpec    `json:"spec,omitempty"`
	Status            *KubeDirectorClusterStatus `json:"status,omitempty"`
	AppSpec           *KubeDirectorApp           `json:"-"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorClusterList contains a list of KubeDirectorCluster.
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
	FileMode  *int32  `json:"fileMode,omitempty"`
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

// VolumeProjections describes an individual volume projection
// spec for mounting user created volumes to
type VolumeProjections struct {
	PvcName   string `json:"pvcName"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// Role describes a subset of the virtual cluster members that shares a common
// image, resource requirements, persistent storage definition, and (as
// defined by the cluster's KubeDirectorApp) set of service endpoints.
type Role struct {
	Name               string                      `json:"id"`
	PodLabels          map[string]string           `json:"podLabels,omitempty"`
	PodAnnotations     map[string]string           `json:"podAnnotations,omitempty"`
	ServiceLabels      map[string]string           `json:"serviceLabels,omitempty"`
	ServiceAnnotations map[string]string           `json:"serviceAnnotations,omitempty"`
	Members            *int32                      `json:"members,omitempty"`
	Resources          corev1.ResourceRequirements `json:"resources"`
	Affinity           *corev1.Affinity            `json:"affinity,omitempty"`
	Storage            *ClusterStorage             `json:"storage,omitempty"`
	EnvVars            []corev1.EnvVar             `json:"env,omitempty"`
	FileInjections     []FileInjections            `json:"fileInjections,omitempty"`
	Secret             *KDSecret                   `json:"secret,omitempty"`
	BlockStorage       *BlockStorage               `json:"blockStorage,omitempty"`
	ServiceAccountName string                      `json:"serviceAccountName,omitempty"`
	SecretKeys         []SecretKey                 `json:"secretKeys,omitempty"`
	VolumeProjections  []VolumeProjections         `json:"volumeProjections,omitempty"`
}

// SecretKey holds data which is supposed to be only available on configuration phase
type SecretKey struct {
	// Name is required and cannot be empty
	Name string `json:"name"`
	// Non-empty Value overrides EncryptedValue after encryption.
	// Empty Value is allowed but requires EncryptedValue to be empty when submitting.
	Value string `json:"value,omitempty"`
	// EncryptedValue is supposed to be generated by KD webhook
	// by encrypting Value with AES using master encryption key from KubeDirectorConfig
	EncryptedValue string `json:"encryptedValue,omitempty"`
}

// RestoreProgress identifies any necessary kdcluster components that have
// not yet been re-created by a backup restore.
type RestoreProgress struct {
	AwaitingApp       bool   `json:"awaitingApp"`
	AwaitingStatus    bool   `json:"awaitingStatus"`
	AwaitingResources bool   `json:"awaitingResources"`
	Error             string `json:"error"`
}

// StateRollup surfaces whether any per-member statuses have problems that
// should be investigated.
type StateRollup struct {
	MembershipChanging  bool `json:"membershipChanging"`
	MembersDown         bool `json:"membersDown"`
	MembersInitializing bool `json:"membersInitializing"`
	MembersWaiting      bool `json:"membersWaiting"`
	MembersRestarting   bool `json:"membersRestarting"`
	MembersUpgrading    bool `json:"membersUpgrading"`
	MembersRollingBack  bool `json:"membersRollingBack"`
	ConfigErrors        bool `json:"configErrors"`
	MembersNotScheduled bool `json:"membersNotScheduled"`
}

// ClusterStorage defines the persistent storage size/type, if any, to be used
// for certain specified directories of each container filesystem in a role.
type ClusterStorage struct {
	Size         string  `json:"size"`
	StorageClass *string `json:"storageClassName,omitempty"`
}

// BlockStorage defines the block storage type, path, and optionally size, if any, to be used
// for mounting a block volume in a role.
type BlockStorage struct {
	StorageClass *string `json:"storageClassName,omitempty"`
	Path         *string `json:"pathPrefix,omitempty"`
	Size         *string `json:"size,omitempty"`
	NumDevices   *int32  `json:"numDevices,omitempty"`
}

// RoleUpgradeStatus describes the common current upgrading state of the current role
// If at least one member of this role is being upgraded/reverted, the role
// should be in the corresponding status
type RoleUpgradeStatus string

const (
	// RoleConfigured is default empty value means the role is in ready state
	RoleConfigured RoleUpgradeStatus = ""
	// RoleUpgraded means upgrade process is finished
	RoleUpgraded RoleUpgradeStatus = "upgraded"
	// RoleRolledBack means rollback process is finished
	RoleRolledBack RoleUpgradeStatus = "rolledBack"
	// RoleUpgrading means the role is in the middle of upgrade process
	RoleUpgrading RoleUpgradeStatus = "upgrading"
	// RoleRollingBack means the role is in the middle of rollback process
	RoleRollingBack RoleUpgradeStatus = "rollingBack"
)

// RoleStatus describes the component objects of a virtual cluster role.
type RoleStatus struct {
	Name                string            `json:"id"`
	StatefulSet         string            `json:"statefulSet"`
	Members             []MemberStatus    `json:"members"`
	EncryptedSecretKeys map[string]string `json:"encryptedSecretKeys,omitempty"`
	RoleUpgradeStatus   RoleUpgradeStatus `json:"roleUpgradeStatus,omitempty"`
}

// UpgradeInfo decribes cluster upgrading status
// IsRollingBack represents if cluster is rolling back to the previous app
// PrevApp is the app name from which upgrade is being
type UpgradeInfo struct {
	IsRollingBack bool   `json:"isRollingBack"`
	PrevApp       string `json:"prevApp,omitempty"`
}

// RollbackInfo describes the last working application spec for each RoleStatus.
type RollbackInfo struct {
	AppID    string `json:"appId"`
	DistroID string `json:"distroId"`
	Version  string `json:"appVersion"`
}

// MemberUpgradeStatus describes the current upgrading state of the current pod
type MemberUpgradeStatus string

const (
	// PodConfigured is default empty value means the pod is in ready state
	PodConfigured MemberUpgradeStatus = ""
	// PodUpgrading means the pod is in the middle of upgrade process
	PodUpgrading MemberUpgradeStatus = "upgrading"
	// PodRollingBack means the pod is in the middle of rollback process
	PodRollingBack MemberUpgradeStatus = "rollingBack"
	// PodUpgraded means the pod finished upgrade process but waits for the
	// upgrade finalization on cluster level
	PodUpgraded MemberUpgradeStatus = "upgraded"
	// PodRolledBack means the pod finished the rollback process but waits for the
	// upgrade finalization on cluster level
	PodRolledBack MemberUpgradeStatus = "rolledBack"
)

// MemberStatus describes the component objects of a virtual cluster member.
type MemberStatus struct {
	Pod              string              `json:"pod"`
	Service          string              `json:"service"`
	AuthToken        string              `json:"authToken,omitempty"`
	PVC              string              `json:"pvc,omitempty"`
	State            string              `json:"state"`
	StateDetail      MemberStateDetail   `json:"stateDetail,omitempty"`
	NodeID           int64               `json:"nodeID"`
	BlockDevicePaths []string            `json:"blockDevicePaths,omitempty"`
	PodUpgradeStatus MemberUpgradeStatus `json:"podUpgradeStatus,omitempty"`
}

// MemberStateDetail digs into detail about the management of configmeta and
// app scripts in the member.
type MemberStateDetail struct {
	ConfigErrorDetail        *string             `json:"configErrorDetail,omitempty"`
	LastConfigDataGeneration *int64              `json:"lastConfigDataGeneration,omitempty"`
	LastSetupGeneration      *int64              `json:"lastSetupGeneration,omitempty"`
	ConfiguringContainer     string              `json:"configuringContainer,omitempty"`
	LastConfiguredContainer  string              `json:"lastConfiguredContainer,omitempty"`
	LastKnownContainerState  string              `json:"lastKnownContainerState,omitempty"`
	PendingNotifyCmds        []*NotificationDesc `json:"pendingNotifyCmds,omitempty"`
	LastConnectionVersion    *int64              `json:"lastConnectionVersion,omitempty"`
	StartScriptOutMsg        string              `json:"startScriptStdoutMessage,omitempty"`
	StartScriptErrMsg        string              `json:"startScriptStderrMessage,omitempty"`
	SchedulingErrorMessage   *string             `json:"schedulingErrorMessage,omitempty"`
}

// NotificationDesc contains the info necessary to perform a notify command.
type NotificationDesc struct {
	Arguments []string `json:"arguments,omitempty"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorCluster{}, &KubeDirectorClusterList{})
}
