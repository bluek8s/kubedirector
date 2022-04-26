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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/controller/kubedirectorcluster"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/secretkeys"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/authentication/v1"
	sar "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type secretValidateResult int

const maxKDMembers = 1000

const (
	secretIsValid secretValidateResult = iota
	secretPrefixNotMatched
	secretNotFound
)

// clusterPatchSpec is used to create the PATCH operation for populating
// default values for omitted properties.
type clusterPatchSpec struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value clusterPatchValue `json:"value"`
}

type clusterPatchValue struct {
	ValueInt           *int32
	ValueStr           *string
	ValueClusterStatus *kdv1.KubeDirectorClusterStatus
	ValueKDSecret      *kdv1.KDSecret
	ValueSecretKey     *kdv1.SecretKey
	ValueDict          *dictValue
}

func (obj clusterPatchValue) MarshalJSON() ([]byte, error) {

	if obj.ValueInt != nil {
		return json.Marshal(obj.ValueInt)
	}
	if obj.ValueKDSecret != nil {
		return json.Marshal(obj.ValueKDSecret)
	}
	if obj.ValueClusterStatus != nil {
		return json.Marshal(obj.ValueClusterStatus)
	}
	if obj.ValueSecretKey != nil {
		return json.Marshal(obj.ValueSecretKey)
	}
	if obj.ValueDict != nil {
		return json.Marshal(obj.ValueDict)
	}
	return json.Marshal(obj.ValueStr)
}

