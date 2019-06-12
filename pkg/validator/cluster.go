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
	"reflect"
	"strconv"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// clusterPatchSpec is used to create the PATCH operation for populating
// default values for omitted properties.
type clusterPatchSpec struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value clusterPatchValue `json:"value"`
}

type clusterPatchValue struct {
	ValueInt *int32
	ValueStr *string
}

func (obj clusterPatchValue) MarshalJSON() ([]byte, error) {
	if obj.ValueInt != nil {
		return json.Marshal(obj.ValueInt)
	}
	return json.Marshal(obj.ValueStr)
}

// validateCardinality checks the member count specified for a role in the
// cluster CR against the cardinality value from the app CR. Any generated
// error messages will be added to the input list and returned. If there were
// no errors generated, a list of PATCH specs will also be returned for the
// purpose of populating default members values as necessary.
func validateCardinality(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) ([]string, []clusterPatchSpec) {

	var patches []clusterPatchSpec
	anyError := false

	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		appRole := catalog.GetRoleFromID(appCR, role.Name)
		if appRole == nil {
			// Do nothing; this error will be reported from validateRoles.
			continue
		}
		cardinality, isScaleOut := catalog.GetRoleCardinality(appRole)
		if role.Members != nil {
			var invalidMemberCount = false
			if isScaleOut {
				if *(role.Members) < cardinality {
					invalidMemberCount = true
				}
			} else {
				if *(role.Members) != cardinality {
					invalidMemberCount = true
				}
			}
			if invalidMemberCount {
				anyError = true
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidCardinality,
						role.Name,
						*(role.Members),
						appRole.Cardinality,
					),
				)
			}
		} else {
			role.Members = &cardinality
			patches = append(
				patches,
				clusterPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(i) + "/members",
					Value: clusterPatchValue{
						ValueInt: role.Members,
					},
				},
			)
		}
	}

	if anyError {
		var emptyPatchList []clusterPatchSpec
		return valErrors, emptyPatchList
	}
	return valErrors, patches
}

// validateClusterRoles checks that 1) all configured roles actually exist in
// the app type, 2) all active roles (according to the app config) that
// require more than 0 members are covered by the cluster config, and 3) we
// don't try to configure the same role more than once. Any generated error
// messages will be added to the input list and returned.
func validateClusterRoles(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	var configuredRoles []string

	allRoles := catalog.GetAllRoleIDs(appCR)
	roleSeen := make(map[string]bool)
	uniqueRoles := true
	for _, role := range cr.Spec.Roles {
		if shared.StringInList(role.Name, allRoles) {
			configuredRoles = append(configuredRoles, role.Name)
		} else {
			invalidRoleMsg := fmt.Sprintf(
				invalidRole,
				role.Name,
				appCR.Name,
				strings.Join(allRoles, ","),
			)
			valErrors = append(valErrors, invalidRoleMsg)
		}
		if _, ok := roleSeen[role.Name]; ok {
			uniqueRoles = false
		}
		roleSeen[role.Name] = true
	}
	if !uniqueRoles {
		valErrors = append(valErrors, nonUniqueRoleID)
	}
	for _, activeRole := range catalog.GetSelectedRoleIDs(appCR) {
		if !shared.StringInList(activeRole, configuredRoles) {
			role := catalog.GetRoleFromID(appCR, activeRole)
			// If our app CR validation is on point this should never be nil,
			// but it doesn't hurt to be careful.
			if role != nil {
				validMin, _ := catalog.GetRoleCardinality(role)
				if validMin != 0 {
					unconfiguredRoleMsg := fmt.Sprintf(
						unconfiguredRole,
						activeRole,
						appCR.Name,
					)
					valErrors = append(valErrors, unconfiguredRoleMsg)
				}
			}
		}
	}
	return valErrors
}

// validateGeneralChanges checks for modifications to any property that is
// not ever allowed to change after initial deployment. Currently this covers
// the top-level app. Any generated error messages will be added to the input
// list and returned.
func validateGeneralChanges(
	cr *kdv1.KubeDirectorCluster,
	prevCr *kdv1.KubeDirectorCluster,
	valErrors []string,
) []string {

	if cr.Spec.AppID != prevCr.Spec.AppID {
		appModifiedMsg := fmt.Sprintf(
			modifiedProperty,
			"app",
		)
		valErrors = append(valErrors, appModifiedMsg)
	}

	return valErrors
}

