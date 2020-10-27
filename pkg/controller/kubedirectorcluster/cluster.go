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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

var (
	// ClusterStatusGens is exported so that the validator can have access.
	ClusterStatusGens = shared.NewStatusGens()
)

// syncCluster runs the reconciliation logic. It is invoked because of a
// change in or addition of a KubeDirectorCluster instance, or a periodic
// polling to check on such a resource.
func (r *ReconcileKubeDirectorCluster) syncCluster(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) error {

	// Memoize state of the incoming object.
	hadFinalizer := shared.HasFinalizer(cr)
	oldStatus := cr.Status.DeepCopy()

	// Make sure we have a Status object to work with.
	if cr.Status == nil {
		cr.Status = &kdv1.KubeDirectorClusterStatus{}
		cr.Status.Roles = make([]kdv1.RoleStatus, 0)
		if cr.Status.SpecGenerationToProcess == nil {
			initSpecGen := int64(0)
			cr.Status.SpecGenerationToProcess = &initSpecGen
		}
	}

	annotations := cr.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
		cr.Annotations = annotations

		if shared.Update(context.TODO(), cr) == nil {
			shared.LogInfo(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"Initialized Annotations and updated context",
			)
		}
	}

	// Set a defer func to write new status and/or finalizers if they change.
	defer func() {
		syncMemberNotifies(reqLogger, cr)
		updateStateRollup(cr)
		nowHasFinalizer := shared.HasFinalizer(cr)
		// Bail out if nothing has changed. Note that if we are deleting we
		// don't care if status has changed.
		statusChanged := false
		if (cr.DeletionTimestamp == nil) || nowHasFinalizer {
			statusChanged = !reflect.DeepEqual(cr.Status, oldStatus)
		}
		finalizersChanged := (hadFinalizer != nowHasFinalizer)
		if !(statusChanged || finalizersChanged) {
			return
		}
		// Write back the status. Don't exit this reconciler until we
		// succeed (will block other reconcilers for this resource).
		wait := time.Second
		maxWait := 4096 * time.Second
		for {
			// If status has changed, write it back.
			var updateErr error
			if statusChanged {
				cr.Status.GenerationUID = uuid.New().String()
				ClusterStatusGens.WriteStatusGen(cr.UID, cr.Status.GenerationUID)
				updateErr = executor.UpdateClusterStatus(cr)
				// If this succeeded, no need to do it again on next iteration
				// if we're just cycling because of a failure to update the
				// finalizer.
				if updateErr == nil {
					statusChanged = false
				}
			}
			// If any necessary status update worked, let's also update
			// finalizers if necessary. To be safe, don't include the status
			// stanza in this write.
			if (updateErr == nil) && finalizersChanged {
				// See https://github.com/bluek8s/kubedirector/issues/194
				// Migrate Client().Update() calls back to Patch() calls.
				crWithoutStatus := cr.DeepCopy()
				crWithoutStatus.Status = nil
				updateErr = shared.Update(context.TODO(), crWithoutStatus)
			}
			// Bail out if we're done.
			if updateErr == nil {
				return
			}
			// Some necessary update failed. If the cluster has been deleted,
			// that's ok... otherwise we'll try again.
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
					statusChanged = false
				}
				// If we got a conflict error, update the CR with its current
				// form, restore our desired status/finalizers, and try again
				// immediately.
				if errors.IsConflict(updateErr) {
					currentCluster.Status = cr.Status
					currentHasFinalizer := shared.HasFinalizer(currentCluster)
					if currentHasFinalizer {
						if !nowHasFinalizer {
							shared.RemoveFinalizer(currentCluster)
						}
					} else {
						if nowHasFinalizer {
							shared.EnsureFinalizer(currentCluster)
						}
					}
					*cr = *currentCluster
					continue
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
	}()
	// Calculate md5check sum to generate unique hash for connection object
	currentHash := calcConnectionsHash(&cr.Spec.Connections, cr.Namespace)

	// We use a finalizer to maintain KubeDirector state consistency;
	// e.g. app references and ClusterStatusGens.
	doExit, finalizerErr := r.handleFinalizers(reqLogger, cr)
	if finalizerErr != nil {
		return finalizerErr
	}
	if doExit {
		return nil
	}

	// For a new CR just update the status state/gen.
	shouldProcessCR, processErr := r.handleNewCluster(reqLogger, cr)
	if processErr != nil {
		return processErr
	}
	if !shouldProcessCR {
		return nil
	}

	// Define a common error function for sync problems.
	errLog := func(domain string, err error) {
		shared.LogErrorf(
			reqLogger,
			err,
			cr,
			shared.EventReasonCluster,
			"failed to sync %s: %v",
			domain,
			err,
		)
	}

	checkContainerStates(reqLogger, cr)

	clusterServiceErr := syncClusterService(reqLogger, cr)
	if clusterServiceErr != nil {
		errLog("cluster service", clusterServiceErr)
		return clusterServiceErr
	}

	roles, state, rolesErr := syncClusterRoles(reqLogger, cr)
	if rolesErr != nil {
		errLog("roles", rolesErr)
		return rolesErr
	}

	// The "state" calculated above can be different on next handler pass,
	// so we need to make sure we bump the spec gen now if necessary.
	// If we delay doing this, a handler error (e.g. in syncMemberServices)
	// could cause a handler exit and we would lose the necessary spec gen
	// update.
	if state == clusterMembersChangedUnready || (currentHash != cr.Status.LastConnectionHash) {

		if currentHash != cr.Status.LastConnectionHash {

			annotations := cr.Annotations
			if hashVersion, ok := annotations[shared.HashChangeIncrementor]; ok {
				newV, _ := strconv.ParseInt(hashVersion, 10, 64)
				annotations[shared.HashChangeIncrementor] = strconv.FormatInt(newV+int64(1), 10)
			} else {
				annotations[shared.HashChangeIncrementor] = "1"
				shared.LogInfo(
					reqLogger,
					cr,
					shared.EventReasonCluster,
					"Annotation initialized to 1",
				)
			}
			cr.Annotations = annotations

			if shared.Update(context.TODO(), cr) == nil {
				shared.LogInfo(
					reqLogger,
					cr,
					shared.EventReasonCluster,
					"Updated context",
				)
			}

		}
		incremented := *cr.Status.SpecGenerationToProcess + int64(1)
		cr.Status.SpecGenerationToProcess = &incremented
		cr.Status.LastConnectionHash = currentHash
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

			amIBeingConnectedToThis := func(otherCluster kdv1.KubeDirectorCluster) bool {
				for _, connectedName := range otherCluster.Spec.Connections.Clusters {
					if cr.Name == connectedName {
						return true
					}
				}
				return false
			}

			// Once the cluster is deemed ready, if this cluster is connected to any cluster then
			// we need to notify that cluster that configmeta here has
			// changed, so bump up connectionsGenerationToProcess for that cluster
			allClusters := &kdv1.KubeDirectorClusterList{}
			shared.List(context.TODO(), allClusters)
			// notify clusters to which this cluster is
			// connected
			for _, kubecluster := range allClusters.Items {
				if amIBeingConnectedToThis(kubecluster) {
					shared.LogInfof(
						reqLogger,
						cr,
						shared.EventReasonCluster,
						"connected to cluster {%s}; updating it",
						kubecluster.Name,
					)
					shared.LogInfof(
						reqLogger,
						&kubecluster,
						shared.EventReasonCluster,
						"connected cluster {%s} has changed",
						cr.Name,
					)
					// Annotate cluster to trigger connected cluster's reconciler
					wait := time.Second
					maxWait := 4096 * time.Second
					for {
						updateMetaGenerator := &kubecluster
						annotations := updateMetaGenerator.Annotations
						if annotations == nil {
							annotations = make(map[string]string)
							updateMetaGenerator.Annotations = annotations
						}
						if v, ok := annotations[shared.ConnectionsIncrementor]; ok {
							newV, _ := strconv.Atoi(v)
							annotations[shared.ConnectionsIncrementor] = strconv.Itoa(newV + 1)
						} else {
							annotations[shared.ConnectionsIncrementor] = "1"
						}
						updateMetaGenerator.Annotations = annotations
						if shared.Update(context.TODO(), updateMetaGenerator) == nil {
							break
						}
						// Since update failed, get a fresh copy of this cluster to work with and
						// try update
						updateMetaGenerator, fetchErr := observer.GetCluster(kubecluster.Namespace, kubecluster.Name)
						if fetchErr != nil {
							if errors.IsNotFound(fetchErr) {
								break
							}
						}
						if wait > maxWait {
							return fmt.Errorf(
								"Unable to notify cluster {%s} of configmeta change",
								updateMetaGenerator.Name)
						}
						time.Sleep(wait)
						wait = wait * 2
					}
				}
			}
			cr.Status.State = string(clusterReady)
		}

		if currentHash == cr.Status.LastConnectionHash {
			return nil
		}
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

	membersErr := syncMembers(reqLogger, cr, roles, configmetaGen)
	if membersErr != nil {
		errLog("members", membersErr)
		return membersErr
	}

	return nil
}

