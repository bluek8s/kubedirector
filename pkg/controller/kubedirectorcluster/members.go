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
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-logr/logr"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/exec"
)

// syncMembers is responsible for adding or deleting members. It and
// syncMemberNotifies are the only functions in this file that are invoked
// from another file (from the syncCluster function in cluster.go). Along with
// k8s interactions (changing statefulset replica count), this involves
// creating notifications to existing members about additions/deletions,
// injecting configmeta data into members, and triggering application setup.
// This function will modify the member status data structures to update their
// states.
func syncMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	roles []*roleInfo,
	configmetaGenerator func(string) string,
) error {

	// Update configmeta in current ready members if necessary. These may not
	// all succeed if any members are down. We'll return early if we fail to
	// update any ready members or if there are rebooting members that will
	// eventually need to be updated.
	allMembersUpdated := true
	checkGenOk := func(stateMembers []*kdv1.MemberStatus) bool {
		for _, member := range stateMembers {
			if member.StateDetail.LastConfigDataGeneration == nil {
				continue
			}
			if *member.StateDetail.LastConfigDataGeneration == *cr.Status.SpecGenerationToProcess {
				continue
			}
			return false
		}
		return true
	}
	for _, r := range roles {
		if ready, readyOk := r.membersByState[memberReady]; readyOk {
			handleReadyMembers(reqLogger, cr, r, configmetaGenerator)
			if allMembersUpdated {
				allMembersUpdated = checkGenOk(ready)
			}
		}
		if allMembersUpdated {
			if createPending, createPendingOk := r.membersByState[memberCreatePending]; createPendingOk {
				allMembersUpdated = checkGenOk(createPending)
			}
		}
		if allMembersUpdated {
			if creating, creatingOk := r.membersByState[memberCreating]; creatingOk {
				allMembersUpdated = checkGenOk(creating)
			}
		}
	}
	if !allMembersUpdated {
		// Not an error, we're just not done yet.
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"cluster spec change processing blocked on member updates",
		)
		return nil
	}

	// Do the state-appropriate actions for each member in each role.
	// Note that we don't handle roles in parallel currently because some
	// role handling involves "execute script on all cluster ready members",
	// and such operations need to be serialized. (For simplicity of
	// implementation in the app setup package.)
	for _, r := range roles {
		if _, ok := r.membersByState[memberCreatePending]; ok {
			handleCreatePendingMembers(reqLogger, cr, r)
		}
		if _, ok := r.membersByState[memberCreating]; ok {
			handleCreatingMembers(reqLogger, cr, r, roles, configmetaGenerator)
		}
		if _, ok := r.membersByState[memberDeletePending]; ok {
			handleDeletePendingMembers(reqLogger, cr, r, roles)
		}
		if _, ok := r.membersByState[memberDeleting]; ok {
			handleDeletingMembers(reqLogger, cr, r)
		}
	}

	return nil
}

