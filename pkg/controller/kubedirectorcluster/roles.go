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

package kubedirectorcluster

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/go-logr/logr"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/apimachinery/pkg/api/errors"
)

// syncClusterRoles is responsible for dealing with roles being changed, added,
// or removed. It is the only function in this file that is invoked from another
// file (from the syncCluster function in cluster.go). Managing role changes
// may result in operations on k8s statefulsets. This function will also
// modify the role status data structures, and create a role info slice that
// can be referenced by the later syncs for other concerns.
func syncClusterRoles(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) ([]*roleInfo, clusterStateInternal, error) {

	// Construct the role info slice. Bail out now if that fails.
	roles, rolesErr := initRoleInfo(reqLogger, cr)
	if rolesErr != nil {
		return nil, clusterMembersUnknown, rolesErr
	}

	// Role changes will be postponed if any members are currently in the
	// creating state. Such members may have been informed of the current
	// member set, and they are not yet ready to receive updates about
	// changes to the member set.

	for _, r := range roles {
		if len(r.membersByState[memberCreating]) != 0 {
			return roles, clusterMembersStableUnready, nil
		}
	}

	// Assume cluster is stable until found otherwise.
	allMembersReady := true
	anyMembersChanged := false

	// Reconcile each role as necessary.
	for _, r := range roles {
		switch {
		case r.statefulSet == nil && r.roleStatus == nil:
			// Role did not previously exist. Create it now.
			createErr := handleRoleCreate(
				reqLogger, cr, r, &anyMembersChanged)
			if createErr != nil {
				return nil, clusterMembersUnknown, createErr
			}
		case r.statefulSet == nil && r.roleStatus != nil:
			// Role exists but there is no statefulset for it in k8s.
			// Hmm, weird. Statefulset was deleted out-of-band? Let's fix.
			reCreateErr := handleRoleReCreate(reqLogger, cr, r, &anyMembersChanged)
			if reCreateErr != nil {
				return nil, clusterMembersUnknown, reCreateErr
			}
		case r.statefulSet != nil && r.roleStatus != nil:
			// Deal with an existing role and statefulset.
			// First see if we need to reconcile any out-of-band statefulset
			// changes.
			handleRoleConfig(reqLogger, cr, r)
			// Now check for desired changes in role population.
			if len(r.roleStatus.Members) == 0 && r.desiredPop == 0 {
				// Role is going away and we have finished removing pods.
				handleRoleDelete(reqLogger, cr, r)
			} else {
				// Might need to change role population.
				handleRoleResize(reqLogger, cr, r, &anyMembersChanged)
			}
		case r.statefulSet != nil && r.roleStatus == nil:
			// "Can't happen" ... there should be no way to find the
			// statefulset unless we have a role status.
			panicMsg := fmt.Sprintf(
				"StatefulSet{%s} for KubeDirectorCluster{%s/%s} has no role status",
				r.statefulSet.Name,
				cr.Namespace,
				cr.Name,
			)
			panic(panicMsg)
		}
		if !allRoleMembersReadyOrError(cr, r) {
			allMembersReady = false
		}
	}
	// Let the caller know about significant changes that happened.
	var returnState clusterStateInternal
	if anyMembersChanged {
		returnState = clusterMembersChangedUnready
	} else {
		if allMembersReady {
			returnState = clusterMembersStableReady
		} else {
			returnState = clusterMembersStableUnready
		}
	}

	return roles, returnState, nil
}