// Calculates md5sum of resource-versions of all resources
// connected to this cluster
func calcConnectionsHash(
	con *kdv1.Connections,
	ns string,
) string {

	clusterNames := con.Clusters
	var buffer bytes.Buffer
	for _, c := range clusterNames {
		clusterObj, clusterErr := observer.GetCluster(ns, c)
		buffer.WriteString(c)
		var specNum string
		if clusterErr == nil {
			// extra careful while dereferencing
			if clusterObj.Status.SpecGenerationToProcess == nil {
				specNum = "nil"
			} else {
				specNum = strconv.Itoa(
					int(*clusterObj.Status.SpecGenerationToProcess))
			}
		}
		buffer.WriteString(specNum)
	}
	cmNames := con.ConfigMaps
	for _, c := range cmNames {
		cmObj, cmErr := observer.GetConfigMap(ns, c)
		var rv string
		if cmErr == nil {
			rv = cmObj.ResourceVersion
		}
		buffer.WriteString(c)
		buffer.WriteString(rv)
	}
	secretNames := con.Secrets
	for _, c := range secretNames {
		secretObj, secErr := observer.GetSecret(ns, c)
		var rv string
		if secErr == nil {
			rv = secretObj.ResourceVersion
		}
		buffer.WriteString(c)
		buffer.WriteString(rv)
	}
	// md5 is very cheap for small strings
	md5Sum := md5.Sum([]byte(buffer.String()))
	return hex.EncodeToString(md5Sum[:])
}

