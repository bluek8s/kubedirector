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
	"strconv"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type appPatchSpec struct {
	Op    string        `json:"op"`
	Path  string        `json:"path"`
	Value appPatchValue `json:"value,omitempty"`
}

type appPatchValue struct {
	packageInfoValue *kdv1.SetupPackageInfo
	stringValue      *string
	stringSliceValue *[]string
}

func (obj appPatchValue) MarshalJSON() ([]byte, error) {

	if obj.packageInfoValue != nil {
		return json.Marshal(obj.packageInfoValue)
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

// validateServiceRoles checks serviceIDs and roleID from roleServices
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

// validateSelectedRoles checks the selectedRoles array to make sure it
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

// validateRoles checks each role for property constraints not expressible
// in the schema. If any overrideable properties are unspecified, the corresponding
// global values are used. This will add an PATCH spec for mutation the app CR.
// The role in appCR will be correspondingly updated so that it can later be
// used to check whether the resulting CR differs from the current stored appCR.
func validateRoles(
	appCR *kdv1.KubeDirectorApp,
	patches []appPatchSpec,
	valErrors []string,
) ([]appPatchSpec, []string) {

	// Any global defaults will be removed from the CR. Remember their values
	// though for use in populating the role definitions.
	var globalImageRepoTag *string
	var globalSetupPackageInfo *kdv1.SetupPackageInfo
	var globalPersistDirs *[]string
	var globalEventList *[]string
	var globalMaxLogLinesDump *string

	if appCR.Spec.DefaultImageRepoTag == nil {
		globalImageRepoTag = nil
	} else {
		tagCopy := *appCR.Spec.DefaultImageRepoTag
		globalImageRepoTag = &tagCopy
		appCR.Spec.DefaultImageRepoTag = nil
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/defaultImageRepoTag",
			},
		)
	}
	if !appCR.Spec.DefaultSetupPackage.IsSet {
		globalSetupPackageInfo = nil
	} else {
		if appCR.Spec.DefaultSetupPackage.IsNull {
			globalSetupPackageInfo = nil
		} else {
			packageInfoCopy := appCR.Spec.DefaultSetupPackage.Info
			globalSetupPackageInfo = &packageInfoCopy
		}
		appCR.Spec.DefaultSetupPackage = kdv1.SetupPackage{}
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/defaultConfigPackage",
			},
		)
	}
	if appCR.Spec.DefaultPersistDirs == nil {
		globalPersistDirs = nil
	} else {
		dirsCopy := make([]string, len(*appCR.Spec.DefaultPersistDirs))
		copy(dirsCopy, *appCR.Spec.DefaultPersistDirs)
		globalPersistDirs = &dirsCopy
		appCR.Spec.DefaultPersistDirs = nil
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/defaultPersistDirs",
			},
		)
	}
	if appCR.Spec.DefaultEventList == nil {
		globalEventList = nil
	} else {
		eventsCopy := make([]string, len(*appCR.Spec.DefaultEventList))
		copy(eventsCopy, *appCR.Spec.DefaultEventList)
		globalEventList = &eventsCopy
		appCR.Spec.DefaultEventList = nil
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/defaultEventList",
			},
		)
	}
	if appCR.Spec.DefaultMaxLogLinesDump == nil {
		globalMaxLogLinesDump = nil
	} else {
		globalMaxLogLinesDump = appCR.Spec.DefaultMaxLogLinesDump
		patches = append(
			patches,
			appPatchSpec{
				Op:   "remove",
				Path: "/spec/defaultMaxLogLinesDump",
			},
		)
	}

	// OK let's do the roles.
	numRoles := len(appCR.Spec.NodeRoles)
	for index := 0; index < numRoles; index++ {
		role := &(appCR.Spec.NodeRoles[index])
		if role.SetupPackage.IsSet == false {
			// Nothing specified so, inherit the global specification
			if globalSetupPackageInfo == nil {
				role.SetupPackage.IsSet = true
				role.SetupPackage.IsNull = true
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/configPackage",
						Value: appPatchValue{
							stringValue: nil,
						},
					},
				)
			} else {
				role.SetupPackage.IsSet = true
				role.SetupPackage.IsNull = false
				role.SetupPackage.Info = *globalSetupPackageInfo
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/configPackage",
						Value: appPatchValue{
							packageInfoValue: globalSetupPackageInfo,
						},
					},
				)
			}
		}
		if role.MinStorage != nil {
			_, minErr := resource.ParseQuantity(role.MinStorage.Size)
			if minErr != nil {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidMinStorageDef,
						role.ID,
					),
				)
			}
		}
		if role.ContainerSpec != nil {
			if role.ContainerSpec.Tty {
				if !role.ContainerSpec.Stdin {
					valErrors = append(
						valErrors,
						fmt.Sprintf(
							ttyWithoutStdin,
							role.ID,
						),
					)
				}
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
			role.ImageRepoTag = globalImageRepoTag
			patches = append(
				patches,
				appPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(index) + "/imageRepoTag",
					Value: appPatchValue{
						stringValue: globalImageRepoTag,
					},
				},
			)
		}
		if role.PersistDirs == nil {
			if globalPersistDirs != nil {
				role.PersistDirs = globalPersistDirs
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/persistDirs",
						Value: appPatchValue{
							stringSliceValue: globalPersistDirs,
						},
					},
				)
			}
		}
		if role.EventList == nil {
			if globalEventList != nil {
				role.EventList = globalEventList
				patches = append(
					patches,
					appPatchSpec{
						Op:   "add",
						Path: "/spec/roles/" + strconv.Itoa(index) + "/eventList",
						Value: appPatchValue{
							stringSliceValue: globalEventList,
						},
					},
				)
			}
		}
		if role.MaxLogLinesDump == "" && globalMaxLogLinesDump != nil {
			role.MaxLogLinesDump = *globalMaxLogLinesDump
			patches = append(
				patches,
				appPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(index) + "/maxLogLinesDump",
					Value: appPatchValue{
						stringValue: globalMaxLogLinesDump,
					},
				},
			)
		}
	}

	return patches, valErrors
}

