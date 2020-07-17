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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDirectorConfigSpec defines the desired state of KubeDirectorConfig.
type KubeDirectorConfigSpec struct {
	StorageClass         *string `json:"defaultStorageClassName,omitempty"`
	ServiceType          *string `json:"defaultServiceType,omitempty"`
	NativeSystemdSupport *bool   `json:"nativeSystemdSupport,omitempty"`
	RequiredSecretPrefix *string `json:"requiredSecretPrefix,omitempty"`
	ClusterSvcDomainBase *string `json:"clusterSvcDomainBase,omitempty"`
}

// KubeDirectorConfigStatus defines the observed state of KubeDirectorConfig.
type KubeDirectorConfigStatus struct {
	GenerationUID string `json:"generationUID"`
	State         string `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfig is the Schema for the kubedirectorconfigs API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubedirectorconfigs,scope=Namespaced
type KubeDirectorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *KubeDirectorConfigSpec   `json:"spec,omitempty"`
	Status *KubeDirectorConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfigList contains a list of KubeDirectorConfig.
type KubeDirectorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDirectorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorConfig{}, &KubeDirectorConfigList{})
}