// validateSpecChange is only called when an update is changing the spec. It
// enforces that the cluster spec may not be modified if there are pending
// member notifies or if the previous spec change has not been seen by the
// reconciler. (These invariants are required for some error-handling cases.)
// If the spec is being modified & that's ok, will return a patch to change
// the cluster overall status to "spec modified".
func validateSpecChange(
	cr *kdv1.KubeDirectorCluster,
	prevCr *kdv1.KubeDirectorCluster,
	valErrors []string,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	// If this is an update and the reconciler has not yet created the status
	// stanza, that's a problem.
	if cr.Status == nil {
		valErrors = append(
			valErrors,
			multipleSpecChange,
		)
		return valErrors, patches
	}

	// Spec change not allowed if pending notifies.
	for _, roleStatus := range cr.Status.Roles {
		for _, memberStatus := range roleStatus.Members {
			if len(memberStatus.StateDetail.PendingNotifyCmds) != 0 {
				valErrors = append(
					valErrors,
					pendingNotifies,
				)
				return valErrors, patches
			}
		}
	}

	stringStateModified := string(kubedirectorcluster.ClusterSpecModified)

	// Spec change not allowed if the overall cluster state is still
	// "spec modified".
	if cr.Status.State == stringStateModified {
		valErrors = append(
			valErrors,
			multipleSpecChange,
		)
		return valErrors, patches
	}

	// Spec is changing and that's OK. Update state to indicate a modify is
	// waiting for the reconciler to pick it up.
	newState := stringStateModified
	patches = append(
		patches,
		clusterPatchSpec{
			Op:   "replace",
			Path: "/status/state",
			Value: clusterPatchValue{
				ValueStr: &newState,
			},
		},
	)
	return valErrors, patches
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
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	anyError := false
	totalMembers := int32(0)

	numRoles := len(cr.Spec.Roles)
	rolesPath := field.NewPath("spec", "roles")
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

		totalMembers += *role.Members
		if totalMembers > maxKDMembers {
			anyError = true
			valErrors = append(
				valErrors,
				fmt.Sprint(
					maxMemberLimit,
					maxKDMembers,
				),
			)
			break
		}

		// validate user-specified labels and annotations
		var anyLabelAnnError bool
		valErrors, anyLabelAnnError = validateLabelsAndAnnotations(
			rolesPath.Index(i),
			role.PodLabels,
			role.PodAnnotations,
			role.ServiceLabels,
			role.ServiceAnnotations,
			valErrors,
		)
		anyError = anyError || anyLabelAnnError
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

// validateGeneralClusterChanges checks for modifications to any property that
// is not ever allowed to change after initial deployment. Currently this
// covers the top-level app and appCatalog. Any generated error messages will
// be added to the input list and returned.
func validateGeneralClusterChanges(
	cr *kdv1.KubeDirectorCluster,
	prevCr *kdv1.KubeDirectorCluster,
	valErrors []string,
) []string {

	// Check if cluster is ready. If so, permit the changes
	configured := (*prevCr).Status.State == "configured"
	appModified := cr.Spec.AppID != prevCr.Spec.AppID
	rollbackRequested := false

	upgradeInfo := prevCr.Status.UpgradeInfo

	crApp, err := catalog.GetApp(cr)
	if err != nil {
		valErrors = append(valErrors, err.Error())
	}

	prevCrApp, err := catalog.GetApp(prevCr)
	if err != nil {
		valErrors = append(valErrors, err.Error())
	}

	// If changed cluster has actual upgradeInfo (was in Updating state)
	// we may permit the changes for rollback case only
	// It means, the current proposed AppId should be the same as the stored in the upgradeInfo object
	if appModified && upgradeInfo != nil && !upgradeInfo.IsRollingBack {
		rollbackRequested = cr.Spec.AppID == upgradeInfo.PrevApp
	}

	// Check if any fields other than spec.app were changed
	prevAppID := prevCr.Spec.AppID
	prevCr.Spec.AppID = cr.Spec.AppID
	fieldsModified := !equality.Semantic.DeepEqual(cr.Spec, prevCr.Spec)
	prevCr.Spec.AppID = prevAppID

	// If cluster is not configured and no rollback requested, reject any changes (app-related and other)
	if !configured {
		if fieldsModified {
			errorMsg := fmt.Sprintf(clusterNotReady, prevCr.Name)
			valErrors = append(valErrors, errorMsg)
		}
		if !rollbackRequested && appModified {
			errorMsg := fmt.Sprintf(clusterAppIsUpgrading, prevCr.Name)
			valErrors = append(valErrors, errorMsg)
		}

		// In case when rollbackRequested==true && fieldModified==true
		// valErrors will be returned by the cobdition below (appModified && fieldsModified)
		// as rollbackRequested==true means also appModified==true
		if !rollbackRequested {
			return valErrors
		}
	}

	// Can't change the spec.app and other spec fields in the same time
	if appModified && fieldsModified {
		valErrors = append(valErrors, notOnlyAppModified)
		return valErrors
	}

	if appModified {

		// DistroID should stay the same
		if prevCrApp.Spec.DistroID != crApp.Spec.DistroID {
			appModifiedMsg := fmt.Sprintf(
				invalidDistroID,
				crApp.Spec.DistroID,
				prevCrApp.Spec.DistroID,
			)
			valErrors = append(valErrors, appModifiedMsg)
		}

		// App version should be different from previous one
		if prevCrApp.Spec.Version == crApp.Spec.Version {
			appModifiedMsg := fmt.Sprintf(
				versionIsNotModified,
				crApp.Spec.DistroID,
				crApp.Spec.Version,
			)
			valErrors = append(valErrors, appModifiedMsg)
		}

		// Also, the current app should support upgrade
		if prevCrApp.Spec.Upgradable == nil || !*prevCrApp.Spec.Upgradable {
			appModifiedMsg := fmt.Sprintf(
				appNotUpgradable,
				prevCrApp.Spec.DistroID,
				prevCrApp.Spec.Version,
			)
			valErrors = append(valErrors, appModifiedMsg)
		}
	}

	// appCatalog should not be nil at this point in the flow if everything
	// has worked as expected, but it doesn't hurt to be robust against that.
	appCatalogMatch := true
	if cr.Spec.AppCatalog != nil {
		if prevCr.Spec.AppCatalog != nil {
			appCatalogMatch = (*(cr.Spec.AppCatalog) == *(prevCr.Spec.AppCatalog))
		} else {
			appCatalogMatch = false
		}
	} else {
		appCatalogMatch = (prevCr.Spec.AppCatalog == nil)
	}
	if !appCatalogMatch {
		appCatalogModifiedMsg := fmt.Sprintf(
			modifiedProperty,
			"appCatalog",
		)
		valErrors = append(valErrors, appCatalogModifiedMsg)
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
		if !equality.Semantic.DeepEqual(&compareRole, prevRole) {
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
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	var validateDefault = false
	var missingDefault = false

	globalStorageClass := shared.GetDefaultStorageClass()
	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		if role.Storage == nil {
			// No storage section.
			continue
		}
		// Validate storage size.
		storageSize, err := resource.ParseQuantity(role.Storage.Size)
		if err != nil {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					invalidStorageDef,
					role.Name,
				),
			)
			break
		}
		if storageSize.Sign() != 1 {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					invalidStorageSize,
					role.Name,
				),
			)
			break
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
		if len(globalStorageClass) > 0 {
			// Yep. Use that, and remember to validate it when we're done
			// looping.
			validateDefault = true
			role.Storage.StorageClass = &globalStorageClass
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
		_, err := observer.GetStorageClass(globalStorageClass)
		if err != nil {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					badDefaultStorageClass,
					globalStorageClass,
				),
			)
		}
	}

	return valErrors, patches
}

