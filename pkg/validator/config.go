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
	"fmt"
	"reflect"
	"strings"

	"github.com/bluek8s/kubedirector/pkg/controller/kubedirectorconfig"
	"github.com/bluek8s/kubedirector/pkg/shared"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// configPatchSpec is used to create the PATCH operation for populating
// default values in the config as necessary.
type configPatchSpec struct {
	Op    string           `json:"op"`
	Path  string           `json:"path"`
	Value configPatchValue `json:"value"`
}

type configPatchValue struct {
	ValueStr  *string
	ValueBool *bool
	// if no value is specified, will marshal an empty object instead
}

func (obj configPatchValue) MarshalJSON() ([]byte, error) {

	if obj.ValueStr != nil {
		return json.Marshal(obj.ValueStr)
	}
	if obj.ValueBool != nil {
		return json.Marshal(obj.ValueBool)
	}
	return json.Marshal(struct{}{})
}

// ensureConfigSpec creates a PATCH if necessary to create the top-level spec
// object. (This can be missing since none of the spec properties are
// required.) The CR's Spec property will also be set to an empty struct if
// it is currently nil.
func ensureConfigSpec(
	cr *kdv1.KubeDirectorConfig,
) []configPatchSpec {
	if cr.Spec != nil {
		return []configPatchSpec{}
	}
	cr.Spec = &kdv1.KubeDirectorConfigSpec{}
	return []configPatchSpec{
		{
			Op:   "add",
			Path: "/spec",
		},
	}
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
) *v1beta1.AdmissionResponse {

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// If this is a delete, the admission handler has nothing to do.
	if ar.Request.Operation == v1beta1.Delete {
		admitResponse.Allowed = true
		return &admitResponse
	}

	// Deserialize the object.
	raw := ar.Request.Object.Raw
	configCR := kdv1.KubeDirectorConfig{}
	if jsonErr := json.Unmarshal(raw, &configCR); jsonErr != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + jsonErr.Error(),
		}
		return &admitResponse
	}

	// Only allow KubeDirectorConfig requests in the kubedirector namespace.
	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "Failed to get kubedirector namespace",
		}
		return &admitResponse
	}
	if configCR.Namespace != kdNamespace {
		admitResponse.Result = &metav1.Status{
			Message: fmt.Sprintf("Invalid namespace '%s', must be '%s'.\n",
				configCR.Namespace,
				kdNamespace,
			),
		}
		return &admitResponse
	}

	// If this is an update, get the previous version of the object ready for
	// use in some checks.
	prevConfigCR := kdv1.KubeDirectorConfig{}
	if ar.Request.Operation == v1beta1.Update {
		prevRaw := ar.Request.OldObject.Raw
		if prevJSONErr := json.Unmarshal(prevRaw, &prevConfigCR); prevJSONErr != nil {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + prevJSONErr.Error(),
			}
			return &admitResponse
		}
	}

	// Don't allow Status to be updated except by KubeDirector. Do this by
	// using one-time codes known by KubeDirector.
	if configCR.Status != nil {
		statusViolation := &metav1.Status{
			Message: "\nKubeDirector-related status properties are read-only",
		}
		expectedStatusGen, ok := kubedirectorconfig.StatusGens.ReadStatusGen(configCR.UID)
		// Reject this write if either of:
		// - this status generation UID is not what we're expecting a write for
		// - KubeDirector doesn't know about the CR & the status is changing
		if ok {
			if configCR.Status.GenerationUID != expectedStatusGen.UID {
				admitResponse.Result = statusViolation
				return &admitResponse
			}
		} else {
			if !reflect.DeepEqual(configCR.Status, prevConfigCR.Status) {
				admitResponse.Result = statusViolation
				return &admitResponse
			}
		}
		// If this status generation UID has already been admitted previously,
		// it's OK to write the status again as long as nothing is changing.
		// (For example we'll see this when a PATCH happens.)
		if expectedStatusGen.Validated {
			if !reflect.DeepEqual(configCR.Status, prevConfigCR.Status) {
				admitResponse.Result = statusViolation
				return &admitResponse
			}
		}
	}

	kubedirectorconfig.StatusGens.ValidateStatusGen(configCR.UID)

	var valErrors []string

	patches := ensureConfigSpec(&configCR)

	// Validate storage class name if present.
	valErrors = validateConfigStorageClass(configCR.Spec.StorageClass, valErrors)

	// Populate default service type if necessary.
	if configCR.Spec.ServiceType == nil {
		patches = append(patches,
			newStrPatch("/spec/defaultServiceType", shared.DefaultServiceType),
		)
	}

	// Populate default systemd support if necessary.
	if configCR.Spec.NativeSystemdSupport == nil {
		patches = append(patches,
			newBoolPatch("/spec/nativeSystemdSupport", defaultNativeSystemd),
		)
	}

	// Populate the default ClusterSvcDomainBase if necessary
	if configCR.Spec.ClusterSvcDomainBase == nil {
		svcDomainBase := shared.DefaultSvcDomainBase
		patches = append(
			patches,
			configPatchSpec{
				Op:   "add",
				Path: "/spec/clusterSvcDomainBase",
				Value: configPatchValue{
					ValueStr: &svcDomainBase,
				},
			},
		)
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

func newBoolPatch(
	path string,
	defaultVal bool,
) configPatchSpec {

	valueBool := defaultVal
	return configPatchSpec{
		Op:   "add",
		Path: path,
		Value: configPatchValue{
			ValueBool: &valueBool,
		},
	}
}

func newStrPatch(
	path string,
	defaultVal string,
) configPatchSpec {

	valueStr := defaultVal
	return configPatchSpec{
		Op:   "add",
		Path: path,
		Value: configPatchValue{
			ValueStr: &valueStr,
		},
	}
}