// validateServices checks each service for property constraints not
// expressible in the schema. Currently this just means checking that the
// service endpoint must specify url_schema if isDashboard is true. Any
// generated error messages will be added to the input list and returned.
func validateServices(
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	for _, service := range appCR.Spec.Services {
		if service.Endpoint.IsDashboard {
			if service.Endpoint.URLScheme == "" {
				invalidMsg := fmt.Sprintf(
					noURLScheme,
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
) *v1beta1.AdmissionResponse {

	var valErrors []string
	var patches []appPatchSpec

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// Reject a delete if the app CR is currently in use.
	if ar.Request.Operation == v1beta1.Delete {
		references := shared.ClustersUsingApp(
			ar.Request.Namespace,
			ar.Request.Name,
		)
		if len(references) != 0 {
			referencesStr := strings.Join(references, ", ")
			appInUseMsg := fmt.Sprintf(
				appInUse,
				referencesStr,
			)
			admitResponse.Result = &metav1.Status{
				Message: "\n" + appInUseMsg,
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
				valErrors = append(valErrors, failedToPatch)
			}
		}
	}

	// Reject an update if the app CR is currently in use AND this update is
	// changing the spec. Note that we don't do this at the beginning of the
	// handler because we want to get defaults populated before comparing.
	if ar.Request.Operation == v1beta1.Update {
		references := shared.ClustersUsingApp(
			ar.Request.Namespace,
			ar.Request.Name,
		)
		if len(references) != 0 {
			prevAppCR := kdv1.KubeDirectorApp{}
			prevRaw := ar.Request.OldObject.Raw
			if prevJSONErr := json.Unmarshal(prevRaw, &prevAppCR); prevJSONErr != nil {
				admitResponse.Result = &metav1.Status{
					Message: "\n" + prevJSONErr.Error(),
				}
				return &admitResponse
			}
			// Before doing the comparison, make sure we ignore differences
			// in the global default setup package. Global defaults should
			// NOT be set at this point in either object, and if they are then
			// they have no functional effect on kdclusters, but there was a
			// bug in KD pre-0.5 that could leave defaultConfigPackage set
			// to null. See the commit comments in the PR that closes issue
			// #319 for more details.
			prevAppCR.Spec.DefaultSetupPackage = appCR.Spec.DefaultSetupPackage
			if !equality.Semantic.DeepEqual(appCR.Spec, prevAppCR.Spec) {
				referencesStr := strings.Join(references, ", ")
				appInUseMsg := fmt.Sprintf(
					appInUse,
					referencesStr,
				)
				admitResponse.Result = &metav1.Status{
					Message: "\n" + appInUseMsg,
				}
				return &admitResponse
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