// validateRoleChanges checks for modifications to role properties. The
// members property of a role can always be changed (within cardinality
// constraints that are checked elsewhere). However other properties cannot
// be changed unless the role currently has no members. Any generated error
// messages will be added to the input list and returned.
func validateRoleChanges(
	cr *kdv1.KubeDirectorCluster,
	prevCr *kdv1.KubeDirectorCluster,
	valErrors []string,
) []string {

	prevRoles := make(map[string]*kdv1.Role)
	numPrevRoles := len(prevCr.Spec.Roles)
	for i := 0; i < numPrevRoles; i++ {
		p := &(prevCr.Spec.Roles[i])
		prevRoles[p.Name] = p
	}
	prevRoleHasStatus := make(map[string]bool)
	if prevCr.Status != nil {
		for _, s := range prevCr.Status.Roles {
			prevRoleHasStatus[s.Name] = true
		}
	}
	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		// Skip checking for modified properties if there are no existing role
		// members; i.e. no populated role status at all. Note that this is
		// different from just checking the "members" count of prevRole;
		// "members" is just the desired value which may not yet be reconciled.
		// Restricting change to when all role members are gone is good because:
		// - We won't show misleading configuration for existing members.
		// - We aren't required to do any explicit statefulset reconfig. The
		//   previous role's statefulset will have been deleted.
		if _, ok := prevRoleHasStatus[role.Name]; !ok {
			continue
		}
		// If there is status but no existing spec, the role was deleted.
		// Don't allow resurrecting it until it has finished going away.
		prevRole, hasPrevRole := prevRoles[role.Name]
		if !hasPrevRole {
			roleModifiedMsg := fmt.Sprintf(
				modifiedRole,
				role.Name,
			)
			valErrors = append(valErrors, roleModifiedMsg)
			continue
		}
		// There is status (i.e. current members) and a current spec. Reject
		// the new spec if anything other than the members count is different.
		compareRole := *role
		compareRole.Members = prevRole.Members
		if !reflect.DeepEqual(&compareRole, prevRole) {
			roleModifiedMsg := fmt.Sprintf(
				modifiedRole,
				role.Name,
			)
			valErrors = append(valErrors, roleModifiedMsg)
		}
	}
	return valErrors
}

// validateRoleStorageClass verifies storageClassName definition for a role
// If storage section is defined for a role, see if a storageClassName is
// also defined and if so validate it. If not, but a default is present in the
// global config, validate and use that one. Final fallback is to check to see
// if the underlying platform has a default storage class.
func validateRoleStorageClass(
	cr *kdv1.KubeDirectorCluster,
	valErrors []string,
	kdConfig *kdv1.KubeDirectorConfig,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	var validateDefault = false
	var missingDefault = false

	globalStorageClass := kdConfig.Spec.StorageClass
	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		if role.Storage.Size == "" {
			// No storage section.
			continue
		}
		storageClass := role.Storage.StorageClass
		if storageClass != nil {
			// Storage class is specified, so validate it.
			_, scErr := observer.GetStorageClass(*storageClass)
			if scErr != nil {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidRoleStorageClass,
						*storageClass,
						role.Name,
					),
				)
			}
			continue
		}
		// No storage class specified. How we handle this depends on whether
		// there is a KubeDirector config-specified default.
		if globalStorageClass != nil {
			// Yep. Use that, and remember to validate it when we're done
			// looping.
			validateDefault = true
			role.Storage.StorageClass = globalStorageClass
		} else {
			// Nope. Let's see what K8s says is the default.
			scK8sDefault, _ := observer.GetDefaultStorageClass()
			if scK8sDefault == nil {
				missingDefault = true
				continue
			}
			role.Storage.StorageClass = &(scK8sDefault.Name)
		}
		// Patch in the defaulted value unless it is missing.
		if !missingDefault {
			patches = append(
				patches,
				clusterPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(i) + "/storage/storageClassName",
					Value: clusterPatchValue{
						ValueStr: role.Storage.StorageClass,
					},
				},
			)
		}
	}

	if missingDefault {
		valErrors = append(
			valErrors,
			noDefaultStorageClass,
		)
	} else if validateDefault {
		_, err := observer.GetStorageClass(*globalStorageClass)
		if err != nil {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					badDefaultStorageClass,
					*globalStorageClass,
				),
			)
		}
	}

	return valErrors, patches
}

// validateMinResources function checks to see if minimum resource requiements
// are specified for each role, by checking against associated app roles' minimum
// requirement
func validateMinResources(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	valErrors []string,
) []string {

	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		appRole := catalog.GetRoleFromID(appCR, role.Name)
		if appRole == nil {
			// Do nothing; this error will be reported from validateRoles.
			continue
		}

		minResources := catalog.GetRoleMinResources(appRole)
		if minResources == nil {
			// No minimum requirements for this role.
			continue
		}

		logError := func(
			resName string,
			resValue string,
			roleName string,
			expValue string,
			valErrors []string) []string {

			return append(
				valErrors,
				fmt.Sprintf(
					invalidResource,
					resName,
					resValue,
					roleName,
					expValue,
				),
			)
		}

		for resKey, resVal := range *minResources {
			if resVal.IsZero() {
				continue
			}

			if limit, ok := role.Resources.Requests[resKey]; ok {
				if limit.Value() < resVal.Value() {
					valErrors = logError(resKey.String(), limit.String(), role.Name, resVal.String(), valErrors)
				}
			} else {
				valErrors = logError(resKey.String(), "0", role.Name, resVal.String(), valErrors)
			}
		}
	}

	return valErrors
}

