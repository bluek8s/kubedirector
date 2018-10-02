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
	"k8s.io/apimachinery/pkg/api/errors"
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
// cluster CR against the cardinality value from the app CR. It can also
// return a list of PATCH specs that will populate default members values as
// necessary.
func validateCardinality(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
) (string, []membersPatchSpec) {

	var errorMessages []string
	var patches []membersPatchSpec

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
				errorMessages = append(
					errorMessages,
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

	if len(errorMessages) == 0 {
		return "", patches
	}
	return strings.Join(errorMessages, "\n"), nil
}

// validateRoles checks that 1) all configured roles actually exist in the
// app type, and 2) all active roles (according to the app config) are
// covered by the cluster config.
func validateRoles(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
) string {

	var configuredRoles, errorMessages []string

	allRoles := catalog.GetAllRoleIDs(appCR)
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
			errorMessages = append(errorMessages, invalidRoleMsg)
		}
	}
	for _, activeRole := range catalog.GetSelectedRoleIDs(appCR) {
		if !shared.StringInList(activeRole, configuredRoles) {
			unconfiguredRoleMsg := fmt.Sprintf(
				unconfiguredRole,
				activeRole,
				appCR.Name,
			)
			errorMessages = append(errorMessages, unconfiguredRoleMsg)
		}
	}

	if len(errorMessages) == 0 {
		return ""
	}
	return strings.Join(errorMessages, "\n")
}

// admitClusterCR is the top-level cluster validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitClusterCR(
	ar *v1beta1.AdmissionReview,
	handlerState *reconciler.Handler,
) *v1beta1.AdmissionResponse {

	var errorMessages []string

	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	raw := ar.Request.Object.Raw
	clusterCR := kdv1.KubeDirectorCluster{}

	if err := json.Unmarshal(raw, &clusterCR); err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + err.Error(),
		}
		return &admitResponse
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
			currentCluster, err := observer.GetCluster(
				clusterCR.Namespace,
				clusterCR.Name,
			)
			if err != nil {
				if errors.IsNotFound(err) {
					// Go ahead and let the core API reject this.
					admitResponse.Allowed = true
					return &admitResponse
				}
				admitResponse.Result = &metav1.Status{
					Message: "\nError when fetching current cluster object",
				}
				return &admitResponse
			}
			if !reflect.DeepEqual(clusterCR.Status, currentCluster.Status) {
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

	// Validate app first
	appCR, err := catalog.GetApp(&clusterCR)

	// If app is bad, no need to continue with rest of the validation
	if err != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + fmt.Sprintf(invalidAppMessage, clusterCR.Spec.AppID),
		}
		return &admitResponse
	}

	// Validate cardinality
	cardinalityErr, membersPatches := validateCardinality(&clusterCR, appCR)
	if cardinalityErr != "" {
		errorMessages = append(errorMessages, cardinalityErr)
	}

	// Validate that roles are known & sufficient
	rolesErr := validateRoles(&clusterCR, appCR)
	if rolesErr != "" {
		errorMessages = append(errorMessages, rolesErr)
	}

	if len(errorMessages) == 0 {
		if membersPatches != nil {
			patchResult, patchErr := json.Marshal(membersPatches)
			if patchErr == nil {
				admitResponse.Patch = patchResult
				patchType := v1beta1.PatchTypeJSONPatch
				admitResponse.PatchType = &patchType
			} else {
				errorMessages = append(errorMessages, defaultMemberErr)
			}
		}
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