// validateRoleSA validates whether the SA exists and if it does
// is the user allowed to access it or not
func validateRoleServiceAccount(
	cr *kdv1.KubeDirectorCluster,
	valErrs []string,
	userInfo v1.UserInfo,

) []string {

	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		if role.ServiceAccountName == "" {
			// No SA
			continue
		}
		_, erro := observer.GetServiceAccount(cr.Namespace, role.ServiceAccountName)
		if erro != nil {
			valErrs = append(valErrs,
				"service account "+role.ServiceAccountName+" requested by role "+role.Name+" does not exist")
			continue
		}
		// Convert k8s.io/api/authentication/v1".ExtraValue -> k8s.io/api/authorization/v1".ExtraValue
		xtra := make(map[string]sar.ExtraValue)
		for k, v := range userInfo.Extra {
			xtra[k] = sar.ExtraValue(v)
		}
		sar := &sar.SubjectAccessReview{
			TypeMeta: metav1.TypeMeta{
				Kind:       "SubjectAccessReview",
				APIVersion: "authorization.k8s.io/v1",
			},
			Spec: sar.SubjectAccessReviewSpec{
				ResourceAttributes: &sar.ResourceAttributes{
					Namespace: cr.Namespace,
					Verb:      "get",
					Resource:  "ServiceAccount",
					Name:      role.ServiceAccountName,
				},
				User:   userInfo.Username,
				Groups: userInfo.Groups,
				UID:    userInfo.UID,
				Extra:  xtra,
			},
		}
		err := shared.Create(context.TODO(), sar)
		if err != nil {
			valErrs = append(valErrs, err.Error())
		} else {
			if sar.Status.Denied {
				valErrs = append(valErrs, sar.Status.Reason)
			}
		}
	}

	return valErrs
}

// validateApp function checks for valid app and if necessary creates a patch
// to populate appCatalog in the spec.
func validateApp(
	cr *kdv1.KubeDirectorCluster,
	patches []clusterPatchSpec,
) (*kdv1.KubeDirectorApp, []clusterPatchSpec, string) {

	appCR, err := catalog.FindApp(cr)

	if err != nil {
		return nil, patches,
			fmt.Sprintf(invalidAppMessage, cr.Spec.AppID)
	}

	// Note that we should NOT call shared.EnsureClusterAppReference here,
	// because K8s may yet still reject the creation of this cluster.

	// If spec.appCatalog is already populated then return.
	if cr.Spec.AppCatalog != nil {
		return appCR, patches, ""
	}

	// Generate a patch object to populate spec.appCatalog.
	var appCatalog string
	if appCR.Namespace == cr.Namespace {
		appCatalog = shared.AppCatalogLocal
	} else {
		appCatalog = shared.AppCatalogSystem
	}
	patches = append(
		patches,
		clusterPatchSpec{
			Op:   "add",
			Path: "/spec/appCatalog",
			Value: clusterPatchValue{
				ValueStr: &appCatalog,
			},
		},
	)

	return appCR, patches, ""
}

