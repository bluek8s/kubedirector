// Copyright 2021 Hewlett Packard Enterprise Development LP

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDirectorStatusBackupSpec defines the desired state of KubeDirectorStatusBackup.
// This contains a single property that mirrors the status stanza of the
// associated KubeDirectorCluster.
type KubeDirectorStatusBackupSpec struct {
	StatusBackup *KubeDirectorClusterStatus `json:"statusBackup,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorStatusBackup is the Schema for the kubedirectorstatusbackups API.
// This object represents a single virtual cluster's backed-up status.
// +kubebuilder:resource:path=kubedirectorstatusbackups,scope=Namespaced
type KubeDirectorStatusBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubeDirectorStatusBackupSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorStatusBackupList contains a list of KubeDirectorStatusBackup.
type KubeDirectorStatusBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDirectorStatusBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorStatusBackup{}, &KubeDirectorStatusBackupList{})
}