// initRoleInfo constructs a slice of elements representing all current or
// desired roles. Each element contains useful information about the role
// spec and status that will be used not only in syncRole but also by the
// sync logic for other concerns.
func initRoleInfo(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) ([]*roleInfo, error) {

	roles := make(map[string]*roleInfo)
	numRoleSpecs := len(cr.Spec.Roles)
	numRoleStatuses := len(cr.Status.Roles)

	// Capture the desired roles and member count in the spec. The fields
	// statefulSet, roleStatus, and membersByState may be populated later
	// in this function.
	for i := 0; i < numRoleSpecs; i++ {
		roleSpec := &(cr.Spec.Roles[i])
		roles[roleSpec.Name] = &roleInfo{
			statefulSet:    nil,
			roleSpec:       roleSpec,
			roleStatus:     nil,
			membersByState: make(map[memberState][]*kdv1.MemberStatus),
			desiredPop:     int(*(roleSpec.Members)),
		}
	}

	// We're about to start grabbing pointers into the role status slice,
	// and we may have to add to that slice later. So let's grow its capacity
	// now. We know that at most we will need to add a number of role statuses
	// equal to the number of role specs.
	newRoleStatuses := make(
		[]kdv1.RoleStatus,
		numRoleStatuses,
		numRoleStatuses+numRoleSpecs,
	)
	copy(newRoleStatuses, cr.Status.Roles)
	cr.Status.Roles = newRoleStatuses

	// Now look at the existing roles we have status for. Update or add to
	// the role info accordingly.
	for i := 0; i < numRoleStatuses; i++ {
		roleStatus := &(cr.Status.Roles[i])
		statefulSet, statefulSetErr := observer.GetStatefulSet(
			cr.Namespace,
			roleStatus.StatefulSet,
		)
		if statefulSetErr != nil {
			if errors.IsNotFound(statefulSetErr) {
				statefulSet = nil
			} else {
				shared.LogErrorf(
					reqLogger,
					statefulSetErr,
					cr,
					shared.EventReasonRole,
					"failed to query StatefulSet{%s} for role{%s}",
					roleStatus.StatefulSet,
					roleStatus.Name,
				)
				return nil, statefulSetErr
			}
		}
		if role, ok := roles[roleStatus.Name]; ok {
			// This role is in the spec. Update the roleinfo with the
			// statefulset pointer (if any) and the role status pointer.
			role.statefulSet = statefulSet
			role.roleStatus = roleStatus
			// If we might add to the role status members slice later,
			// increase its capacity. Similarly to the overall role status
			// slice, we want to make sure we can have stable pointers into
			// this slice.
			numMembers := len(roleStatus.Members)
			if role.desiredPop > numMembers {
				newMembers := make(
					[]kdv1.MemberStatus,
					numMembers,
					role.desiredPop,
				)
				copy(newMembers, roleStatus.Members)
				roleStatus.Members = newMembers
			}
		} else {
			// This is not a role desired in the spec. Create a new info
			// entry with desired member count at zero.
			roles[roleStatus.Name] = &roleInfo{
				statefulSet:    statefulSet,
				roleSpec:       nil,
				roleStatus:     roleStatus,
				membersByState: make(map[memberState][]*kdv1.MemberStatus),
				desiredPop:     0,
			}
		}
	}

	// Return a slice of roleinfo made from the map values, and with the
	// membersByState maps populated.
	var result []*roleInfo
	for _, info := range roles {
		calcRoleMembersByState(info)
		result = append(result, info)
	}
	return result, nil
}

// calcRoleMembersByState builds the members-by-state map based on the current
// member statuses in the role.
func calcRoleMembersByState(
	role *roleInfo,
) {

	if role.roleStatus == nil {
		return
	}
	numMembers := len(role.roleStatus.Members)
	for i := 0; i < numMembers; i++ {
		member := &(role.roleStatus.Members[i])
		role.membersByState[memberState(member.State)] = append(
			role.membersByState[memberState(member.State)],
			member)
	}
}

// handleRoleCreate deals with a newly specified role. If the desired population
// is nonzero then it will create an associated statefulset and create the
// role status and its member statuses (initially as create pending). Failure
// to create a statefulset will be a reconciler-stopping error.
func handleRoleCreate(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	anyMembersChanged *bool,
) error {

	if role.desiredPop == 0 {
		// Nothing to do if zero members desired... we won't even create
		// the statefulset or the role status.
		return nil
	}

	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonRole,
		"creating role{%s}",
		role.roleSpec.Name,
	)

	nativeSystemdSupport := shared.GetNativeSystemdSupport()

	// Create the associated statefulset.
	statefulSet, createErr := executor.CreateStatefulSet(
		reqLogger,
		cr,
		nativeSystemdSupport,
		role.roleSpec,
	)
	if createErr != nil {
		// Not much to do if we can't create it... we'll just keep trying
		// on every run through the reconciler.
		shared.LogErrorf(
			reqLogger,
			createErr,
			cr,
			shared.EventReasonRole,
			"failed to create StatefulSet for role{%s}",
			role.roleSpec.Name,
		)
		return createErr
	}

	// OK we have the statefulset, so set up the role and member status.
	*anyMembersChanged = true
	role.statefulSet = statefulSet
	if role.roleStatus == nil {
		newRoleStatus := kdv1.RoleStatus{
			Name:        role.roleSpec.Name,
			StatefulSet: statefulSet.Name,
			Members:     make([]kdv1.MemberStatus, 0, role.desiredPop),
		}
		// cr.Status.Roles was created with enough capacity to avoid
		// realloc, so we can safely grow it w/o disturbing our
		// pointers to its elements.
		cr.Status.Roles = append(cr.Status.Roles, newRoleStatus)
		role.roleStatus = &(cr.Status.Roles[len(cr.Status.Roles)-1])
	}
	addMemberStatuses(cr, role)
	return nil
}

