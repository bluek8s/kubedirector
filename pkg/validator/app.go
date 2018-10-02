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
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateServiceRoles checks service_ids and role_id from role_services
// in the config section, to ensure that they refer to legal/existing service
// and role definitions.
func validateServiceRoles(
	appCR *kdv1.KubeDirectorApp,
	allRoleIDs []string,
	allServiceIDs []string,
) string {

	var errorMessages []string
	for _, nodeRole := range appCR.Spec.Config.RoleServices {
		if !shared.StringInList(nodeRole.RoleID, allRoleIDs) {
			invalidMsg := fmt.Sprintf(
				invalidNodeRoleID,
				nodeRole.RoleID,
				strings.Join(allRoleIDs, ","),
			)
			errorMessages = append(errorMessages, invalidMsg)
		}
		for _, serviceID := range nodeRole.ServiceIDs {
			if !shared.StringInList(serviceID, allServiceIDs) {
				invalidMsg := fmt.Sprintf(
					invalidServiceID,
					serviceID,
					strings.Join(allServiceIDs, ","),
				)
				errorMessages = append(errorMessages, invalidMsg)
			}
		}
	}

	if len(errorMessages) == 0 {
		return ""
	}
	return strings.Join(errorMessages, "\n")
}

// validateSelectedRoles checks the selected_roles array to make sure it
// only contains valid role IDs.
func validateSelectedRoles(
	appCR *kdv1.KubeDirectorApp,
	allRoleIDs []string,
) string {

	var errorMessages []string
	for _, role := range appCR.Spec.Config.SelectedRoles {
		if catalog.GetRoleFromID(appCR, role) == nil {
			invalidMsg := fmt.Sprintf(
				invalidSelectedRoleID,
				role,
				strings.Join(allRoleIDs, ","),
			)
			errorMessages = append(errorMessages, invalidMsg)
		}
	}

	if len(errorMessages) == 0 {
		return ""
	}
	return strings.Join(errorMessages, "\n")
}

// admitAppCR is the top-level app validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitAppCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var errorMessages []string

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	raw := ar.Request.Object.Raw
	appCR := kdv1.KubeDirectorApp{}

	if err := json.Unmarshal(raw, &appCR); err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + err.Error(),
		}
		return &admitResponse
	}

	allRoleIDs := catalog.GetAllRoleIDs(&appCR)
	allServiceIDs := catalog.GetAllServiceIDs(&appCR)

	// Verify node services from the config section of the app
	serviceRoleErr := validateServiceRoles(&appCR, allRoleIDs, allServiceIDs)
	if serviceRoleErr != "" {
		errorMessages = append(errorMessages, serviceRoleErr)
	}

	// Verify selected_roles from the config section of the app
	selectedRoleErr := validateSelectedRoles(&appCR, allRoleIDs)
	if selectedRoleErr != "" {
		errorMessages = append(errorMessages, selectedRoleErr)
	}

	if len(errorMessages) == 0 {
		admitResponse.Allowed = true
	} else {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + strings.Join(errorMessages, "\n"),
		}
	}

	return &admitResponse
}