// syncMemberNotifies is responsible processing any existing member
// notification queues. It and syncMembers are the only functions in this file
// that are invoked from another file (from the syncCluster function in
// cluster.go). Along with executing the notify commands into members, this
// function will modify the member status data structures to update their
// notification queues.
func syncMemberNotifies(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	// Let's iterate over the per-member status checking their state and
	// whether they have any pending notifies.
	var membersToProcess []*kdv1.MemberStatus
	var membersSkippingNotifies []*kdv1.MemberStatus
	transitionalMembers := false
	numRoleStatuses := len(cr.Status.Roles)
	for i := 0; i < numRoleStatuses; i++ {
		roleStatus := &(cr.Status.Roles[i])
		numMembers := len(roleStatus.Members)
		for j := 0; j < numMembers; j++ {
			memberStatus := &(roleStatus.Members[j])
			if memberStatus.State == string(memberReady) {
				// Ready-member handling depends on whether it has any
				// pending notifies.
				if len(memberStatus.StateDetail.PendingNotifyCmds) != 0 {
					// If it does, we'll need to process the notifies below.
					membersToProcess = append(membersToProcess, memberStatus)
				} else if !transitionalMembers {
					// If not, AND if there are no transitional-state members
					// (who might be on their way to generating a notify),
					// then we are going to skip notifies on this member.
					membersSkippingNotifies = append(membersSkippingNotifies, memberStatus)
				}
			} else if memberStatus.State != string(memberConfigError) {
				// Once we find any members in a transitional state (neither
				// ready/configured nor config-error) make a note of that and
				// clear out any previously noted members-to-skip-notifies.
				// Notification skipping will have to wait until everyone is
				// stable.
				transitionalMembers = true
				membersSkippingNotifies = nil
			}
		}
	}
	// For any ready members where it is safe to do-no-notifies, ffwd their
	// setup generation number to the desired point.
	for _, memberSkipping := range membersSkippingNotifies {
		if memberSkipping.StateDetail.LastSetupGeneration != nil {
			memberSkipping.StateDetail.LastSetupGeneration =
				memberSkipping.StateDetail.LastConfigDataGeneration
		}
	}
	// Bail out now if there are no notifies to send.
	numToProcess := len(membersToProcess)
	if numToProcess == 0 {
		return
	}
	// Spawn notify commands in goroutines.
	var wgReady sync.WaitGroup
	wgReady.Add(numToProcess)
	for _, member := range membersToProcess {
		go func(m *kdv1.MemberStatus) {
			defer wgReady.Done()
			var newQueue []*kdv1.NotificationDesc
			for _, notify := range m.StateDetail.PendingNotifyCmds {
				cmd := appPrepStartscript + " " + strings.Join(notify.Arguments, " ")
				notifyError := executor.RunScript(
					reqLogger,
					cr,
					cr.Namespace,
					m.Pod,
					m.StateDetail.LastConfiguredContainer,
					executor.AppContainerName,
					"app reconfig",
					strings.NewReader(cmd),
				)
				// XXX Note that we don't distinguish here between pod-down
				// or unreachable and the case where the script runs but
				// actually returns an error. Arguably in the latter case we
				// should transition this node to a config error state.
				if notifyError != nil {
					newQueue = append(newQueue, notify)
					shared.LogErrorf(
						reqLogger,
						notifyError,
						cr,
						shared.EventReasonMember,
						"failed to notify member{%s} about member changes",
						m.Pod,
					)
				} else {
					// Update the setup generation number if no transitional
					// members are left to process. (We could omit this and
					// let the next handler poll take care of it as a "skip
					// notifies" case above, but let's be more proactive.)
					if !transitionalMembers {
						m.StateDetail.LastSetupGeneration = m.StateDetail.LastConfigDataGeneration
					}
				}
			}
			// Avoid a useless status write if we just rebuilt the same queue.
			if len(m.StateDetail.PendingNotifyCmds) != len(newQueue) {
				m.StateDetail.PendingNotifyCmds = newQueue
			}
		}(member)
	}
	wgReady.Wait()
}

// handleReadyMembers operates on all members in the role that are currently
// in the ready state. It will update the configmeta inside each guest with
// the latest content.
func handleReadyMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	configmetaGenerator func(string) string,
) {

	connectionsVersion := getConnectionVersion(reqLogger, cr, role)

	ready := role.membersByState[memberReady]
	var wgReady sync.WaitGroup
	wgReady.Add(len(ready))
	for _, member := range ready {
		go func(m *kdv1.MemberStatus) {
			defer wgReady.Done()
			// If this pod never got configmeta (because it has no setup
			// package), it doesn't need an update.
			if m.StateDetail.LastConfigDataGeneration == nil {
				return
			}
			// If this pod has already been updated on a previous handler
			// pass, skip it.
			if *m.StateDetail.LastConfigDataGeneration == *cr.Status.SpecGenerationToProcess {
				return
			}
			// Drop in the new configmeta.
			configmeta := configmetaGenerator(m.Pod)
			createFileErr := executor.CreateFile(
				reqLogger,
				cr,
				cr.Namespace,
				m.Pod,
				m.StateDetail.LastConfiguredContainer,
				executor.AppContainerName,
				configMetaFile,
				strings.NewReader(configmeta),
			)
			if createFileErr != nil {
				shared.LogErrorf(
					reqLogger,
					createFileErr,
					cr,
					shared.EventReasonMember,
					"failed to update config in member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				return
			}

			memberVersion := *m.StateDetail.LastConnectionVersion

			if memberVersion < connectionsVersion {
				shared.LogInfo(
					reqLogger,
					cr,
					shared.EventReasonCluster,
					fmt.Sprintf("--reconnect will be called for pod : %s", m.Pod),
				)

				containerID := m.StateDetail.LastConfiguredContainer
				cmd := fmt.Sprintf(appPrepConfigReconnectCmd, containerID)

				cmdErr := executor.RunScript(
					reqLogger,
					cr,
					cr.Namespace,
					m.Pod,
					m.StateDetail.LastConfiguredContainer,
					executor.AppContainerName,
					"app config",
					strings.NewReader(cmd),
				)
				if cmdErr != nil {
					shared.LogErrorf(
						reqLogger,
						cmdErr,
						cr,
						shared.EventReasonMember,
						"failed to run startcsript with --reconnect in member{%s} in role{%s}",
						m.Pod,
						role.roleStatus.Name,
					)
					return
				}
				memberVersion = memberVersion + int64(1)
				m.StateDetail.LastConnectionVersion = &memberVersion

			}

			m.StateDetail.LastConfigDataGeneration = cr.Status.SpecGenerationToProcess
		}(member)
	}
	wgReady.Wait()

}

// handleCreatePendingMembers operates on all members in the role that are
// currently in the create pending state. It first adjusts the statefulset
// replicas count as necessary, then checks each new member to see if it is
// running. If so, it moves it to the creating state. It is quite possible for
// members to be left in the create pending state across multiple reconciler
// passes.
func handleCreatePendingMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	// Fix statefulset if necessary, and bail out if it is not good yet.
	if !checkMemberCount(reqLogger, cr, role) {
		return
	}
	if !replicasSynced(reqLogger, cr, role) {
		return
	}

	createPending := role.membersByState[memberCreatePending]

	// Check each new member to see if it is running yet.
	var wgRunning sync.WaitGroup
	wgRunning.Add(len(createPending))
	for _, member := range createPending {
		go func(m *kdv1.MemberStatus) {
			defer wgRunning.Done()
			pod, podGetErr := observer.GetPod(cr.Namespace, m.Pod)
			if podGetErr != nil {
				// Can't get the pod. Skip it and try again later. This is
				// not necessarily an error; K8s might be slow.
				if apierrors.IsNotFound(podGetErr) {
					shared.LogInfof(
						reqLogger,
						cr,
						shared.EventReasonMember,
						"failed to find member{%s} in role{%s}; will retry",
						m.Pod,
						role.roleStatus.Name,
					)
				} else {
					shared.LogErrorf(
						reqLogger,
						podGetErr,
						cr,
						shared.EventReasonMember,
						"failed to find member{%s} in role{%s}",
						m.Pod,
						role.roleStatus.Name,
					)
				}
				return
			}
			if pod.Status.Phase == corev1.PodRunning {
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if (containerStatus.Name == executor.AppContainerName) &&
						(containerStatus.ContainerID != "") {
						m.StateDetail.ConfiguringContainer = containerStatus.ContainerID
						m.State = string(memberCreating)
						// We don't need to update membersByState; the newly
						// creating-state members will be processed on a
						// subsequent reconciler pass.
						return
					}
				}
			}
		}(member)
	}
	wgRunning.Wait()
}