// validateMinResources function checks to see if minimum resource requirements
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

// validateFileInjections validates fileInjection spec defined for each role.
// Validation is done for the srcURL field by doing a HTTP HEAD on the url.
func validateFileInjections(
	cr *kdv1.KubeDirectorCluster,
	valErrors []string,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])
		if len(role.FileInjections) == 0 {
			// No file injections
			continue
		}
		numInjections := len(role.FileInjections)
		for j := 0; j < numInjections; j++ {
			fileInjection := role.FileInjections[j]
			srcURL := fileInjection.SrcURL

			// Validate to make sure srcURL is valid by doing a http head
			// we want to support insecure https. may be kdconfig can disallow
			// this in the future?
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
			_, headErr := client.Head(srcURL)
			if headErr != nil {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidSrcURL,
						srcURL,
						role.Name,
						headErr,
					),
				)
				continue
			}
		}
	}

	return valErrors, patches
}

// validateSecrets validates defaultSecret and individual secret field for
// each role. Validation is done to make sure secret object with the given
// name is present in the cluster CR's namespace, and that its name includes
// the required secret prefix (if any). Also if required, create a patch for
// individual role objects to populate them with the default secret.
func validateSecrets(
	cr *kdv1.KubeDirectorCluster,
	valErrors []string,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	requiredNamePrefix := shared.GetRequiredSecretPrefix()

	validateFunc := func(
		secretName string,
	) secretValidateResult {

		// First check the name against any required prefix.
		if strings.HasPrefix(secretName, requiredNamePrefix) {
			// Now also check that the secret exists in this namespace.
			_, fetchErr := observer.GetSecret(
				cr.Namespace,
				secretName,
			)
			if fetchErr != nil {
				return secretNotFound
			}
		} else {
			return secretPrefixNotMatched
		}
		return secretIsValid
	}

	defaultSecret := cr.Spec.DefaultSecret
	if defaultSecret != nil {
		// Validate the default secret, and return early if there are errors.
		defaultSecretValidateResult := validateFunc(defaultSecret.Name)
		if defaultSecretValidateResult == secretPrefixNotMatched {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					invalidDefaultSecretPrefix,
					defaultSecret.Name,
					requiredNamePrefix,
				),
			)
			return valErrors, patches
		}
		if defaultSecretValidateResult == secretNotFound {
			valErrors = append(
				valErrors,
				fmt.Sprintf(
					invalidDefaultSecret,
					defaultSecret.Name,
					cr.Namespace,
				),
			)
			return valErrors, patches
		}
	}

	// Now also validate any role-specific secrets, and also handle populating
	// unspecified ones with the default (if any).
	numRoles := len(cr.Spec.Roles)
	for i := 0; i < numRoles; i++ {
		role := &(cr.Spec.Roles[i])

		if role.Secret != nil {
			secretValidateResult := validateFunc(role.Secret.Name)
			if secretValidateResult == secretPrefixNotMatched {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidSecretPrefix,
						role.Secret.Name,
						role.Name,
						requiredNamePrefix,
					),
				)
				continue
			}
			if secretValidateResult == secretNotFound {
				valErrors = append(
					valErrors,
					fmt.Sprintf(
						invalidSecret,
						role.Secret.Name,
						role.Name,
						requiredNamePrefix,
					),
				)
				continue
			}
		}

		// If there is a defaultSecret, use that for this role (if not specified)
		if role.Secret == nil && cr.Spec.DefaultSecret != nil {
			patches = append(
				patches,
				clusterPatchSpec{
					Op:   "add",
					Path: "/spec/roles/" + strconv.Itoa(i) + "/secret",
					Value: clusterPatchValue{
						ValueKDSecret: defaultSecret,
					},
				},
			)
		}
	}

	return valErrors, patches
}