// addServiceType function checks to see if serviceType is provided for a
// cluster CR. If unspecified, check to see if there is a default serviceType
// provided through kubedirector's config CR, otherwise use a global constant
// for service type. In either of those cases add an entry to PATCH spec for mutating
// cluster CR.
func addServiceType(
	cr *kdv1.KubeDirectorCluster,
	kdConfig *kdv1.KubeDirectorConfig,
	patches []clusterPatchSpec,
) []clusterPatchSpec {

	if cr.Spec.ServiceType != nil {
		return patches
	}

	serviceType := defaultServiceType
	if kdConfig.Spec.ServiceType != nil {
		serviceType = *kdConfig.Spec.ServiceType
	}
	cr.Spec.ServiceType = &serviceType

	patches = append(
		patches,
		clusterPatchSpec{
			Op:   "add",
			Path: "/spec/serviceType",
			Value: clusterPatchValue{
				ValueStr: cr.Spec.ServiceType,
			},
		},
	)

	return patches
}

// admitClusterCR is the top-level cluster validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitClusterCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var valErrors []string
	var patches []clusterPatchSpec
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
	clusterCR := kdv1.KubeDirectorCluster{}
	if jsonErr := json.Unmarshal(raw, &clusterCR); jsonErr != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + jsonErr.Error(),
		}
		return &admitResponse
	}

	// If this is an update, get the previous version of the object ready for
	// use in some checks.
	prevClusterCR := kdv1.KubeDirectorCluster{}
	if ar.Request.Operation == v1beta1.Update {
		prevRaw := ar.Request.OldObject.Raw
		if prevJSONErr := json.Unmarshal(prevRaw, &prevClusterCR); prevJSONErr != nil {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + prevJSONErr.Error(),
			}
			return &admitResponse
		}
	}

	// Don't allow Status to be updated except by KubeDirector. Do this by
	// using one-time codes known by KubeDirector.
	if clusterCR.Status != nil {
		expectedStatusGen, ok := reconciler.ReadStatusGen(
			&clusterCR,
			handlerState,
		)
		// Reject this write if either of:
		// - KubeDirector doesn't know about the cluster resource
		// - this status generation UID is not what we're expecting a write for
		if !ok || clusterCR.Status.GenerationUID != expectedStatusGen.UID {
			admitResponse.Result = &metav1.Status{
				Message: "\nKubeDirector-related status properties are read-only",
			}
			return &admitResponse
		}
		// If this status generation UID has already been admitted previously,
		// it's OK to write the status again as long as nothing is changing.
		// (For example we'll see this when a PATCH happens.)
		if expectedStatusGen.Validated {
			if !reflect.DeepEqual(clusterCR.Status, prevClusterCR.Status) {
				admitResponse.Result = &metav1.Status{
					Message: "\nKubeDirector-related status properties are read-only",
				}
				return &admitResponse
			}
		}
	}
	reconciler.ValidateStatusGen(
		&clusterCR,
		handlerState,
	)

	// Shortcut out of here if the spec is not being changed. Among other
	// things this allows KD to update status or metadata even if the
	// referenced app is bad/gone.
	if ar.Request.Operation == v1beta1.Update {
		if reflect.DeepEqual(clusterCR.Spec, prevClusterCR.Spec) {
			admitResponse.Allowed = true
			return &admitResponse
		}
	}

	// At this point, if app is bad, no need to continue with validation.
	appCR, err := catalog.GetApp(&clusterCR)
	if err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + fmt.Sprintf(invalidAppMessage, clusterCR.Spec.AppID),
		}
		return &admitResponse
	}

	// Fetch global config CR (if present)
	kdConfigCR, _ := observer.GetKDConfig(shared.KubeDirectorGlobalConfig)

	// Validate cardinality and generate patches for defaults members values.
	valErrors, patches = validateCardinality(&clusterCR, appCR, valErrors)

	// Validate that roles are known & sufficient.
	valErrors = validateClusterRoles(&clusterCR, appCR, valErrors)

	// Validate minimum resources for all roles
	valErrors = validateMinResources(&clusterCR, appCR, valErrors)

	valErrors, patches = validateRoleStorageClass(
		&clusterCR,
		valErrors,
		kdConfigCR,
		patches,
	)

	patches = addServiceType(&clusterCR, kdConfigCR, patches)

	// If cluster already exists, check for property changes.
	if ar.Request.Operation == v1beta1.Update {
		var changeErrors []string
		changeErrors = validateGeneralChanges(&clusterCR, &prevClusterCR, changeErrors)
		changeErrors = validateRoleChanges(&clusterCR, &prevClusterCR, changeErrors)
		// If un-change-able properties are being changed, ignore all other error
		// messages in favor of those. (The reason we didn't just do this check
		// first and then skip other validation is because this check depends on
		// the defaulting logic that happens in those other functions.)
		if len(changeErrors) != 0 {
			valErrors = changeErrors
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