// handleCreatingMembers operates on all members in the role that are
// currently in the creating state, handling configmeta and app setup and
// initial configuration.  All ready members in the cluster are notified
// of the addition of the successfully configured members, which are moved to
// ready state. Members that were not successfully configured are left in the
// creating state and we'll tackle them again on next reconciler pass.
func handleCreatingMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
	configmetaGenerator func(string) string,
) {

	creating := role.membersByState[memberCreating]

	// Fetch setup url package
	setupURL, setupURLErr := catalog.AppSetupPackageURL(cr, role.roleStatus.Name)
	if setupURLErr != nil {
		shared.LogErrorf(
			reqLogger,
			setupURLErr,
			cr,
			shared.EventReasonRole,
			"failed to fetch setup url for role{%s}",
			role.roleStatus.Name,
		)
		return
	}

	// Perform setup on all of these members.
	var wgSetup sync.WaitGroup
	wgSetup.Add(len(creating))
	for _, member := range creating {
		go func(m *kdv1.MemberStatus) {
			defer wgSetup.Done()

			containerID := m.StateDetail.ConfiguringContainer
			setFinalState := func(state memberState, errorDetail *string) {
				m.State = string(state)
				m.StateDetail.ConfigErrorDetail = errorDetail
			}

			connectionVersion := getConnectionVersion(reqLogger, cr, role)

			m.StateDetail.LastConnectionVersion = &connectionVersion

			// Check to see if we have to inject one or more files for this member
			if len(role.roleSpec.FileInjections) != 0 {
				injectErr := injectFiles(reqLogger, cr, m.Pod, containerID, role)
				if injectErr != nil {
					shared.LogErrorf(
						reqLogger,
						injectErr,
						cr,
						shared.EventReasonMember,
						"failed to inject one or more files for member{%s} in role{%s}",
						m.Pod,
						role.roleStatus.Name,
					)
					statusErrMsg := fmt.Sprintf(
						"failed requested file injections: %s",
						injectErr.Error(),
					)
					setFinalState(memberConfigError, &statusErrMsg)
					return
				}
			}

			if setupURL == "" {
				setFinalState(memberReady, nil)
				shared.LogInfof(
					reqLogger,
					cr,
					shared.EventReasonMember,
					"initial config skipped for member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				return
			}

			// Start or continue the initial configuration.
			isFinal, configErr := appConfig(
				reqLogger,
				cr,
				setupURL,
				m.Pod,
				containerID,
				&m.StateDetail,
				role.roleStatus.Name,
				configmetaGenerator,
			)
			if !isFinal {
				shared.LogInfof(
					reqLogger,
					cr,
					shared.EventReasonMember,
					"initial config ongoing for member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				return
			}
			if configErr != nil {
				shared.LogErrorf(
					reqLogger,
					configErr,
					cr,
					shared.EventReasonMember,
					"failed to run initial config for member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				statusErrMsg := fmt.Sprintf(
					"execution of app config failed: %s",
					configErr.Error(),
				)
				setFinalState(memberConfigError, &statusErrMsg)
				return
			}
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonMember,
				"initial config done for member{%s} in role{%s}",
				m.Pod,
				role.roleStatus.Name,
			)
			setFinalState(memberReady, nil)
		}(member)
	}
	wgSetup.Wait()

	// Generate the notifications to later send to any ready members that
	// aren't up-to-date. Notifications will not be sent about members in this
	// list still in the creating state, and also will not be sent about
	// members that already have lastConfiguredContainer set (i.e. are just
	// reboots) -- see the fqdnsList function.
	generateNotifies(reqLogger, cr, role, allRoles)

	// Now update configuringContainer and lastConfiguredContainer for the
	// any members no longer in creating state. We don't need to update
	// membersByState because these members won't be processed again until a
	// subsequent reconciler pass anyway.
	for _, member := range creating {
		if member.State != string(memberCreating) {
			member.StateDetail.LastConfiguredContainer = member.StateDetail.ConfiguringContainer
			member.StateDetail.ConfiguringContainer = ""
		}
	}
}

// handleDeletingMembers operates on all members in the role that are
// currently in the deleting state. If the replicas count on the statefulset
// has not been successfully updated yet, it attempts that change and returns.
// Otherwise it checks each pod to see if it is gone, and if so deletes the
// corresponding PVC and service. Once all member-related objects are gone,
// the member status is marked for removal. It is quite possible for members
// to be left in the deleting state across multiple reconciler passes.
func handleDeletingMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	// Fix statefulset if necessary. Note that the statefulset might not exist
	// in this case, so check that.
	if role.statefulSet != nil {
		if !checkMemberCount(reqLogger, cr, role) {
			return
		}
	}
	// We won't call replicasSynced here. We've already sent out the delete
	// notifies, so it wouldn't help batch those up. And it's nice to be
	// able to see the member statuses vanish one by one in concert with the
	// pods going away.

	deleting := role.membersByState[memberDeleting]

	// Now handle each of the deleting members in parallel. We want to clean
	// up the corresponding service and volume claim, and ultimately the
	// member status.
	var wgCleanup sync.WaitGroup
	wgCleanup.Add(len(deleting))
	for _, member := range deleting {
		go func(m *kdv1.MemberStatus) {
			defer wgCleanup.Done()
			_, podGetErr := observer.GetPod(cr.Namespace, m.Pod)
			if podGetErr == nil {
				// Pod isn't gone yet. Skip it.
				return
			} else if !apierrors.IsNotFound(podGetErr) {
				// Some error other than "not found". Skip pod and try again
				// later.
				shared.LogErrorf(
					reqLogger,
					podGetErr,
					cr,
					shared.EventReasonMember,
					"failed to find member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				return
			}
			if m.Service != "" {
				serviceDelErr := executor.DeletePodService(
					reqLogger,
					cr.Namespace,
					m.Service,
				)
				if serviceDelErr == nil || apierrors.IsNotFound(serviceDelErr) {
					m.Service = ""
				} else {
					shared.LogErrorf(
						reqLogger,
						serviceDelErr,
						cr,
						shared.EventReasonMember,
						"failed to delete service{%s}",
						m.Service,
					)
				}
			}
			if m.PVC != "" {
				pvcDelErr := executor.DeletePVC(
					cr.Namespace,
					m.PVC,
				)
				if pvcDelErr == nil || apierrors.IsNotFound(pvcDelErr) {
					m.PVC = ""
				} else {
					shared.LogErrorf(
						reqLogger,
						pvcDelErr,
						cr,
						shared.EventReasonMember,
						"failed to delete PVC{%s}",
						m.PVC,
					)
				}
			}
			// If service and PVC have been cleaned up, mark member status for
			// removal.
			if m.Service == "" && m.PVC == "" {
				m.Pod = ""
			}
		}(member)
	}
	wgCleanup.Wait()
}

// handleDeletePendingMembers operates on all members in the role that are
// currently in the delete pending state. It first notifies all ready members
// in the cluster of the impending deletion; then it moves all of these
// delete pending members to the deleting state.
func handleDeletePendingMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
) {

	// Generate the notifications for these members, to later send to any
	// ready nodes that aren't up-to-date.
	generateNotifies(reqLogger, cr, role, allRoles)

	// All done, change state.
	for _, member := range role.membersByState[memberDeletePending] {
		member.State = string(memberDeleting)
	}
	role.membersByState[memberDeleting] = append(
		role.membersByState[memberDeleting],
		role.membersByState[memberDeletePending]...,
	)
	delete(role.membersByState, memberDeletePending)
}

