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

// validateUniqueness checks the lists of roles and service IDs for duplicates.
// Any generated error messages will be added to the input list and returned.
func validateUniqueness(
	allRoleIDs []string,
	allServiceIDs []string,
	valErrors []string,
) []string {

	if !shared.ListIsUnique(allRoleIDs) {
		valErrors = append(valErrors, nonUniqueRoleID)
	}
	if !shared.ListIsUnique(allServiceIDs) {
		valErrors = append(valErrors, nonUniqueServiceID)
	}
	return valErrors
}

// validateRefUniqueness checks the lists of role references for duplicates.
// Any generated error messages will be added to the input list and returned.
func validateRefUniqueness(
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	if !shared.ListIsUnique(appCR.Spec.Config.SelectedRoles) {
		valErrors = append(valErrors, nonUniqueSelectedRole)
	}
	roleSeen := make(map[string]bool)
	for _, roleService := range appCR.Spec.Config.RoleServices {
		if _, ok := roleSeen[roleService.RoleID]; ok {
			valErrors = append(valErrors, nonUniqueServiceRole)
			break
		}
		roleSeen[roleService.RoleID] = true
	}
	return valErrors
}

// validateServiceRoles checks service_ids and role_id from role_services
// in the config section, to ensure that they refer to legal/existing service
// and role definitions. Any generated error messages will be added to the
// input list and returned.
func validateServiceRoles(
	appCR *kdv1.KubeDirectorApp,
	allRoleIDs []string,
	allServiceIDs []string,
	valErrors []string,
) []string {

	for _, nodeRole := range appCR.Spec.Config.RoleServices {
		if !shared.StringInList(nodeRole.RoleID, allRoleIDs) {
			invalidMsg := fmt.Sprintf(
				invalidNodeRoleID,
				nodeRole.RoleID,
				strings.Join(allRoleIDs, ","),
			)
			valErrors = append(valErrors, invalidMsg)
		}
		for _, serviceID := range nodeRole.ServiceIDs {
			if !shared.StringInList(serviceID, allServiceIDs) {
				invalidMsg := fmt.Sprintf(
					invalidServiceID,
					serviceID,
					strings.Join(allServiceIDs, ","),
				)
				valErrors = append(valErrors, invalidMsg)
			}
		}
	}
	return valErrors
}

// validateSelectedRoles checks the selected_roles array to make sure it
// only contains valid role IDs. Any generated error messages will be added to
// the input list and returned.
func validateSelectedRoles(
	appCR *kdv1.KubeDirectorApp,
	allRoleIDs []string,
	valErrors []string,
) []string {

	for _, role := range appCR.Spec.Config.SelectedRoles {
		if catalog.GetRoleFromID(appCR, role) == nil {
			invalidMsg := fmt.Sprintf(
				invalidSelectedRoleID,
				role,
				strings.Join(allRoleIDs, ","),
			)
			valErrors = append(valErrors, invalidMsg)
		}
	}
	return valErrors
}

// validateRoles checks each role for property constraints not expressable
// in the schema. Currently this just means checking that the role must
// specify an image if there is no top-level default image. Any generated
// error messages will be added to the input list and returned.
func validateRoles(
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	for _, role := range appCR.Spec.NodeRoles {
		if role.Image.RepoTag == "" {
			if appCR.Spec.Image.RepoTag == "" {
				invalidMsg := fmt.Sprintf(
					noDefaultImage,
					role.ID,
				)
				valErrors = append(valErrors, invalidMsg)
			}
		}
	}
	return valErrors
}

// validateServices checks each service for property constraints not
// expressable in the schema. Currently this just means checking that the
// service endpoint must specify url_schema if is_dashboard is true. Any
// generated error messages will be added to the input list and returned.
func validateServices(
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	for _, service := range appCR.Spec.Services {
		if service.Endpoint.IsDashboard {
			if service.Endpoint.URLScheme == "" {
				invalidMsg := fmt.Sprintf(
					noUrlScheme,
					service.ID,
				)
				valErrors = append(valErrors, invalidMsg)
			}
		}
	}
	return valErrors
}

// admitAppCR is the top-level app validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitAppCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var valErrors []string

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

	valErrors = validateUniqueness(allRoleIDs, allServiceIDs, valErrors)
	valErrors = validateRefUniqueness(&appCR, valErrors)
	valErrors = validateServiceRoles(&appCR, allRoleIDs, allServiceIDs, valErrors)
	valErrors = validateSelectedRoles(&appCR, allRoleIDs, valErrors)
	valErrors = validateRoles(&appCR, valErrors)
	valErrors = validateServices(&appCR, valErrors)

	if len(valErrors) == 0 {
		admitResponse.Allowed = true
	} else {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + strings.Join(valErrors, "\n"),
		}
	}

	return &admitResponse
}
