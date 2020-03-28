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

package validator

import (
	"encoding/json"
	"reflect"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// patchOperation is used to create the PATCH operation for the sidecar container
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (obj configPatchValue) MarshalCMJSON() ([]byte, error) {

	if obj.ValueStr != nil {
		return json.Marshal(obj.ValueStr)
	}
	if obj.ValueBool != nil {
		return json.Marshal(obj.ValueBool)
	}
	return json.Marshal(struct{}{})
}

// admitConfigMap is the top-level config validation function, which invokes
// specific validation subroutines and composes the admission response. The
// admission response will include PATCH operations as necessary to populate
// values for missing properties.
func admitConfigMap(
	ar *v1beta1.AdmissionReview,
) *v1beta1.AdmissionResponse {
	// No-op for now
	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: true,
	}

	return &admitResponse
}

func ensureChange(newMap corev1.ConfigMap) (bool, error) {
	//current configmap
	existingCM, fetcherr := observer.GetConfigMap(newMap.Namespace, newMap.Name)
	if fetcherr != nil {
		if !reflect.DeepEqual(existingCM.ResourceVersion, newMap.ResourceVersion) {
			return true, nil
		}
	} else {
		return false, fetcherr
	}
	//all good, nothing changed
	return false, nil
}

// updateAnnotation sets processConfigMap to true so that reconciller can process it
func updateAnnotation(target map[string]string) (patch []patchOperation) {

	key := "recentResourceVersion"
	val := target["recentResourceVersion"]
	if target == nil || target[key] == "" {

		patch = append(patch, patchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				key: val,
			},
		})
	} else {
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/metadata/annotations/" + key,
			Value: val,
		})
	}

	return patch
}
