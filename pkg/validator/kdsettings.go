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
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateName validates CR name. This has to match the expected value
func validateName(
	name string,
	valErrors []string,
) []string {

	if name != shared.KubeDirectorSettingsCR {
		invalidNameMsg := fmt.Sprintf(
			invalidSettingsName,
			shared.KubeDirectorSettingsCR,
			name,
		)

		valErrors = append(valErrors, invalidNameMsg)
	}
	return valErrors
}

// validateSettingsStorageClass validates storageClassName by checking
// for a valid storageClass k8s resource.
func validateSettingsStorageClass(
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
			err,
		),
	)

	return valErrors
}

// admitKDSettingsCR is the top-level settings validation function, which invokes
// specific validation subroutines and composes the admission response.
func admitKDSettingsCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var valErrors []string

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	raw := ar.Request.Object.Raw
	settingsCR := kdv1.KubeDirectorSettings{}

	if err := json.Unmarshal(raw, &settingsCR); err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + err.Error(),
		}
		return &admitResponse
	}

	// Settings Name MUST match our default
	valErrors = validateName(settingsCR.Name, valErrors)

	// Validate storage class name if present
	valErrors = validateSettingsStorageClass(settingsCR.Spec.StorageClass, valErrors)

	if len(valErrors) == 0 {
		admitResponse.Allowed = true
	} else {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + strings.Join(valErrors, "\n"),
		}
	}

	return &admitResponse
}