// handleRoleCreate deals with the unusual-but-possible case of the role
// status existing but the statefulset gone missing. It may need to clean up
// the role status or re-create the statefulset. Failure to create a
// statefulset will be a reconciler-stopping error.
func handleRoleReCreate(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	anyMembersChanged *bool,
) error {

	if len(role.roleStatus.Members) == 0 {
		// No lingering pod status to deal with.
		if role.desiredPop == 0 {
			// Looks like the role should be gone anyway, so mark it for removal.
			role.roleStatus.StatefulSet = ""
		} else {
			// Create a new statefulset for the role.
			return handleRoleCreate(reqLogger, cr, role, anyMembersChanged)
		}
	} else {
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonRole,
			"restoring role{%s}",
			role.roleStatus.Name,
		)
		*anyMembersChanged = true
		// Need to clean up from the old status before we make a new
		// statefulset. For any pods that had reached "ready" or "error" state,
		// we should mark them first as "delete pending" -- we know we have notified
		// other pods of these creations, so we should now notify of deletions.
		// For other pods we can just move them straight into deleting state.
		numMembers := len(role.roleStatus.Members)
		for i := 0; i < numMembers; i++ {
			member := &(role.roleStatus.Members[i])
			switch memberState(member.State) {
			case memberDeletePending:
				fallthrough
			case memberReady:
				fallthrough
			case memberConfigError:
				member.State = string(memberDeletePending)
			default:
				member.State = string(memberDeleting)
			}
		}
		// This should be quite unusual so we won't try to be clever about
		// updating the membersByState map. Just nuke and re-create it.
		role.membersByState = make(map[memberState][]*kdv1.MemberStatus)
		calcRoleMembersByState(role)
	}
	return nil
}

// handleRoleConfig checks an existing statefulset to see if any of its
// important properties (other than replicas count) need to be reconciled.
// Failure to reconcile will not be treated as a reconciler-stopping error; we'll
// just try again next time.
func handleRoleConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	updateErr := executor.UpdateStatefulSetNonReplicas(
		cr,
		role.roleSpec,
		role.statefulSet)
	if updateErr != nil {
		shared.LogErrorf(
			reqLogger,
			updateErr,
			cr,
			shared.EventReasonRole,
			"failed to update StatefulSet{%s}",
			role.statefulSet.Name,
		)
	}
}

// handleRoleDelete takes care of deleting the associated statefulset after
// the role members have been cleaned up. Failure to delete will not be
// treated as a reconciler-stopping error; we'll just try again next time.
func handleRoleDelete(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonRole,
		"finishing cleanup on role{%s}",
		role.roleStatus.Name,
	)
	deleteErr := executor.DeleteStatefulSet(cr.Namespace, role.statefulSet.Name)
	if deleteErr == nil || errors.IsNotFound(deleteErr) {
		// Mark the role status for removal.
		role.roleStatus.StatefulSet = ""
	} else {
		shared.LogErrorf(
			reqLogger,
			deleteErr,
			cr,
			shared.EventReasonRole,
			"failed to delete StatefulSet{%s}",
			role.statefulSet.Name,
		)
	}
}

// handleRoleResize deals with roles that already have corresponding
// statefulsets in k8s. If the desired population is different than the
// current member count it may need adjust the role/member status to start
// the resize process.
func handleRoleResize(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	anyMembersChanged *bool,
) {

	// We won't even be attempting a resize if there are any creating-state
	// members, so the current set of "requested members" is just ready plus
	// create pending.
	prevDesiredPop :=
		len(role.membersByState[memberReady]) +
			len(role.membersByState[memberConfigError]) +
			len(role.membersByState[memberCreatePending])
	if role.desiredPop == prevDesiredPop {
		return
	}
	if role.desiredPop > prevDesiredPop {
		// Only expand if no members are in delete pending or deleting states;
		// we can't use expand to "rescue" a member that is currently being
		// deleted. (The way statefulsets reuse FQDNs, we might be able to get
		// away with that actually, but let's not complicate things.)
		if len(role.roleStatus.Members) == prevDesiredPop {
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonRole,
				"expanding role{%s}",
				role.roleStatus.Name,
			)
			*anyMembersChanged = true
			addMemberStatuses(cr, role)
		}
	} else {
		// We can shrink in any state. This is a helpful thing to allow when
		// the expand was overambitious and is waiting for resources.
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonRole,
			"shrinking role{%s}",
			role.roleStatus.Name,
		)
		*anyMembersChanged = true
		deleteMemberStatuses(role)
	}
}

