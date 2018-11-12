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
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// membersPatchSpec is used to create the PATCH operation for adding a default
// member count to a role.
type membersPatchSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int32  `json:"value"`
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
) ([]string, []membersPatchSpec) {

	var patches []membersPatchSpec
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
			patches = append(
				patches,
				membersPatchSpec{
					Op:    "add",
					Path:  "/spec/roles/" + strconv.Itoa(i) + "/members",
					Value: cardinality,
				},
			)
		}
	}

	if anyError {
		var emptyPatchList []membersPatchSpec
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
// the top-level app and serviceType properties. Any generated error messages
// will be added to the input list and returned.
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
	if cr.Spec.ServiceType != prevCr.Spec.ServiceType {
		serviceTypeModifiedMsg := fmt.Sprintf(
			modifiedProperty,
			"serviceType",
		)
		valErrors = append(valErrors, serviceTypeModifiedMsg)
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

// admitClusterCR is the top-level cluster validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitClusterCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var valErrors []string
	var membersPatches []membersPatchSpec
	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// If this is a delete, all we need to do is note that this cluster is
	// no longer referencing its app type.
	if ar.Request.Operation == v1beta1.Delete {
		reconciler.RemoveClusterAppReference(
			ar.Request.Namespace,
			ar.Request.Name,
			&(handlerState.ClusterState),
		)
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

	// Make sure that if we're adding a new cluster, we don't escape from
	// this handler without recording a reference to its app type.
	defer func() {
		if ar.Request.Operation == v1beta1.Create && admitResponse.Allowed == true {
			reconciler.AddClusterAppReference(
				&clusterCR,
				&(handlerState.ClusterState),
			)
		}
	}()

	// Now do validation for create/update.

	prevClusterCR := kdv1.KubeDirectorCluster{}
	if ar.Request.Operation == v1beta1.Update {
		// On update, get the previous version of the object ready for use in
		// some checks.
		prevRaw := ar.Request.OldObject.Raw
		if prevJsonErr := json.Unmarshal(prevRaw, &prevClusterCR); prevJsonErr != nil {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + prevJsonErr.Error(),
			}
			return &admitResponse
		}
	}

	// Don't allow Status to be updated except by KubeDirector. Do this by
	// using one-time codes known by KubeDirector.
	if clusterCR.Status != nil {
		expectedStatusGen, ok := reconciler.ReadStatusGen(
			&clusterCR,
			&(handlerState.ClusterState),
		)
		// Reject this write if either of:
		// - KubeDirector doesn't know about the cluster resource
		// - this status generation UID is not what we're expecting a write for
		if !ok || clusterCR.Status.GenerationUid != expectedStatusGen.Uid {
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
		&(handlerState.ClusterState),
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

	// If cluster already exists, check for property changes.
	if ar.Request.Operation == v1beta1.Update {
		valErrors = validateGeneralChanges(&clusterCR, &prevClusterCR, valErrors)
		valErrors = validateRoleChanges(&clusterCR, &prevClusterCR, valErrors)
		// We coooooould continue to do other validation at this point, but
		// that could be misleading. Let's not do other validation unless we
		// know that only change-able properties are being changed.
		if len(valErrors) != 0 {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + strings.Join(valErrors, "\n"),
			}
			return &admitResponse
		}
	}

	// Validate cardinality and generate patches for defaults members values.
	valErrors, membersPatches = validateCardinality(&clusterCR, appCR, valErrors)

	// Validate that roles are known & sufficient.
	valErrors = validateClusterRoles(&clusterCR, appCR, valErrors)

	if len(valErrors) == 0 {
		if len(membersPatches) != 0 {
			patchResult, patchErr := json.Marshal(membersPatches)
			if patchErr == nil {
				admitResponse.Patch = patchResult
				patchType := v1beta1.PatchTypeJSONPatch
				admitResponse.PatchType = &patchType
			} else {
				valErrors = append(valErrors, defaultMemberErr)
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