// checkMemberCount examines an existing statefulset to see if its replicas
// count needs to be reconciled, and does so if necessary. Return false if the
// statefulset had to be changed.
func checkMemberCount(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) bool {

	// Calculate the number of members that a statefulset/role SHOULD
	// currently have. Don't use roleSpec here. roleSpec could flap around and
	// we'll ignore it if we're still working on a previous change.
	replicas := int32(len(role.membersByState[memberCreatePending]) +
		len(role.membersByState[memberCreating]) +
		len(role.membersByState[memberReady]) +
		len(role.membersByState[memberConfigError]))

	// Fix the statefulset if we haven't successfully resized it yet.
	if *(role.statefulSet.Spec.Replicas) != replicas {
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonRole,
			"changing replicas count for role{%s}: %v -> %v",
			role.roleStatus.Name,
			*(role.statefulSet.Spec.Replicas),
			replicas,
		)
		updateErr := executor.UpdateStatefulSetReplicas(
			reqLogger,
			cr,
			replicas,
			role.statefulSet,
		)
		if updateErr != nil {
			shared.LogErrorf(
				reqLogger,
				updateErr,
				cr,
				shared.EventReasonRole,
				"failed to change StatefulSet{%s} replicas",
				role.statefulSet.Name,
			)
		}
		return false
	}

	return true
}

