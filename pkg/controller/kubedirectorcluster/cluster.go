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

package kubedirectorcluster

import (
	"github.com/go-logr/logr"
	"reflect"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
)

// syncCluster runs the reconciliation logic. It is invoked because of a
// change in or addition of a KubeDirectorCluster instance, or a periodic
// polling to check on such a resource.
func (r *ReconcileKubeDirectorCluster) syncCluster(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) error {

	// Make sure this cluster marks a reference to its app.
	shared.EnsureClusterAppReference(cr.Namespace, cr.Name, cr.Spec.AppID)

	// Make sure we have a Status object to work with.
	if cr.Status == nil {
		cr.Status = &kdv1.KubeDirectorClusterStatus{}
		cr.Status.Roles = make([]kdv1.RoleStatus, 0)
	}

	// Set up logic to update status as necessary when reconciler exits.
	oldStatus := cr.Status.DeepCopy()
	defer func() {
		if !reflect.DeepEqual(cr.Status, oldStatus) {
			// Write back the status. Don't exit this reconciler until we
			// succeed (will block other reconcilers for this resource).
			wait := time.Second
			maxWait := 4096 * time.Second
			for {
				cr.Status.GenerationUID = uuid.New().String()
				shared.WriteStatusGen(cr.UID, cr.Status.GenerationUID)
				updateErr := executor.UpdateStatus(reqLogger, cr)
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
				shared.LogErrorf(
					reqLogger,
					updateErr,
					cr,
					shared.EventReasonCluster,
					"trying status update again in %v; failed",
					wait,
				)
				time.Sleep(wait)
			}
		}
	}()

	// We use a finalizer to ensure that only KubeDirector updates status
	doExit, finalizerErr := r.handleFinalizers(reqLogger, cr)
	if finalizerErr != nil {
		return finalizerErr
	}
	if doExit {
		return nil
	}

	// For a new CR just update the status state/gen.
	shouldProcessCR := r.handleNewCluster(reqLogger, cr)
	if !shouldProcessCR {
		return nil
	}

	errLog := func(domain string, err error) {
		shared.LogErrorf(
			reqLogger,
			err,
			cr,
			shared.EventReasonCluster,
			"failed to sync %s",
			domain,
		)
	}

	clusterServiceErr := syncClusterService(reqLogger, cr)
	if clusterServiceErr != nil {
		errLog("cluster service", clusterServiceErr)
		return clusterServiceErr
	}

	roles, state, rolesErr := syncRoles(reqLogger, cr)
	if rolesErr != nil {
		errLog("roles", rolesErr)
		return rolesErr
	}

	memberServicesErr := syncMemberServices(reqLogger, cr, roles)
	if memberServicesErr != nil {
		errLog("member services", memberServicesErr)
		return memberServicesErr
	}

	if state == clusterMembersStableReady {
		if cr.Status.State != string(clusterReady) {
			shared.LogInfo(
				reqLogger,
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
		shared.LogError(
			reqLogger,
			configMetaErr,
			cr,
			shared.EventReasonCluster,
			"failed to generate cluster config",
		)
		return configMetaErr
	}

	membersHaveChanged := (state == clusterMembersChangedUnready)
	membersErr := syncMembers(reqLogger, cr, roles, membersHaveChanged, configmetaGen)
	if membersErr != nil {
		errLog("members", membersErr)
		return membersErr
	}

	return nil
}

// handleNewCluster looks in the cache for the last-known status generation
// UID for this CR. If there is one, return true to keep processing the CR.
// If there is not any last-known UID, this is either a new CR or one that
// was created before this KD came up. In the former case, where the CR status
// itself has no generation UID: set the cluster state to creating (this will
// also trigger population of the generation UID) and return false to cause
// this handler to exit; we'll pick up further processing in the next handler.
// In the latter case, sync up our internal state with the visible state of
// the CR and return true to continue processing.
func (r *ReconcileKubeDirectorCluster) handleNewCluster(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) bool {

	_, ok := shared.ReadStatusGen(cr.UID)
	if ok {
		return true
	}
	incoming := cr.Status.GenerationUID
	if incoming == "" {
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"new",
		)
		cr.Status.State = string(clusterCreating)
		return false
	}
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonNoEvent,
		"unknown with incoming gen uid %s",
		incoming,
	)
	shared.WriteStatusGen(cr.UID, incoming)
	shared.ValidateStatusGen(cr.UID)
	shared.EnsureClusterAppReference(cr.Namespace, cr.Name, cr.Spec.AppID)
	return true
}

// handleFinalizers will remove our finalizer if deletion has been requested.
// Otherwise it will add our finalizer if it is absent.
func (r *ReconcileKubeDirectorCluster) handleFinalizers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) (bool, error) {

	if cr.DeletionTimestamp != nil {
		// If a deletion has been requested, while ours (or other) finalizers
		// existed on the CR, go ahead and remove our finalizer.
		removeErr := executor.RemoveFinalizer(reqLogger, cr)
		if removeErr == nil {
			shared.LogInfo(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"greenlighting for deletion",
			)
		}
		// Also clear the status gen from our cache, regardless of whether
		// finalizer modification succeeded.
		shared.DeleteStatusGen(cr.UID)
		shared.RemoveClusterAppReference(cr.Namespace, cr.Name)
		return true, removeErr
	}

	// If our finalizer doesn't exist on the CR, put it in there.
	ensureErr := executor.EnsureFinalizer(reqLogger, cr)
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