// checkContainerStates updates the lastKnownContainerState in each member
// status. It will also move ready or config-error nodes back to create pending
// status if their container ID has changed.
func checkContainerStates(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	numRoleStatuses := len(cr.Status.Roles)
	for i := 0; i < numRoleStatuses; i++ {
		roleStatus := &(cr.Status.Roles[i])
		numMemberStatuses := len(roleStatus.Members)
		for j := 0; j < numMemberStatuses; j++ {
			memberStatus := &(roleStatus.Members[j])
			containerID := ""
			if memberStatus.Pod != "" {
				memberStatus.StateDetail.LastKnownContainerState = containerMissing
				pod, podErr := observer.GetPod(cr.Namespace, memberStatus.Pod)
				if podErr == nil {
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.Name == executor.AppContainerName {
							containerID = containerStatus.ContainerID
							if containerStatus.State.Running != nil {
								if (cr.Status.SpecGenerationToProcess != nil) &&
									(memberStatus.StateDetail.LastConfigDataGeneration != nil) &&
									(*cr.Status.SpecGenerationToProcess != *memberStatus.StateDetail.LastConfigDataGeneration) {
									memberStatus.StateDetail.LastKnownContainerState = containerUnresponsive
								} else if len(memberStatus.StateDetail.PendingNotifyCmds) != 0 {
									memberStatus.StateDetail.LastKnownContainerState = containerUnresponsive
								} else {
									memberStatus.StateDetail.LastKnownContainerState = containerUnknown
									for _, condition := range pod.Status.Conditions {
										if condition.Type == corev1.PodReady {
											switch condition.Status {
											case corev1.ConditionTrue:
												memberStatus.StateDetail.LastKnownContainerState = containerRunning
											case corev1.ConditionFalse:
												memberStatus.StateDetail.LastKnownContainerState = containerUnresponsive
											}
											break
										}
									}
								}
							} else if containerStatus.State.Waiting != nil {
								// Don't rely on the waiting state Reason here
								// to determine if init is running; it's an
								// arbitrary string we possibly can't depend on.
								if (len(pod.Status.InitContainerStatuses) != 0) &&
									(pod.Status.InitContainerStatuses[0].State.Terminated == nil) {
									memberStatus.StateDetail.LastKnownContainerState = containerInitializing
								} else {
									memberStatus.StateDetail.LastKnownContainerState = containerWaiting
								}
							} else if containerStatus.State.Terminated != nil {
								memberStatus.StateDetail.LastKnownContainerState = containerTerminated
							} else {
								memberStatus.StateDetail.LastKnownContainerState = containerUnknown
							}
							break
						}
					}
				} else {
					if !errors.IsNotFound(podErr) {
						memberStatus.StateDetail.LastKnownContainerState = containerUnknown
					}
				}
				if (memberStatus.State == string(memberReady)) ||
					(memberStatus.State == string(memberConfigError)) {
					if containerID != memberStatus.StateDetail.LastConfiguredContainer {
						memberStatus.State = string(memberCreatePending)
						if memberStatus.PVC == "" {
							shared.LogInfof(
								reqLogger,
								cr,
								shared.EventReasonMember,
								"container ID has changed for member{%s}; no persistent storage, will re-run setup",
								memberStatus.Pod,
							)
							// No persistent storage, so any previously uploaded
							// stuff has been lost.
							memberStatus.StateDetail.LastConfigDataGeneration = nil
							memberStatus.StateDetail.LastSetupGeneration = nil
							// We will completely rerun the config, so drop any
							// pending notifies.
							memberStatus.StateDetail.PendingNotifyCmds = []*kdv1.NotificationDesc{}
						} else {
							shared.LogInfof(
								reqLogger,
								cr,
								shared.EventReasonMember,
								"container ID has changed for member{%s}; will re-check setup",
								memberStatus.Pod,
							)
						}
					}
				}
			}
		}
	}
}