// replicasSynced returns true if the role's statefulset has its status
// replicas count matching its spec replicas count.
func replicasSynced(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) bool {

	if role.statefulSet.Status.Replicas != *(role.statefulSet.Spec.Replicas) {
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonRole,
			"waiting for replicas count for role{%s}: %v -> %v",
			role.roleStatus.Name,
			role.statefulSet.Status.Replicas,
			*(role.statefulSet.Spec.Replicas),
		)
		return false
	}

	return true
}

// setupNodePrep injects the configcli package (configcli et al) into the member's
// container and installs it.
func setupNodePrep(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	expectedContainerID string,
) error {

	// Check to see if the destination file exists already, in which case just
	// return. Also bail out if we cannot manage to check file existence.
	fileExists, fileError := executor.IsFileExists(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		configcliTestFile,
	)
	if fileError != nil {
		return fileError
	} else if fileExists {
		return nil
	}

	// Inject the configcli package, taken from the KubeDirector's container.
	nodePrepFile, openErr := os.Open(configcliSrcFile)
	if openErr != nil {
		return fmt.Errorf(
			"failed to open file %s: %v",
			configcliSrcFile,
			openErr,
		)
	}
	defer nodePrepFile.Close()
	createErr := executor.CreateFile(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		configcliDestFile,
		bufio.NewReader(nodePrepFile),
	)
	if createErr != nil {
		return createErr
	}

	// Install it.
	return executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		"configcli setup",
		strings.NewReader(configcliInstallCmd),
	)
}

// setupAppConfig injects the app setup package (if any) into the member's
// container and installs it.
func setupAppConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	setupURL string,
	podName string,
	expectedContainerID string,
	roleName string,
) error {

	// Check to see if the destination file exists already, in which case just
	// return. Also bail out if we cannot manage to check file existence.
	fileExists, fileError := executor.IsFileExists(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		appPrepStartscript,
	)
	if fileError != nil {
		return fileError
	} else if fileExists {
		return nil
	}

	// Fetch and install it.
	cmd := strings.Replace(appPrepInitCmd, "{{APP_CONFIG_URL}}", setupURL, -1)
	return executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		"app config setup",
		strings.NewReader(cmd),
	)
}

