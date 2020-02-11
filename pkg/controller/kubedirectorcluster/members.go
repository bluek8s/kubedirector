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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-logr/logr"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.hpe.com/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// syncMembers is responsible for adding or deleting members. It is the only
// function in this file that is invoked from another file (from the
// syncCluster function in cluster.go). Along with k8s interactions (changing
// statefulset replica count), this involves notifying existing members about
// additions/deletions, and doing any necessary agent installation and/or
// triggering application setup. This function will modify the member status
// data structures to update their states.
func syncMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	roles []*roleInfo,
	membersHaveChanged bool,
	configmetaGenerator func(string) string,
) error {

	// Notify current ready members about membership changes.
	readyMembersUpdated := true
	if membersHaveChanged {
		for _, r := range roles {
			if _, ok := r.membersByState[memberReady]; ok {
				readyMembersUpdated = readyMembersUpdated &&
					handleReadyMembers(reqLogger, cr, r, configmetaGenerator)
			}
		}
	}
	if !readyMembersUpdated {
		// Not an error, we're just not done yet.
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

// handleReadyMembers operates on all members in the role that are currently
// in the ready state. It will update the configmeta inside each guest with
// the latest content.
func handleReadyMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	configmetaGenerator func(string) string,
) bool {

	ready := role.membersByState[memberReady]
	allReadyFinished := true
	var wgReady sync.WaitGroup
	wgReady.Add(len(ready))
	for _, member := range ready {
		go func(m *kdv1.MemberStatus) {
			defer wgReady.Done()
			pod, podGetErr := observer.GetPod(cr.Namespace, m.Pod)
			if podGetErr != nil {
				// Can't get the pod. Skip it and try again later.
				shared.LogErrorf(
					reqLogger,
					podGetErr,
					cr,
					shared.EventReasonMember,
					"failed to find member{%s} in role{%s}",
					m.Pod,
					role.roleStatus.Name,
				)
				allReadyFinished = false
				return
			}
			// Only attempt to push the file if the pod is running.
			if pod.Status.Phase != corev1.PodRunning {
				// We don't treat this as a problem; pod will get updated
				// later.
				return
			}
			configmeta := configmetaGenerator(m.Pod)
			createFileErr := executor.CreateFile(
				reqLogger,
				cr,
				cr.Namespace,
				m.Pod,
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
				allReadyFinished = false
				return
			}
		}(member)
	}
	wgReady.Wait()
	if !allReadyFinished {
		// Will try again on next reconciler pass.
		return false
	}
	return true
}

// handleCreatePendingMembers operates on all members in the role that are
// currently in the create_pending state. It first adjusts the statefulset
// replicas count as necessary, then checks each new member to see if it is
// running. If so, it moves it to the creating state. It is quite possible for
// members to be left in the create_pending state across multiple reconciler
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
				if errors.IsNotFound(podGetErr) {
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
				m.State = string(memberCreating)
				// We don't need to update membersByState; the newly
				// creating-state members will be processed on a subsequent
				// reconciler pass.
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

			// Check to see if we have to inject one or more files for this member
			if len(role.roleSpec.FileInjections) != 0 {
				injectErr := injectFiles(reqLogger, cr, m.Pod, role)
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
					m.State = string(memberConfigError)
					return
				}
			}

			if setupURL == "" {
				// Leave this in memberConfigured state so, we don't send
				// ready notifications to itself below. The next reconciler cycle
				// will handle this appropriately.
				m.State = string(memberConfigured)

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
				m.State = string(memberConfigError)
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
			// Set a temporary state used below so we won't send notifies
			// to this member yet.
			m.State = string(memberConfigured)
		}(member)
	}
	wgSetup.Wait()

	// Now let any ready nodes know that some new nodes have appeared.
	if !notifyReadyNodes(reqLogger, cr, role, allRoles) {
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"failed to notify all ready nodes for addnodes event",
		)
	}

	// All done, change state for the ones that we configured. We don't need
	// to update membersByState because these members won't be processed again
	// until a subsequent reconciler pass anyway.
	for _, member := range creating {
		if member.State == string(memberConfigured) {
			member.State = string(memberReady)
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
			} else if !errors.IsNotFound(podGetErr) {
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
				if serviceDelErr == nil || errors.IsNotFound(serviceDelErr) {
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
				if pvcDelErr == nil || errors.IsNotFound(pvcDelErr) {
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
// currently in the delete_pending state. It first notifies all ready members
// in the cluster of the impending deletion; then it moves all of these
// delete_pending members to the deleting state.
func handleDeletePendingMembers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
) {

	if !notifyReadyNodes(reqLogger, cr, role, allRoles) {
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"failed to notify all ready nodes for delnodes event",
		)
	}

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
) error {

	// Check to see if the destination file exists already, in which case just
	// return. Also bail out if we cannot manage to check file existence.
	fileExists, fileError := executor.IsFileExists(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
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
		executor.AppContainerName,
		configcliDestFile,
		bufio.NewReader(nodePrepFile),
	)
	if createErr != nil {
		return createErr
	}

	// Install it,
	return executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
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
	roleName string,
) error {

	// Check to see if the destination file exists already, in which case just
	// return. Also bail out if we cannot manage to check file existence.
	fileExists, fileError := executor.IsFileExists(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
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
		executor.AppContainerName,
		"app config setup",
		strings.NewReader(cmd),
	)
}

// injectFiles injects one or more files as specified through role spec
// Each file will be downloaded to the specified location inside the pod and
// file permissions and ownership will be updated based on the spec
func injectFiles(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
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

// notifyReadyNodes sends a lifecycle event notification to all ready nodes
// in the cluster, informing about changes in the indicated role.
func notifyReadyNodes(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
) bool {

	totalReady := 0
	for _, rCheck := range allRoles {
		if ready, ok := rCheck.membersByState[memberReady]; ok {
			totalReady += len(ready)
		}
	}
	if totalReady == 0 {
		return true
	}
	allNotifyFinished := true
	var wgReady sync.WaitGroup
	wgReady.Add(totalReady)
	for _, otherRole := range allRoles {
		if len(otherRole.membersByState[memberReady]) == 0 {
			// This is not just an optimization; note also that in the case
			// of a role with zero members (not just zero READY members)
			// then otherRole.roleStatus referenced below will be nil.
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
		if ready, ok := otherRole.membersByState[memberReady]; ok {
			for _, member := range ready {
				go func(m *kdv1.MemberStatus, r *roleInfo) {
					defer wgReady.Done()

					if setupURL == "" {
						// No notification necessary for this role
						shared.LogInfof(
							reqLogger,
							cr,
							shared.EventReasonMember,
							"notify skipped for member{%s} in role{%s}",
							m.Pod,
							r.roleStatus.Name,
						)
						return
					}

					configErr := appReConfig(reqLogger, cr, m.Pod, r.roleStatus.Name, role)
					if configErr != nil {
						shared.LogErrorf(
							reqLogger,
							configErr,
							cr,
							shared.EventReasonMember,
							"failed to notify member{%s} in role{%s}",
							m.Pod,
							role.roleStatus.Name,
						)
						allNotifyFinished = false
						return
					}
				}(member, otherRole)
			}
		}
	}
	wgReady.Wait()
	return allNotifyFinished
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
	roleName string,
	configmetaGenerator func(string) string,
) (bool, error) {

	// For initial configuration, startscript will run asynchronously and we
	// will check back periodically. So let's have a look at the existing
	// status if any.
	var statusStrB strings.Builder
	fileExists, fileError := executor.ReadFile(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		executor.AppContainerName,
		appPrepConfigStatus,
		&statusStrB,
	)
	if fileError != nil {
		return true, fileError
	}

	if fileExists {
		// Configure script was previously started.
		statusStr := statusStrB.String()
		if statusStr == "" {
			// Script is still running.
			return false, nil
		}
		// All done, what status did we get?
		status, convErr := strconv.Atoi(statusStr)
		if convErr == nil && status == 0 {
			return true, nil
		}
		statusErr := fmt.Errorf(
			"configure failed with exit status {%s}",
			statusStr,
		)
		return true, statusErr
	}
	// We haven't successfully started the configure script yet.
	// First upload the configmeta file
	configmetaErr := executor.CreateFile(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		executor.AppContainerName,
		configMetaFile,
		strings.NewReader(configmetaGenerator(podName)),
	)
	if configmetaErr != nil {
		return true, configmetaErr
	}
	// Set up configcli package for this member (if not set up already).
	prepErr := setupNodePrep(reqLogger, cr, podName)
	if prepErr != nil {
		return true, prepErr
	}
	// Make sure the necessary app-specific materials are in place.
	setupErr := setupAppConfig(reqLogger, cr, setupURL, podName, roleName)
	if setupErr != nil {
		return true, setupErr
	}
	// Now kick off the initial config.
	cmdErr := executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		executor.AppContainerName,
		"app config",
		strings.NewReader(appPrepConfigRunCmd),
	)
	if cmdErr != nil {
		return true, cmdErr
	}
	return false, nil
}

// appReConfig notifies a member's app setup script, if any, about cluster
// lifecycle events after initial configuration. We are notifying about new
// members either being added to the otherRole (if it has members in
// creating state) or being removed (if it has members in delete_pending
// state).
func appReConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	roleName string,
	otherRole *roleInfo,
) error {

	// Figure out which lifecycle event we're dealing with, and collect the
	// FQDNs of the affected members.
	op := ""
	deltaFqdns := ""
	if creating, ok := otherRole.membersByState[memberCreating]; ok {
		// Members in this list are either marked with the creating state
		// or configured_internal. The fqdnsList function will appropriately
		// skip the ones in the creating state since they are unconfigured.
		op = "addnodes"
		deltaFqdns = fqdnsList(cr, creating)
	}
	if op == "" {
		if deletePending, ok := otherRole.membersByState[memberDeletePending]; ok {
			op = "delnodes"
			deltaFqdns = fqdnsList(cr, deletePending)
		}
	}
	if deltaFqdns == "" {
		// No nodes actually being created/deleted. One example of this
		// is in the creating case where none have been successfully
		// configured.
		return nil
	}

	// Compose and run the command line.
	cmd := strings.Join(
		[]string{
			appPrepStartscript,
			"--" + op,
			"--nodegroup 1", // currently only 1 nodegroup possible
			"--role",
			otherRole.roleStatus.Name,
			"--fqdns",
			deltaFqdns,
		},
		" ",
	)
	return executor.RunScript(
		reqLogger,
		cr,
		cr.Namespace,
		podName,
		executor.AppContainerName,
		"app reconfig",
		strings.NewReader(cmd),
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
		// Skip any member in the creating state, since it has not been
		// successfully configured.
		if members[i].State != memberCreating {
			fqdns = append(fqdns, getMemberFqdn(members[i]))
		}
	}
	return strings.Join(fqdns, ",")
}