// encryptSecretKeys encrypts secret keys per each role and generates patches if needed
func encryptSecretKeys(
	cr *kdv1.KubeDirectorCluster,
	prevCr *kdv1.KubeDirectorCluster,
	valErrors []string,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {
	for roleIndex, role := range cr.Spec.Roles {
		prevEncryptedValues := map[string]string{}
		for _, prevRole := range prevCr.Spec.Roles {
			if prevRole.Name == role.Name {
				for _, secretKey := range prevRole.SecretKeys {
					prevEncryptedValues[secretKey.Name] = secretKey.EncryptedValue
				}
			}
		}
		for secretKeyIndex, secretKey := range role.SecretKeys {
			if secretKey.Value == "" && secretKey.EncryptedValue != "" {
				if secretKey.EncryptedValue != prevEncryptedValues[secretKey.Name] {
					valErrors = append(valErrors,
						fmt.Sprintf(forbiddenManualSecretKeyEncryptedValuePlacement, secretKey.Name),
					)
				}
				continue
			}
			encryptedValue, err := secretkeys.Encrypt(secretKey.Value)
			if err != nil {
				validatorLog.Error(err, "cannot encrypt secret key",
					"namespace", cr.Namespace, "kdcluster", cr.Name,
					"role", role.Name,
					"secret key", secretKeyIndex)
				valErrors = append(valErrors,
					fmt.Sprintf(failedSecretKeyEncryption, secretKey.Name),
				)
				continue
			}
			patches = append(patches, clusterPatchSpec{
				Op:   "replace",
				Path: "/spec/roles/" + strconv.Itoa(roleIndex) + "/secretKeys/" + strconv.Itoa(secretKeyIndex),
				Value: clusterPatchValue{
					ValueSecretKey: &kdv1.SecretKey{
						Name:           secretKey.Name,
						EncryptedValue: encryptedValue,
					},
				},
			})
		}
	}

	return valErrors, patches
}

// addServiceType function checks to see if serviceType is provided for a
// cluster CR. If unspecified, check to see if there is a default serviceType
// provided through kubedirector's config CR, otherwise use a global constant
// for service type. In either of those cases add an entry to PATCH spec for mutating
// cluster CR.
func addServiceType(
	cr *kdv1.KubeDirectorCluster,
	valErrors []string,
	patches []clusterPatchSpec,
) ([]string, []clusterPatchSpec) {

	if cr.Spec.ServiceType != nil {
		return valErrors, patches
	}

	serviceType := shared.GetDefaultServiceType()
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

	return valErrors, patches
}

// If the naming scheme is unspecified, use a global constant for the naming scheme.
func validateNamingScheme(
	cr *kdv1.KubeDirectorCluster,
	appCR *kdv1.KubeDirectorApp,
	patches []clusterPatchSpec,
) []clusterPatchSpec {

	if cr.Spec.NamingScheme == nil {
		namingScheme := shared.GetDefaultNamingScheme()
		cr.Spec.NamingScheme = &namingScheme
		patches = append(
			patches,
			clusterPatchSpec{
				Op:   "add",
				Path: "/spec/namingScheme",
				Value: clusterPatchValue{
					ValueStr: cr.Spec.NamingScheme,
				},
			},
		)
	}

	return patches
}

// addRestoreLabel adds the kubedirector.hpe.com/restoring label (with no value)
// to the CR's set of labels, if it's not already there.
func addRestoreLabel(
	cr *kdv1.KubeDirectorCluster,
	patches []clusterPatchSpec,
) []clusterPatchSpec {

	if len(cr.Labels) == 0 {
		labelKeyAndVal := make(dictValue)
		labelKeyAndVal[shared.RestoringLabel] = ""
		patches = append(
			patches,
			clusterPatchSpec{
				Op:   "add",
				Path: "/metadata/labels",
				Value: clusterPatchValue{
					ValueDict: &labelKeyAndVal,
				},
			},
		)
	} else {
		if _, ok := cr.Labels[shared.RestoringLabel]; !ok {
			labelVal := ""
			patches = append(
				patches,
				clusterPatchSpec{
					Op: "add",
					Path: fmt.Sprintf("/metadata/labels/%s",
						strings.ReplaceAll(shared.RestoringLabel, "/", "~1")),
					Value: clusterPatchValue{
						ValueStr: &labelVal,
					},
				},
			)
		}
	}

	return patches
}

// admitClusterCR is the top-level cluster validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitClusterCR(
	ar *v1beta1.AdmissionReview,
) *v1beta1.AdmissionResponse {

	var valErrors []string
	var patches []clusterPatchSpec
	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// Set a defer func to handle any patches and errors. Set the admission
	// response to allowed=true if no errors.
	defer func() {
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
	}()

	// We'll need the existing object in update and delete cases.
	prevClusterCR := kdv1.KubeDirectorCluster{}
	if (ar.Request.Operation == v1beta1.Update) || (ar.Request.Operation == v1beta1.Delete) {
		prevRaw := ar.Request.OldObject.Raw
		if prevJSONErr := json.Unmarshal(prevRaw, &prevClusterCR); prevJSONErr != nil {
			valErrors = append(valErrors, prevJSONErr.Error())
			return &admitResponse
		}
	}

	// Record whether we're in "restoring" state.
	_, isRestoring := prevClusterCR.Labels[shared.RestoringLabel]

	// If this is a delete and the being-restored label is set, reject the
	// deletion unless allow-delete-while-restoring is also set. In all other
	// cases allow the deletion.
	if ar.Request.Operation == v1beta1.Delete {
		if !isRestoring {
			return &admitResponse
		}
		if _, allowDelete := prevClusterCR.Labels[allowDeleteLabel]; allowDelete {
			return &admitResponse
		}
		valErrors = append(
			valErrors,
			"delete not allowed while "+shared.RestoringLabel+" label exists, "+
				"unless "+allowDeleteLabel+" label also exists",
		)
		return &admitResponse
	}

	// Deserialize the incoming object.
	raw := ar.Request.Object.Raw
	clusterCR := kdv1.KubeDirectorCluster{}
	if jsonErr := json.Unmarshal(raw, &clusterCR); jsonErr != nil {
		valErrors = append(valErrors, jsonErr.Error())
		return &admitResponse
	}

	// If this is a re-creation (as indicated by annotation existing), set
	// the being-restored label and skip validation.
	if ar.Request.Operation == v1beta1.Create {
		if clusterCR.Annotations != nil {
			if _, ok := clusterCR.Annotations[shared.StatusBackupAnnotation]; ok {
				patches = addRestoreLabel(&clusterCR, patches)
				return &admitResponse
			}
		}
	}

	// If this is an update and the being-restored label is set, don't allow
	// any spec change.
	if ar.Request.Operation == v1beta1.Update {
		if isRestoring {
			if !equality.Semantic.DeepEqual(clusterCR.Spec, prevClusterCR.Spec) {
				valErrors = append(
					valErrors,
					"spec changes not allowed while "+shared.RestoringLabel+" label exists",
				)
				return &admitResponse
			}
		}
	}

	// Don't allow Status to be updated except by KubeDirector. Do this by
	// using one-time codes known by KubeDirector.
	if clusterCR.Status != nil {
		expectedStatusGen, ok := kubedirectorcluster.ClusterStatusGens.ReadStatusGen(clusterCR.UID)
		// Reject this write if either of:
		// - this status generation UID is not what we're expecting a write for
		// - KubeDirector doesn't know about the CR & the status is changing
		statusViolation := "KubeDirector-related status properties are read-only"
		if ok {
			if clusterCR.Status.GenerationUID != expectedStatusGen.UID {
				valErrors = append(valErrors, statusViolation)
				return &admitResponse
			}
		} else {
			if !equality.Semantic.DeepEqual(clusterCR.Status, prevClusterCR.Status) {
				valErrors = append(valErrors, statusViolation)
				return &admitResponse
			}
		}
		// If this status generation UID has already been admitted previously,
		// it's OK to write the status again as long as nothing is changing.
		// (For example we'll see this when a PATCH happens.)
		if expectedStatusGen.Validated {
			if !equality.Semantic.DeepEqual(clusterCR.Status, prevClusterCR.Status) {
				valErrors = append(valErrors, statusViolation)
				return &admitResponse
			}
		}
	}

	kubedirectorcluster.ClusterStatusGens.ValidateStatusGen(clusterCR.UID)

	// Shortcut out of here if the spec is not being changed. Among other
	// things this allows KD to update status or metadata even if the
	// referenced app is bad/gone. Note that we can't just check the
	// metadata generation number here because that is incremented after this
	// validator sees the request.
	// We will NOT take this shortcut if we're trying to change from
	// "restoring" to "reconciling". Need to validate in that case.
	if ar.Request.Operation == v1beta1.Update {
		doShortcut := true
		if isRestoring {
			_, willStillBeRestoring := clusterCR.Labels[shared.RestoringLabel]
			if !willStillBeRestoring {
				doShortcut = false
			}
		}
		if doShortcut {
			if equality.Semantic.DeepEqual(clusterCR.Spec, prevClusterCR.Spec) {
				return &admitResponse
			}
		}
	}

	// At this point, if app is bad, no need to continue with validation.
	appCR, patches, errorMsg := validateApp(&clusterCR, patches)

	// If app error, fail right away
	if appCR == nil {
		valErrors = append(valErrors, errorMsg)
		return &admitResponse
	}

	// Validate that it's OK to change the spec. Note that this check assumes
	// that the above "shortcut" is in place, i.e. we are only calling this
	// if the spec is changing.
	if ar.Request.Operation == v1beta1.Update {
		valErrors, patches = validateSpecChange(&clusterCR, &prevClusterCR, valErrors, patches)
	}

	// Validate cardinality and generate patches for defaults members values.
	valErrors, patches = validateCardinality(&clusterCR, appCR, valErrors, patches)

	// Validate that roles are known & sufficient.
	valErrors = validateClusterRoles(&clusterCR, appCR, valErrors)

	// Validate minimum resources for all roles
	valErrors = validateMinResources(&clusterCR, appCR, valErrors)

	// Validate if the role's service account exists and if the user has permission to use
	valErrors = validateRoleServiceAccount(&clusterCR, valErrors, ar.Request.UserInfo)

	valErrors, patches = validateRoleStorageClass(&clusterCR, valErrors, patches)

	// Validate service type and generate patch in case no service type defined or change
	valErrors, patches = addServiceType(&clusterCR, valErrors, patches)

	// Validate naming scheme and generate patch in case no naming scheme defined or change
	patches = validateNamingScheme(&clusterCR, appCR, patches)

	// Validate file injections and generate patches for default values (if any)
	valErrors, patches = validateFileInjections(&clusterCR, valErrors, patches)

	// Validate secret and generate patches for default values (if any)
	valErrors, patches = validateSecrets(&clusterCR, valErrors, patches)

	// Generate patches to conceal raw secret keys' values
	valErrors, patches = encryptSecretKeys(&clusterCR, &prevClusterCR, valErrors, patches)

	// If cluster already exists, check for invalid property changes.
	if ar.Request.Operation == v1beta1.Update {
		var changeErrors []string
		changeErrors = validateGeneralClusterChanges(&clusterCR, &prevClusterCR, changeErrors)
		changeErrors = validateRoleChanges(&clusterCR, &prevClusterCR, changeErrors)
		// If un-change-able properties are being changed, ignore all other error
		// messages in favor of those. (The reason we didn't just do this check
		// first and then skip other validation is because this check may
		// depend on the defaulting logic that happens in those other functions.)
		if len(changeErrors) != 0 {
			valErrors = changeErrors
		}
	}

	return &admitResponse
}
