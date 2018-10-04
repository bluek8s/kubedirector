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

package reconciler

import (
	"reflect"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/google/uuid"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// syncCluster runs the reconciliation logic. It is invoked because of a
// change in or addition of a KubeDirectorCluster instance, or a periodic
// polling to check on such a resource.
func syncCluster(
	event sdk.Event,
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
) error {

	// Exit early if deleting the resource... nothing else for us to do.
	if event.Deleted {
		shared.LogInfo(
			cr,
			"deleted",
		)
		DeleteStatusGen(cr, handlerState)
		return nil
	}

	// Make sure we have a Status object to work with.
	if cr.Status == nil {
		cr.Status = &kdv1.ClusterStatus{}
		cr.Status.Roles = make([]kdv1.RoleStatus, 0)
	}

	// Set up logic to update status as necessary when handler exits.
	oldStatus := cr.Status.DeepCopy()
	defer func() {
		if !reflect.DeepEqual(cr.Status, oldStatus) {
			// Write back the status. Don't exit this handler until we
			// succeed (will block other handlers for this resource).
			updated := false
			for updated == false {
				cr.Status.GenerationUid = uuid.New().String()
				WriteStatusGen(cr, handlerState, cr.Status.GenerationUid)
				updateErr := executor.UpdateStatus(cr)
				if updateErr == nil {
					updated = true
				} else {
					time.Sleep(10 * time.Second)
				}
			}
		}
	}()

	// Ignore stale poll-driven handler for a resource we have since
	// updated. Also for a new CR just update the status state/gen and return.
	if !handleStatusGen(cr, handlerState) {
		return nil
	}

	// We use a finalizer to prevent races between polled CR updates and
	// CR deletion.
	doExit, finalizerErr := handleFinalizers(cr)
	if finalizerErr != nil {
		return finalizerErr
	}
	if doExit {
		return nil
	}

	errLog := func(domain string, err error) {
		shared.LogErrorf(
			cr,
			"failed to sync %s: %v",
			domain,
			err,
		)
	}

	clusterServiceErr := syncClusterService(cr)
	if clusterServiceErr != nil {
		errLog("cluster service", clusterServiceErr)
		return clusterServiceErr
	}

	roles, state, rolesErr := syncRoles(cr)
	if rolesErr != nil {
		errLog("roles", rolesErr)
		return rolesErr
	}

	memberServicesErr := syncMemberServices(cr, roles)
	if memberServicesErr != nil {
		errLog("member services", memberServicesErr)
		return memberServicesErr
	}

	if state == clusterMembersStableReady {
		if cr.Status.State != string(clusterReady) {
			shared.LogInfo(
				cr,
				"stable",
			)
			cr.Status.State = string(clusterReady)
		}
		return nil
	} else if state == clusterMembersError {
		if cr.Status.State != string(clusterWarning) {
			shared.LogInfo(
				cr,
				"error",
			)
			cr.Status.State = string(clusterWarning)
		}
		return nil
	} else {
		if cr.Status.State != string(clusterCreating) {
			cr.Status.State = string(clusterUpdating)
		}
	}

	configmetaGen, configMetaErr := catalog.ConfigmetaGenerator(
		cr,
		calcMemberNamesForRoles(roles),
	)
	if configMetaErr != nil {
		shared.LogErrorf(
			cr,
			"failed to generate cluster config: %v",
			configMetaErr,
		)
		return configMetaErr
	}

	membersHaveChanged := (state == clusterMembersChangedUnready)
	membersErr := syncMembers(cr, roles, membersHaveChanged, configmetaGen)
	if membersErr != nil {
		errLog("members", membersErr)
		return membersErr
	}

	return nil
}

// handleStatusGen compares the incoming status generation to its last known
// value. If there is no last known value, this is either a new CR or one that
// was created before this KD came up. In the former case, set the cluster
// state to creating and return false. In the latter case, figure out the
// current state of the CR. Otherwise (if there IS a last known value), we
// want to return true if the incoming gen number is expected; return false to
// reject old/stale versions of the CR.
func handleStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
) bool {

	incoming := cr.Status.GenerationUid
	lastKnown, ok := ReadStatusGen(cr, handlerState)
	if !ok {
		if incoming == "" {
			shared.LogInfo(
				cr,
				"new",
			)
			cr.Status.State = string(clusterCreating)
			return false
		}
		shared.LogWarnf(
			cr,
			"unknown with incoming gen uid %s",
			incoming,
		)
		WriteStatusGen(cr, handlerState, incoming)
		ValidateStatusGen(cr, handlerState)
		return true
	}

	if lastKnown.Uid == incoming {
		return true
	}

	shared.LogInfo(
		cr,
		"dropping stale poll",
	)
	return false
}

// handleFinalizers will remove our finalizer if deletion has been requested.
// Otherwise it will add our finalizer if it is absent.
func handleFinalizers(
	cr *kdv1.KubeDirectorCluster,
) (bool, error) {

	if cr.DeletionTimestamp != nil {
		// If a deletion has been requested, while ours (or other) finalizers
		// existed on the CR, go ahead and remove our finalizer.
		removeErr := executor.RemoveFinalizer(cr)
		if removeErr == nil {
			shared.LogInfo(
				cr,
				"greenlighting for deletion",
			)
		}
		return true, removeErr
	} else {
		// If our finalizer doesn't exist on the CR, put it in there.
		ensureErr := executor.EnsureFinalizer(cr)
		if ensureErr != nil {
			return true, ensureErr
		} else {
			return false, nil
		}
	}
}

// calcMemberNamesForRoles generates a map of role name to list of all member
// names the role that are intended to exist -- i.e. members in states
// memberCreatePending, memberCreating, memberReady or memberError
func calcMemberNamesForRoles(
	roles []*roleInfo,
) map[string][]string {

	result := make(map[string][]string)
	for _, roleInfo := range roles {
		if roleInfo.roleSpec != nil {
			membersStatus := append(
				append(
					append(
						roleInfo.membersByState[memberCreatePending],
						roleInfo.membersByState[memberCreating]...,
					),
					roleInfo.membersByState[memberReady]...,
				),
				roleInfo.membersByState[memberError]...,
			)
			var memberNamesForRole []string
			for _, member := range membersStatus {
				memberNamesForRole = append(memberNamesForRole, member.Pod)
			}
			result[roleInfo.roleSpec.Name] = memberNamesForRole
		}
	}
	return result
}
