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
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/google/uuid"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/api/errors"
)

// syncCluster runs the reconciliation logic. It is invoked because of a
// change in or addition of a KubeDirectorCluster instance, or a periodic
// polling to check on such a resource.
func syncCluster(
	event sdk.Event,
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) error {

	// Exit early if deleting the resource.
	if event.Deleted {
		shared.LogInfo(
			cr,
			shared.EventReasonCluster,
			"deleted",
		)
		deleteStatusGen(cr, handler)
		removeClusterAppReference(cr, handler)
		return nil
	}

	// Otherwise, make sure this cluster marks a reference to its app.
	ensureClusterAppReference(cr, handler)

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
			wait := time.Second
			maxWait := 4096 * time.Second
			for {
				cr.Status.GenerationUid = uuid.New().String()
				writeStatusGen(cr, handler, cr.Status.GenerationUid)
				updateErr := executor.UpdateStatus(cr)
				if updateErr == nil {
					return
				}
				// Update failed. If the cluster has been or is being
				// deleted, that's ok... otherwise wait and try again.
				currentCluster, currentClusterErr := observer.GetCluster(
					cr.Namespace,
					cr.Name,
				)
				if currentClusterErr != nil {
					if errors.IsNotFound(currentClusterErr) {
						return
					}
				} else {
					if currentCluster.DeletionTimestamp != nil {
						return
					}
				}
				if wait < maxWait {
					wait = wait * 2
				}
				shared.LogWarnf(
					cr,
					shared.EventReasonCluster,
					"trying status update again in %v; failed because: %v",
					wait,
					updateErr,
				)
				time.Sleep(wait)
			}
		}
	}()

	// Ignore stale poll-driven handler for a resource we have since
	// updated. Also for a new CR just update the status state/gen.
	shouldProcessCR := handleStatusGen(cr, handler)

	// Regardless of whether the status gen is as expected, make sure the CR
	// finalizers are as we want them. We use a finalizer to prevent races
	// between polled CR updates and CR deletion.
	doExit, finalizerErr := handleFinalizers(cr)
	if finalizerErr != nil {
		return finalizerErr
	}
	if doExit {
		return nil
	}

	if !shouldProcessCR {
		return nil
	}

	errLog := func(domain string, err error) {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
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

	roles, state, rolesErr := syncRoles(cr, handler)
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
				shared.EventReasonCluster,
				"stable",
			)
			cr.Status.State = string(clusterReady)
		}
		return nil
	}

	if cr.Status.State != string(clusterCreating) {
		cr.Status.State = string(clusterUpdating)
	}

	configmetaGen, configMetaErr := catalog.ConfigmetaGenerator(
		cr,
		calcMembersForRoles(roles),
	)
	if configMetaErr != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
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
	handler *Handler,
) bool {

	incoming := cr.Status.GenerationUid
	lastKnown, ok := ReadStatusGen(cr, handler)
	if !ok {
		if incoming == "" {
			shared.LogInfo(
				cr,
				shared.EventReasonCluster,
				"new",
			)
			cr.Status.State = string(clusterCreating)
			return false
		}
		shared.LogWarnf(
			cr,
			shared.EventReasonNoEvent,
			"unknown with incoming gen uid %s",
			incoming,
		)
		writeStatusGen(cr, handler, incoming)
		ValidateStatusGen(cr, handler)
		ensureClusterAppReference(cr, handler)
		return true
	}

	if lastKnown.UID == incoming {
		return true
	}

	shared.LogInfo(
		cr,
		shared.EventReasonNoEvent,
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
				shared.EventReasonCluster,
				"greenlighting for deletion",
			)
		}
		return true, removeErr
	}

	// If our finalizer doesn't exist on the CR, put it in there.
	ensureErr := executor.EnsureFinalizer(cr)
	if ensureErr != nil {
		return true, ensureErr
	}

	return false, nil
}

// calcMembersForRoles generates a map of role name to list of all member
// in the role that are intended to exist -- i.e. members in states
// memberCreatePending, memberCreating, memberReady or memberConfigError
func calcMembersForRoles(
	roles []*roleInfo,
) map[string][]*kdv1.MemberStatus {

	result := make(map[string][]*kdv1.MemberStatus)
	for _, roleInfo := range roles {
		if roleInfo.roleSpec != nil {
			var membersStatus []*kdv1.MemberStatus

			membersStatus = append(
				append(
					append(
						roleInfo.membersByState[memberCreatePending],
						roleInfo.membersByState[memberCreating]...,
					),
					roleInfo.membersByState[memberReady]...,
				),
				roleInfo.membersByState[memberConfigError]...,
			)
			result[roleInfo.roleSpec.Name] = membersStatus
		}
	}
	return result
}
