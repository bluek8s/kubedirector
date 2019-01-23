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

package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// configPatchSpec is used to create the PATCH operation for populating
// default values in the config as necessary.
type configPatchSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// validateConfigStorageClass validates storageClassName by checking
// for a valid storageClass k8s resource.
func validateConfigStorageClass(
	storageClassName *string,
	valErrors []string,
) []string {

	if storageClassName == nil {
		return valErrors
	}

	_, err := observer.GetStorageClass(*storageClassName)

	if err == nil {
		return valErrors
	}

	valErrors = append(
		valErrors,
		fmt.Sprintf(
			invalidStorageClass,
			*storageClassName,
		),
	)

	return valErrors
}

// admitKDConfigCR is the top-level config validation function, which invokes
// specific validation subroutines and composes the admission response. The
// admission response will include PATCH operations as necessary to populate
// values for missing properties.
func admitKDConfigCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var valErrors []string
	var patches []configPatchSpec

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	raw := ar.Request.Object.Raw
	configCR := kdv1.KubeDirectorConfig{}

	// For a delete operation, we're done now.
	if ar.Request.Operation == v1beta1.Delete {
		admitResponse.Allowed = true
		return &admitResponse
	}

	if jsonErr := json.Unmarshal(raw, &configCR); jsonErr != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + jsonErr.Error(),
		}
		return &admitResponse
	}

	// Validate storage class name if present. If not, see if we can populate
	// it with a default value.
	storageClass := configCR.Spec.StorageClass
	if storageClass != nil {
		valErrors = validateConfigStorageClass(storageClass, valErrors)
	} else {
		storageClassList, scErr := observer.GetStorageClassList()
		if scErr != nil {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + scErr.Error(),
			}
			return &admitResponse
		}
		if len(storageClassList) != 0 {
			// XXX Currently taking the first one; actually should cycle
			// through and look for IsDefaultClass? Or should we leave it
			// undefined here and instead look for the default class each time
			// a cluster is created without specifying the classname?
			patches = append(
				patches,
				configPatchSpec{
					Op:    "add",
					Path:  "/spec/defaultStorageClassName",
					Value: storageClassList[0].Name,
				},
			)
		}
	}

	if len(valErrors) == 0 {
		if len(patches) != 0 {
			patchResult, patchErr := json.Marshal(patches)
			if patchErr == nil {
				admitResponse.Patch = patchResult
				patchType := v1beta1.PatchTypeJSONPatch
				admitResponse.PatchType = &patchType
			} else {
				valErrors = append(valErrors, failedToPatch)
			}
		}
	}

	if len(valErrors) == 0 {
		admitResponse.Allowed = true
	} else {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + strings.Join(valErrors, "\n"),
		}
	}

	return &admitResponse
}