// injectFiles injects one or more files as specified through role spec
// Each file will be downloaded to the specified location inside the pod and
// file permissions and ownership will be updated based on the spec.
func injectFiles(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	expectedContainerID string,
	role *roleInfo,
) error {

	for _, fileInjection := range role.roleSpec.FileInjections {
		// Get base file name
		fileName := filepath.Base(fileInjection.SrcURL)
		// Construct the full destination path
		destFile := filepath.Join(fileInjection.DestDir, fileName)
		// Build the complete injection command. Include setting mode/owner
		// if specified.
		fileInjectCmd := fmt.Sprintf(
			fileInjectionCommand,
			fileInjection.DestDir,
			fileInjection.DestDir,
			fileInjection.SrcURL,
			destFile,
		)
		if fileInjection.Permissions != nil {
			if fileInjection.Permissions.FileMode != nil {
				fileModeStr := strconv.FormatInt(int64(*fileInjection.Permissions.FileMode), 8)
				fileInjectCmd = strings.Join(
					[]string{fileInjectCmd, "&&", "chmod", fileModeStr, destFile},
					" ",
				)
			}
			if fileInjection.Permissions.FileOwner != nil {
				fileInjectCmd = strings.Join(
					[]string{fileInjectCmd, "&&", "chown", *fileInjection.Permissions.FileOwner, destFile},
					" ",
				)
			}
			if fileInjection.Permissions.FileGroup != nil {
				fileInjectCmd = strings.Join(
					[]string{fileInjectCmd, "&&", "chgrp", *fileInjection.Permissions.FileGroup, destFile},
					" ",
				)
			}
		}
		// Away we go!
		err := executor.RunScript(
			reqLogger,
			cr,
			cr.Namespace,
			podName,
			expectedContainerID,
			executor.AppContainerName,
			"file injection ("+destFile+")",
			strings.NewReader(fileInjectCmd),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateNotifies prepares the info for handling a lifecycle event to all
// currently ready or rebooting members that have a stale last setup gen. That
// info is added to each such member's notification queue.
func generateNotifies(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
) {

	// specGenerationToProcess should always be non-nil in current usage,
	// but doesn't hurt to check.
	if cr.Status.SpecGenerationToProcess == nil {
		return
	}

	for _, otherRole := range allRoles {
		if len(otherRole.membersByState[memberReady])+
			len(otherRole.membersByState[memberCreatePending])+
			len(otherRole.membersByState[memberCreating]) == 0 {
			// This is not just an optimization; note also that in the case
			// of a role with zero overall members then otherRole.roleStatus
			// referenced below will be nil. That case is covered here too.
			continue
		}
		setupURL, setupURLErr := catalog.AppSetupPackageURL(cr, otherRole.roleStatus.Name)
		if setupURLErr != nil {
			shared.LogErrorf(
				reqLogger,
				setupURLErr,
				cr,
				shared.EventReasonRole,
				"failed to fetch setup url for role{%s}",
				otherRole.roleStatus.Name,
			)
			setupURL = ""
		}
		if setupURL == "" {
			// No notification necessary for any member in this role.
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonRole,
				"notify skipped for members in role{%s}",
				otherRole.roleStatus.Name,
			)
			continue
		}
		processor := func(stateMembers []*kdv1.MemberStatus) {
			for _, member := range stateMembers {
				if member.StateDetail.LastSetupGeneration == nil {
					continue
				}
				if *member.StateDetail.LastSetupGeneration == *cr.Status.SpecGenerationToProcess {
					continue
				}
				queueNotify(
					reqLogger,
					cr,
					member.Pod,
					&member.StateDetail,
					otherRole.roleStatus.Name,
					role,
				)
			}
		}
		if ready, readyOk := otherRole.membersByState[memberReady]; readyOk {
			processor(ready)
		}
		if createPending, createPendingOk := otherRole.membersByState[memberCreatePending]; createPendingOk {
			processor(createPending)
		}
		if creating, creatingOk := otherRole.membersByState[memberCreating]; creatingOk {
			processor(creating)
		}
	}
}

// systemdOk returns immediately without error if the app does not require
// systemd; otherwise it checks whether systemd is usable.
func systemdOk(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	expectedContainerID string,
) (bool, error) {

	appCR, appErr := catalog.GetApp(cr)
	if appErr != nil {
		return false, appErr
	}
	if !appCR.Spec.SystemdRequired {
		// App doesn't require systemd so we don't care.
		return true, nil
	}
	cmd := "systemctl status systemd-journald > /tmp/journald-status.out 2>&1"
	cmdErr := executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		"systemd check",
		strings.NewReader(cmd),
	)
	if cmdErr == nil {
		// No error, including in the return status of the command. All good.
		return true, nil
	}
	// If this was an error status returned from the command, the answer to
	// our question is "no not ok" but we didn't experience an error trying to
	// run the command.
	_, iscoe := cmdErr.(exec.CodeExitError)
	if iscoe {
		return false, nil
	}
	return false, cmdErr
}

// appConfig does the initial run of a member's app setup script, including
// the installation of any prerequisite materials. Check the returned
// "result is final" boolean to see if this needs to be called again on next
// reconciler pass.
func appConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	setupURL string,
	podName string,
	expectedContainerID string,
	stateDetail *kdv1.MemberStateDetail,
	roleName string,
	configmetaGenerator func(string) string,
) (bool, error) {

	// If a config error detail already exists, this is a restart of a member
	// that had been in config error state. In that case we won't try
	// checking the existing state within the guest.
	if stateDetail.ConfigErrorDetail != nil {
		// Clean up for the retry.
		stateDetail.ConfigErrorDetail = nil
		stateDetail.LastSetupGeneration = nil
		stateDetail.PendingNotifyCmds = []*kdv1.NotificationDesc{}
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonMember,
			"member{%s} was previously in config error state; re-trying setup",
			podName,
		)
	} else {
		// For initial configuration, startscript will run asynchronously and we
		// will check back periodically. So let's have a look at the existing
		// status if any.
		var statusStrB strings.Builder
		fileExists, fileError := executor.ReadFile(
			reqLogger,
			cr,
			cr.Namespace,
			podName,
			expectedContainerID,
			executor.AppContainerName,
			appPrepConfigStatus,
			&statusStrB,
		)
		if fileError != nil {
			return true, fileError
		}
		if fileExists {
			// Configure script was previously started. Extract the container
			// ID where it is/was run, and see if we have a final config status.
			statusStr := statusStrB.String()
			splitPoint := strings.LastIndex(statusStr, "=")
			if splitPoint == -1 {
				// That's odd. It's not the file that we wrote...
				err := errors.New("configure failed with malformed status file")
				return true, err
			}
			configContainerID, configStatus := statusStr[:splitPoint], statusStr[splitPoint+1:]
			if configStatus == "" {
				// Script isn't done. But was it interrupted by a container
				// restart? If not we will return and check again later; if so
				// we will fall through and try to start setup from scratch.
				if configContainerID == expectedContainerID {
					return false, nil
				}
				shared.LogInfof(
					reqLogger,
					cr,
					shared.EventReasonMember,
					"previous setup for member{%s} interrupted; re-trying setup",
					podName,
				)
			} else {
				// Setup has previously completed with success or error. If
				// the current container is the container that setup was run
				// on, update LastSetupGeneration to indicate that the last
				// pushed configmeta was processed. Clear any pending notifies
				// because we captured that info as part of setup.
				if configContainerID == expectedContainerID {
					stateDetail.LastSetupGeneration = stateDetail.LastConfigDataGeneration
					stateDetail.PendingNotifyCmds = []*kdv1.NotificationDesc{}
				}
				status, convErr := strconv.Atoi(configStatus)
				if convErr == nil && status == 0 {
					return true, nil
				}
				statusErr := fmt.Errorf(
					"configure failed with exit status {%s}",
					configStatus,
				)
				return true, statusErr
			}
		}
	}
	// We haven't successfully started the configure script yet.
	// Don't do anything yet if the app requires systemd and systemd is not
	// responsive yet.
	systemdOk, systemdErr := systemdOk(
		reqLogger,
		cr,
		podName,
		expectedContainerID,
	)
	if systemdErr != nil {
		// Some problem trying to check systemd.
		return true, systemdErr
	}
	if !systemdOk {
		// Systemd not responsive yet; try again later.
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonMember,
			"systemd not yet responsive in member{%s}",
			podName,
		)
		return false, nil
	}
	// Now upload the configmeta file.
	configmetaErr := executor.CreateFile(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		configMetaFile,
		strings.NewReader(configmetaGenerator(podName)),
	)
	if configmetaErr != nil {
		return true, configmetaErr
	}
	// Successfully injected configmeta so record that.
	stateDetail.LastConfigDataGeneration = cr.Status.SpecGenerationToProcess
	// Set up configcli package for this member (if not set up already).
	prepErr := setupNodePrep(reqLogger, cr, podName, expectedContainerID)
	if prepErr != nil {
		return true, prepErr
	}
	// Make sure the necessary app-specific materials are in place.
	setupErr := setupAppConfig(reqLogger, cr, setupURL, podName, expectedContainerID, roleName)
	if setupErr != nil {
		return true, setupErr
	}
	// Run the config file iff the event is registered during initial configuration.
	appCr, appErr := catalog.GetApp(cr)
	if appErr != nil {
		shared.LogError(
			reqLogger,
			appErr,
			cr,
			shared.EventReasonCluster,
			"app referenced by cluster does not exist",
		)
		return true, appErr
	}
	role := catalog.GetRoleFromID(appCr, roleName)
	if role.EventList != nil && !shared.StringInList("configure", *role.EventList) {
		return true, nil
	}
	// Now kick off the initial config.
	cmd := fmt.Sprintf(appPrepConfigRunCmd, expectedContainerID)
	cmdErr := executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		expectedContainerID,
		executor.AppContainerName,
		"app config",
		strings.NewReader(cmd),
	)
	if cmdErr != nil {
		return true, cmdErr
	}
	return false, nil
}