// updateStateRollup examines current per-member status and sets the top-level
// config rollup appropriately.
func updateStateRollup(
	cr *kdv1.KubeDirectorCluster,
) {

	cr.Status.MemberStateRollup.MembershipChanging = false
	cr.Status.MemberStateRollup.MembersDown = false
	cr.Status.MemberStateRollup.MembersInitializing = false
	cr.Status.MemberStateRollup.MembersWaiting = false
	cr.Status.MemberStateRollup.MembersRestarting = false
	cr.Status.MemberStateRollup.ConfigErrors = false

	checkMemberDown := func(memberStatus kdv1.MemberStatus) {
		if (memberStatus.StateDetail.LastKnownContainerState == containerTerminated) ||
			(memberStatus.StateDetail.LastKnownContainerState == containerUnresponsive) ||
			(memberStatus.StateDetail.LastKnownContainerState == containerMissing) {
			cr.Status.MemberStateRollup.MembersDown = true
		}
	}

	for _, roleStatus := range cr.Status.Roles {
		for _, memberStatus := range roleStatus.Members {
			switch memberState(memberStatus.State) {
			case memberCreatePending:
				// DO NOT check member down here; missing container is OK.
				// See if this member is new or is "rebooting".
				if memberStatus.StateDetail.LastConfiguredContainer == "" {
					cr.Status.MemberStateRollup.MembershipChanging = true
				} else {
					cr.Status.MemberStateRollup.MembersRestarting = true
				}
				// Count missing container as waiting, at this point.
				if memberStatus.StateDetail.LastKnownContainerState == containerMissing {
					cr.Status.MemberStateRollup.MembersWaiting = true
				}
			case memberCreating:
				checkMemberDown(memberStatus)
				// See if this member is new or is "rebooting".
				if memberStatus.StateDetail.LastConfiguredContainer == "" {
					cr.Status.MemberStateRollup.MembershipChanging = true
				} else {
					cr.Status.MemberStateRollup.MembersRestarting = true
				}
				// DO NOT treat missing container as waiting, at this point.
			case memberReady:
				checkMemberDown(memberStatus)
			case memberDeletePending:
				checkMemberDown(memberStatus)
				cr.Status.MemberStateRollup.MembershipChanging = true
			case memberDeleting:
				// DO NOT check member down here; missing container is OK.
				cr.Status.MemberStateRollup.MembershipChanging = true
			case memberConfigError:
				checkMemberDown(memberStatus)
				cr.Status.MemberStateRollup.ConfigErrors = true
			}
			if memberStatus.StateDetail.LastKnownContainerState == containerInitializing {
				cr.Status.MemberStateRollup.MembersInitializing = true
			}
			if memberStatus.StateDetail.LastKnownContainerState == containerWaiting {
				cr.Status.MemberStateRollup.MembersWaiting = true
			}
		}
	}
}