// addMemberStatuses adds member statuses to a role, in create pending state,
// to bring it up to the desired number of members. It also updates the
// members-by-state map accordingly.
func addMemberStatuses(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	lastNodeID := &cr.Status.LastNodeID
	currentPop := len(role.roleStatus.Members)
	for i := currentPop; i < role.desiredPop; i++ {
		indexString := strconv.Itoa(i)
		// Pod name and PVC name will be generated by K8s in a predictable
		// way, so go ahead and populate those here.
		memberName := role.roleStatus.StatefulSet + "-" + indexString
		var pvcName string
		if role.roleSpec.Storage == nil {
			pvcName = ""
		} else {
			pvcName = executor.PvcNamePrefix + "-" + memberName
		}
		// role.roleStatus.Members was created with enough capacity to
		// avoid realloc, so we can safely grow it w/o disturbing our
		// pointers to its elements.
		role.roleStatus.Members = append(
			role.roleStatus.Members,
			kdv1.MemberStatus{
				Pod:     memberName,
				Service: "",
				PVC:     pvcName,
				NodeID:  atomic.AddInt64(lastNodeID, 1),
				State:   string(memberCreatePending),
			},
		)
		role.membersByState[memberCreatePending] = append(
			role.membersByState[memberCreatePending],
			&(role.roleStatus.Members[i]))
	}
}

// deleteMemberStatuses changes member statuses in a role by moving them from
// to delete pending state (if currently ready) or deleting state (if currently
// create pending or creating), to prepare to shrink the role to the desired
// number of members. It also updates the members-by-state map accordingly.
func deleteMemberStatuses(
	role *roleInfo,
) {

	currentPop := len(role.roleStatus.Members)
	createPendingPop := len(role.membersByState[memberCreatePending])
	readyPop := len(role.membersByState[memberReady])
	errorPop := len(role.membersByState[memberConfigError])
	// Don't need to worry about creating-state members, since if any existed
	// we wouldn't be able to make role changes.
	for i := role.desiredPop; i < currentPop; i++ {
		member := &(role.roleStatus.Members[i])
		switch memberState(member.State) {
		case memberCreatePending:
			member.State = string(memberDeleting)
			role.membersByState[memberDeleting] = append(
				role.membersByState[memberDeleting],
				member,
			)
			createPendingPop--
		case memberReady:
			member.State = string(memberDeletePending)
			role.membersByState[memberDeletePending] = append(
				role.membersByState[memberDeletePending],
				member,
			)
			readyPop--
		case memberConfigError:
			member.State = string(memberDeletePending)
			role.membersByState[memberDeletePending] = append(
				role.membersByState[memberDeletePending],
				member,
			)
			errorPop--
		default:
		}
	}
	if createPendingPop > 0 {
		role.membersByState[memberCreatePending] =
			role.membersByState[memberCreatePending][:createPendingPop]
	} else {
		delete(role.membersByState, memberCreatePending)
	}
	if readyPop > 0 {
		role.membersByState[memberReady] =
			role.membersByState[memberReady][:readyPop]
	} else {
		delete(role.membersByState, memberReady)
	}
	if errorPop > 0 {
		role.membersByState[memberConfigError] =
			role.membersByState[memberConfigError][:errorPop]
	} else {
		delete(role.membersByState, memberConfigError)
	}
}

// allRoleMembersReadyOrError examines the members-by-state map and returns
// whether all existing members are in the ready-state or error-state bucket.
// (The situation of "no members" will also return true.) Ready members are
// also checked to make sure they have processed all updates.
func allRoleMembersReadyOrError(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) bool {

	switch len(role.membersByState) {
	case 0:
		return true
	default:
		for state, members := range role.membersByState {
			if state != memberReady && state != memberConfigError {
				return false
			}
			if state == memberReady {
				for _, m := range members {
					if (m.StateDetail.LastSetupGeneration != nil) &&
						(cr.Status.SpecGenerationToProcess != nil) {
						if *m.StateDetail.LastSetupGeneration !=
							*cr.Status.SpecGenerationToProcess {
							return false
						}
					}
					if len(m.StateDetail.PendingNotifyCmds) != 0 {
						return false
					}
				}
			}
		}
		return true
	}
}