// queueNotify prepares the info for handling a lifecycle event to a currently
// ready node, and adds the info to the node's notification queue. We are
// notifying about new members either being added to the modifiedRole (if it
// has members in creating state) or being removed (if it has members in
// delete pending state).
func queueNotify(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	stateDetail *kdv1.MemberStateDetail,
	roleName string,
	modifiedRole *roleInfo,
) {

	// Figure out which lifecycle event we're dealing with, and collect the
	// FQDNs of the affected members.
	op := ""
	deltaFqdns := ""
	if creatingOrCreated, ok := modifiedRole.membersByState[memberCreating]; ok {
		// At the time this function is called, members in this list are
		// marked as creating, ready, or config error. The fqdnsList function
		// will appropriately skip the ones that are still creating, or the
		// ones in other states that are just reboots.
		op = "addnodes"
		deltaFqdns = fqdnsList(cr, creatingOrCreated)
	}
	if op == "" {
		if deletePending, ok := modifiedRole.membersByState[memberDeletePending]; ok {
			op = "delnodes"
			deltaFqdns = fqdnsList(cr, deletePending)
		}
	}

	if deltaFqdns == "" {
		// No nodes actually being created/deleted. One example of this
		// is in the creating case where none have been successfully
		// configured.
		return
	}
	// Notify the node iff the event is registered during initial configuration.
	appCr, appErr := catalog.GetApp(cr)
	if appErr != nil {
		shared.LogError(
			reqLogger,
			appErr,
			cr,
			shared.EventReasonCluster,
			"app referenced by cluster does not exist")
	}
	role := catalog.GetRoleFromID(appCr, roleName)
	if role.EventList != nil && !shared.StringInList(op, *role.EventList) {
		return
	}
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonNoEvent,
		"will notify member{%s}: %s",
		podName,
		op,
	)
	// Compose the notify command arguments.
	arguments := []string{
		"--" + op,
		"--nodegroup 1", // currently only 1 nodegroup possible
		"--role",
		modifiedRole.roleStatus.Name,
		"--fqdns",
		deltaFqdns,
	}
	notifyDesc := kdv1.NotificationDesc{
		Arguments: arguments,
	}
	stateDetail.PendingNotifyCmds = append(
		stateDetail.PendingNotifyCmds,
		&notifyDesc,
	)
}