// handleNewCluster looks in the cache for the last-known status generation
// UID for this CR. If there is one, make sure the UID is what we expect, and
// if so return true to keep processing the CR. If there is not any last-known
// UID, this is either a new CR or one that was created before this KD came up.
// In the former case, where the CR status itself has no generation UID: set
// the cluster state to creating (this will also trigger population of the
// generation UID) and return false to cause this handler to exit; we'll pick
// up further processing in the next handler. In the latter case, sync up our
// internal state with the visible state of the CR and return true to continue
// processing. In either-new cluster case invoke shared.EnsureClusterAppReference
// to mark that the app is being used.
func (r *ReconcileKubeDirectorCluster) handleNewCluster(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) (bool, error) {

	// Have we seen this cluster before?
	incoming := cr.Status.GenerationUID
	lastKnown, ok := ClusterStatusGens.ReadStatusGen(cr.UID)
	if ok {
		// Yep we've already done processing for this cluster previously.
		// Sanity check that the UID is what we expect... it REALLY should be,
		// but if there is a bug/race in the client code or unexpected behavior
		// of the K8s API consistency then it might not be.
		if lastKnown.UID == incoming {
			return true, nil
		}
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonNoEvent,
			"ignoring cluster CR with stale status UID; will retry",
		)
		mismatchErr := fmt.Errorf(
			"incoming UID %s != last known UID %s",
			incoming,
			lastKnown.UID,
		)
		return false, mismatchErr
	}
	// This is a new cluster, or at least "new to us", so mark that its app
	// is in use.
	shared.EnsureClusterAppReference(
		cr.Namespace,
		cr.Name,
		*(cr.Spec.AppCatalog),
		cr.Spec.AppID,
	)
	// There are creation-race or KD-recovery cases where the app might not
	// exist, so check that now.
	_, appErr := catalog.GetApp(cr)
	if appErr != nil {
		shared.LogError(
			reqLogger,
			appErr,
			cr,
			shared.EventReasonCluster,
			"app referenced by cluster does not exist",
		)
		// We're not going to take any other steps at this point... not even
		// going to remove the app reference. Operations on this cluster
		// could fail, but it might be recoverable by re-creating the app CR.
	}
	if incoming == "" {
		// This is an actual newly-created cluster, so kick off the processing.
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"new",
		)
		cr.Status.State = string(clusterCreating)
		return false, nil
	}
	// This cluster has been processed before but we're not aware of it yet.
	// Probably KD has been restarted. Make us aware of this cluster.
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonNoEvent,
		"unknown cluster with incoming gen uid %s",
		incoming,
	)
	ClusterStatusGens.WriteValidatedStatusGen(cr.UID, incoming)
	return true, nil
}

// handleFinalizers will, if deletion has been requested, try to do any
// cleanup and then remove our finalizer from the in-memory CR. If deletion
// has NOT been requested then it will add our finalizer to the in-memory CR
// if it is absent.
func (r *ReconcileKubeDirectorCluster) handleFinalizers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) (bool, error) {

	if cr.DeletionTimestamp != nil {
		// If a deletion has been requested, while ours (or other) finalizers
		// existed on the CR, go ahead and remove our finalizer.
		shared.RemoveFinalizer(cr)
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"greenlighting for deletion",
		)
		// Also clear the status gen from our cache.
		ClusterStatusGens.DeleteStatusGen(cr.UID)
		shared.RemoveClusterAppReference(
			cr.Namespace,
			cr.Name,
			*(cr.Spec.AppCatalog),
			cr.Spec.AppID,
		)
		return true, nil
	}

	// If our finalizer doesn't exist on the CR, put it in there.
	shared.EnsureFinalizer(cr)

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
