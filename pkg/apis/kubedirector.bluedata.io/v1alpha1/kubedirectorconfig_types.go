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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDirectorConfigSpec defines the desired state of KubeDirectorConfig
// +k8s:openapi-gen=true
type KubeDirectorConfigSpec struct {
	StorageClass         *string `json:"defaultStorageClassName,omitempty"`
	ServiceType          *string `json:"defaultServiceType,omitempty"`
	NativeSystemdSupport *bool   `json:"nativeSystemdSupport"`
}

// KubeDirectorConfigStatus defines the observed state of KubeDirectorConfig.
// +k8s:openapi-gen=true
type KubeDirectorConfigStatus struct {
	GenerationUID string `json:"generationUID"`
	State         string `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfig represents single global config. This will be referenced
// by kubedirector when processing cluster CRs and app CRs.
// +k8s:openapi-gen=true
type KubeDirectorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *KubeDirectorConfigSpec   `json:"spec,omitempty"`
	Status            *KubeDirectorConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDirectorConfigList is the top-level list type for global config CRs
type KubeDirectorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDirectorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubeDirectorConfig{}, &KubeDirectorConfigList{})
}
