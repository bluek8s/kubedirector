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
	"strconv"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type appPatchSpec struct {
	Op    string        `json:"op"`
	Path  string        `json:"path"`
	Value appPatchValue `json:"value,omitempty"`
}

type appPatchValue struct {
	packageURLValue  *packageURL
	stringValue      *string
	stringSliceValue *[]string
}

type packageURL struct {
	URL string `json:"package_url"`
}

func (obj appPatchValue) MarshalJSON() ([]byte, error) {
	if obj.packageURLValue != nil {
		return json.Marshal(obj.packageURLValue)
	}
	if obj.stringValue != nil {
		return json.Marshal(obj.stringValue)
	}
	return json.Marshal(obj.stringSliceValue)
}

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
// in the schema. If any overrideable properties are unspecified, the corresponding
// global values are used. This will add an PATCH spec for mutation the app CR.
func validateRoles(
	appCR *kdv1.KubeDirectorApp,
	patches []appPatchSpec,
	valErrors []string,
) ([]appPatchSpec, []string) {

	var globalImageRepoTag *string
	var globalSetupPackageURL *string
	var globalPersistDirs *[]string

	globalImageRepoTag = appCR.Spec.DefaultImageRepoTag
	if globalImageRepoTag != nil {
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/default_image_repo_tag",
			},
		)
	}

	if (appCR.Spec.DefaultSetupPackage.IsSet == false) || (appCR.Spec.DefaultSetupPackage.IsNull == true) {
		globalSetupPackageURL = nil
	} else {
		globalSetupPackageURL = &appCR.Spec.DefaultSetupPackage.PackageURL.PackageURL

		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/default_config_package",
			},
		)
	}

	globalPersistDirs = appCR.Spec.DefaultPersistDirs
	if globalPersistDirs != nil {
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/default_persist_dirs",
			},
		)
	}

	for index, role := range appCR.Spec.NodeRoles {
		if role.SetupPackage.IsSet == false {
			// Nothing specified so, inherit the global specification
			if globalSetupPackageURL == nil {
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/config_package",
						Value: appPatchValue{
							stringValue: nil,
						},
					},
				)
			} else {
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/config_package",
						Value: appPatchValue{
							packageURLValue: &packageURL{URL: *globalSetupPackageURL},
						},
					},
				)
			}
		}

		if role.ImageRepoTag == nil {
			// We allow roles to have different container images but unlike the
			// setup package there cannot be a role with no image.
			if globalImageRepoTag == nil {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						noDefaultImage,
						role.ID,
					),
				)
				continue
			}
			// No special image specified so inherit from global.
			patches = append(
				patches,
				appPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(index) + "/image_repo_tag",
					Value: appPatchValue{
						stringValue: globalImageRepoTag,
					},
				},
			)
		}

		// If role didn't set persist dirs, take the default (if any).
		if role.PersistDirs == nil {
			if globalPersistDirs != nil {
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/persist_dirs",
						Value: appPatchValue{
							stringSliceValue: globalPersistDirs,
						},
					},
				)
			}
		}
	}

	return patches, valErrors
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
	var patches []appPatchSpec

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// Reject an update or delete if the app CR is currently in use.
	if ar.Request.Operation == v1beta1.Update || ar.Request.Operation == v1beta1.Delete {
		references := reconciler.ClustersUsingApp(
			ar.Request.Name,
			handlerState,
		)
		if len(references) != 0 {
			referencesStr := strings.Join(references, ", ")
			admitResponse.Result = &metav1.Status{
				Message: "\nKubeDirectorApp resource cannot be deleted or modified " +
					"while referenced by the following KubeDirectorCluster resources: " +
					referencesStr,
			}
			return &admitResponse
		}
	}

	// For a delete operation, we're done now.
	if ar.Request.Operation == v1beta1.Delete {
		admitResponse.Allowed = true
		return &admitResponse
	}

	// Deserialize the object.
	raw := ar.Request.Object.Raw
	appCR := kdv1.KubeDirectorApp{}
	if jsonErr := json.Unmarshal(raw, &appCR); jsonErr != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + jsonErr.Error(),
		}
		return &admitResponse
	}

	// Now do validation for create/update.

	allRoleIDs := catalog.GetAllRoleIDs(&appCR)
	allServiceIDs := catalog.GetAllServiceIDs(&appCR)

	valErrors = validateUniqueness(allRoleIDs, allServiceIDs, valErrors)
	valErrors = validateRefUniqueness(&appCR, valErrors)
	valErrors = validateServiceRoles(&appCR, allRoleIDs, allServiceIDs, valErrors)
	valErrors = validateSelectedRoles(&appCR, allRoleIDs, valErrors)
	patches, valErrors = validateRoles(&appCR, patches, valErrors)
	valErrors = validateServices(&appCR, valErrors)

	if len(valErrors) == 0 {
		if len(patches) != 0 {
			patchResult, patchErr := json.Marshal(patches)
			if patchErr == nil {
				admitResponse.Patch = patchResult
				patchType := v1beta1.PatchTypeJSONPatch
				admitResponse.PatchType = &patchType
			} else {
				valErrors = append(
					valErrors,
					"Failed to marshal the patches.",
				)
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