// fqdnsList generates a comma-separated list of FQDNs given a list of members.
func fqdnsList(
	cr *kdv1.KubeDirectorCluster,
	members []*kdv1.MemberStatus,
) string {

	getMemberFqdn := func(m *kdv1.MemberStatus) string {
		s := []string{
			m.Pod,
			cr.Status.ClusterService,
			cr.Namespace + shared.GetSvcClusterDomainBase(),
		}
		return strings.Join(s, ".")
	}
	numMembers := len(members)
	fqdns := make([]string, 0, numMembers)
	for i := 0; i < numMembers; i++ {
		// Grab any member in the deletePending state.
		if members[i].State == memberDeletePending {
			fqdns = append(fqdns, getMemberFqdn(members[i]))
			continue
		}
		// Skip any member in the creating state, since it has not been
		// successfully configured. Also skip any member with
		// lastConfiguredContainer already set since it is a reboot.
		if (members[i].State != memberCreating) &&
			(members[i].StateDetail.LastConfiguredContainer == "") {
			fqdns = append(fqdns, getMemberFqdn(members[i]))
		}
	}
	return strings.Join(fqdns, ",")
}

// getConnectionVersion will fetch the HashChangeIncrementor from the cluster Annotations
// if the Annotation isn't available or, for some reason cannot be parsed to an int64,
// we get the connection version from members that are in "Ready" state
func getConnectionVersion(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) int64 {
	if connectionsVersionStr, ok := cr.Annotations[shared.HashChangeIncrementor]; ok {
		connectionsVersion, connVersionError := strconv.ParseInt(connectionsVersionStr, 10, 64)
		if connVersionError != nil {
			shared.LogErrorf(
				reqLogger,
				connVersionError,
				cr,
				shared.EventReasonMember,
				"Invalid connectionsIncrementor for role{%s}",
				role.roleStatus.Name,
			)
			return getDefaultConnectionVersion(reqLogger, cr, role)

		}
		return connectionsVersion
	}
	return getDefaultConnectionVersion(reqLogger, cr, role)
}

// getDefaultConnectionVersion returns a connection version by parsing the version of members in "ready" state
// if no member is in "Ready" state, we return 0.
// otherwise, we return the smallest LastConnectionVersion from all the members.
// This is because we are better off running --reconnect when no connection has changed, rather than not running --reconnect when a connection change does happen
func getDefaultConnectionVersion(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) int64 {
	ready := role.membersByState[memberReady]
	if len(ready) == 0 {
		x := int64(0)
		return x
	}
	min := int64(math.MaxInt64)
	for _, memberStatus := range ready {
		memberVersion := *memberStatus.StateDetail.LastConnectionVersion
		if min > memberVersion {
			min = memberVersion
		}
	}
	return min
}
