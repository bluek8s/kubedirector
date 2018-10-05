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
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/core/v1"
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
					handleReadyMembers(cr, r, configmetaGenerator)
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
		// Only handle creating state members when there are no create_pending.
		if _, ok := r.membersByState[memberCreatePending]; ok {
			handleCreatePendingMembers(cr, r)
		} else if _, ok := r.membersByState[memberCreating]; ok {
			handleCreatingMembers(cr, r, roles, configmetaGenerator)
		}
		if _, ok := r.membersByState[memberDeletePending]; ok {
			handleDeletePendingMembers(cr, r, roles)
		}
		if _, ok := r.membersByState[memberDeleting]; ok {
			handleDeletingMembers(cr, r)
		}
	}

	return nil
}

// handleReadyMembers operates on all members in the role that are currently
// in the ready state. It will update the configmeta inside each guest with
// the latest content.
func handleReadyMembers(
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
				shared.LogWarnf(
					cr,
					"failed to find member{%s}: %v",
					m.Pod,
					podGetErr,
				)
				allReadyFinished = false
				return
			}
			// Only attempt to push the file if the pod is running.
			if pod.Status.Phase != v1.PodRunning {
				// We don't treat this as a problem; pod will get updated
				// later.
				return
			}
			configmeta := configmetaGenerator(m.Pod)
			createFileErr := executor.CreateFile(
				cr,
				m.Pod,
				configMetaFile,
				strings.NewReader(configmeta),
			)
			if createFileErr != nil {
				shared.LogWarnf(
					cr,
					"failed to update config in member{%s}: %v",
					m.Pod,
					createFileErr,
				)
				allReadyFinished = false
				return
			}
		}(member)
	}
	wgReady.Wait()
	if !allReadyFinished {
		// Will try again on next handler pass.
		return false
	}
	return true
}

// handleCreatePendingMembers operates on all members in the role that are
// currently in the create_pending state. It first adjusts the statefulset
// replicas count as necessary, then checks each new member to see if it is
// running. If so, it moves it to the creating state. It is quite possible for
// members to be left in the create_pending state across multiple handler
// passes.
func handleCreatePendingMembers(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	// Fix statefulset if necessary, and bail out if it is not good yet.
	if !checkMemberCount(cr, role) {
		return
	}
	if !replicasSynced(cr, role) {
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
				// Can't get the pod. Skip it and try again later.
				shared.LogWarnf(
					cr,
					"failed to find member{%s}: %v",
					m.Pod,
					podGetErr,
				)
				return
			}
			if pod.Status.Phase == v1.PodRunning {
				m.State = string(memberCreating)
				// We don't need to update membersByState because these
				// members won't be processed until a subsequent handler pass
				// anyway.
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
// creating state and we'll tackle them again on next handler pass.
func handleCreatingMembers(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
	configmetaGenerator func(string) string,
) {

	creating := role.membersByState[memberCreating]

	// Fetch setup url package
	setupUrl, setupUrlErr := catalog.AppSetupPackageUrl(cr, role.roleStatus.Name)
	if setupUrlErr != nil {
		shared.LogWarn(
			cr,
			"failed to fetch setup url",
		)
		return
	}
	// Perform setup on all of these members.
	var wgSetup sync.WaitGroup
	wgSetup.Add(len(creating))
	for _, member := range creating {
		go func(m *kdv1.MemberStatus) {
			defer wgSetup.Done()
			configmeta := configmetaGenerator(m.Pod)
			createFileErr := executor.CreateFile(
				cr,
				m.Pod,
				configMetaFile,
				strings.NewReader(configmeta),
			)
			if createFileErr != nil {
				// We'll try again next pass.
				shared.LogWarnf(
					cr,
					"failed to update config in member{%s}: %v",
					m.Pod,
					createFileErr,
				)
				return
			}

			// If setup package is not present, skip doing nodeprep and appconfig
			if setupUrl == "" {
				// Set a temporary state used below for notifies.
				m.State = string(memberConfigured)
				return
			}

			// Set up nodeprep package for this member (if not set up already).
			prepErr := setupNodePrep(cr, m.Pod)
			if prepErr != nil {
				shared.LogWarnf(
					cr,
					"failed to set up nodeprep package in member{%s}: %v",
					m.Pod,
					prepErr,
				)
				return
			}

			// AGENT TBD
			/*
				if agent is to be used
					if agent is not already on pod (executor.IsFileExists)
						use executor.CreateFile and executor.RunScript to push agent
			*/

			// Set up app config package for this member (if not set up already).
			setupErr := setupAppConfig(cr, setupUrl, m.Pod, role.roleStatus.Name)
			if setupErr != nil {
				shared.LogWarnf(
					cr,
					"failed to set up appconfig package in member{%s}: %v",
					m.Pod,
					setupErr,
				)
				return
			}

			// Now use it for initial configuration.
			configErr := runAppConfig(cr, m.Pod, role.roleStatus.Name, nil)
			if configErr != nil {
				shared.LogWarnf(
					cr,
					"failed to run initial config for member{%s}: %v",
					m.Pod,
					configErr,
				)
				return
			}
			// Set a temporary state used below for notifies.
			m.State = string(memberConfigured)
		}(member)
	}
	wgSetup.Wait()

	// Now let any ready nodes know that some new nodes have appeared.
	if setupUrl != "" {
		if !notifyReadyNodes(cr, role, allRoles) {
			shared.LogWarn(
				cr,
				"failed to notify all ready nodes for addnodes event",
			)
		}
	}

	// All done, change state for the ones that we configured. We don't need
	// to update membersByState because these members won't be processed again
	// until a subsequent handler pass anyway.
	for _, member := range creating {
		if member.State == string(memberConfigured) {
			member.State = string(memberReady)
		}
	}
}

// handleDeletePendingMembers operates on all members in the role that are
// currently in the delete_pending state. It first notifies all ready members
// in the cluster of the impending deletion; then it moves all of these
// delete_pending members to the deleting state.
func handleDeletePendingMembers(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	allRoles []*roleInfo,
) {

	// Let any ready nodes know that some nodes will disappear,
	setupUrl, setupUrlErr := catalog.AppSetupPackageUrl(cr, role.roleStatus.Name)
	if setupUrlErr != nil {
		shared.LogWarn(
			cr,
			"failed to fetch setup url",
		)
		return
	}
	if setupUrl != "" {
		if !notifyReadyNodes(cr, role, allRoles) {
			shared.LogWarn(
				cr,
				"failed to notify all ready nodes for delnodes event",
			)
		}
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

// handleDeletingMembers operates on all members in the role that are
// currently in the deleting state. If the replicas count on the statefulset
// has not been successfully updated yet, it attempts that change and returns.
// Otherwise it checks each pod to see if it is gone, and if so deletes the
// corresponding PVC and service. Once all member-related objects are gone,
// the member status is marked for removal. It is quite possible for members
// to be left in the deleting state across multiple handler passes.
func handleDeletingMembers(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) {

	// Fix statefulset if necessary, and bail out if it is not good yet.
	if !checkMemberCount(cr, role) {
		return
	}
	if !replicasSynced(cr, role) {
		return
	}

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
				shared.LogWarnf(
					cr,
					"failed to find member{%s}: %v",
					m.Pod,
					podGetErr,
				)
				return
			}
			if m.Service != "" {
				serviceDelErr := executor.DeletePodService(
					cr.Namespace,
					m.Service,
				)
				if serviceDelErr == nil || errors.IsNotFound(serviceDelErr) {
					m.Service = ""
				} else {
					shared.LogWarnf(
						cr,
						"failed to delete service{%s}: %v",
						m.Service,
						serviceDelErr,
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
					shared.LogWarnf(
						cr,
						"failed to delete PVC{%s}: %v",
						m.PVC,
						pvcDelErr,
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

// checkMemberCount examines an existing statefulset to see if its replicas
// count needs to be reconciled, and does so if necessary. Return false if the
// statefulset had to be changed.
func checkMemberCount(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) bool {

	// Calculate the number of members that a statefulset/role SHOULD
	// currently have. Don't use roleSpec here. roleSpec could flap around and
	// we'll ignore it if we're still working on a previous change.
	replicas := int32(len(role.membersByState[memberCreatePending]) +
		len(role.membersByState[memberCreating]) +
		len(role.membersByState[memberReady]))

	// Fix the statefulset if we haven't successfully resized it yet.
	if *(role.statefulSet.Spec.Replicas) != replicas {
		shared.LogInfof(
			cr,
			"changing replicas count for role{%s}: %v -> %v",
			role.roleStatus.Name,
			*(role.statefulSet.Spec.Replicas),
			replicas,
		)
		updateErr := executor.UpdateStatefulSetReplicas(
			cr,
			replicas,
			role.statefulSet)
		if updateErr != nil {
			shared.LogWarnf(
				cr,
				"failed to change StatefulSet{%s} replicas: %v",
				role.statefulSet.Name,
				updateErr,
			)
		}
		return false
	}

	return true
}

// replicasSynced returns true if the role's statefulset has its status
// replicas count matching its spec replicas count.
func replicasSynced(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
) bool {

	if role.statefulSet.Status.Replicas != *(role.statefulSet.Spec.Replicas) {
		shared.LogInfof(
			cr,
			"waiting for replicas count for role{%s}: %v -> %v",
			role.roleStatus.Name,
			role.statefulSet.Status.Replicas,
			*(role.statefulSet.Spec.Replicas),
		)
		return false
	}

	return true
}

// setupNodePrep injects the nodeprep package (bdvcli et al) into the member's
// container and installs it.
func setupNodePrep(
	cr *kdv1.KubeDirectorCluster,
	podName string,
) error {

	// Check to see if the destination file exists already, in which case just
	// return
        fileExists, fileError := executor.IsFileExists(cr, podName, nodePrepTestFile) 
	if fileError != nil || !fileExists {
		return nil
	}

	// Inject the nodeprep package, taken from the KubeDirector's container.
	nodePrepFile, openErr := os.Open(nodePrepSrcFile)
	if openErr != nil {
		return fmt.Errorf(
			"failed to open file %s: %v",
			nodePrepSrcFile,
			openErr,
		)
	}
	defer nodePrepFile.Close()
	createErr := executor.CreateFile(
		cr,
		podName,
		nodePrepDestFile,
		bufio.NewReader(nodePrepFile),
	)
	if createErr != nil {
		return createErr
	}

	// Install it,
	return executor.RunScript(
		cr,
		podName,
		"nodeprep setup",
		strings.NewReader(nodePrepInstallCmd),
	)
}

// setupAppConfig injects the app setup package (if any) into the member's
// container and installs it.
func setupAppConfig(
	cr *kdv1.KubeDirectorCluster,
	setupUrl string,
	podName string,
	roleName string,
) error {

	// Check to see if the destination file exists already, in which case just
	// return.
        fileExists, fileError := executor.IsFileExists(cr, podName, appPrepStartscript) 
	if fileError != nil || ! fileExists {
		return nil
	}

	// Fetch and install it.
	cmd := strings.Replace(appPrepInitCmd, "{{APP_CONFIG_URL}}", setupUrl, -1)
	return executor.RunScript(
		cr,
		podName,
		"appconfig setup",
		strings.NewReader(cmd),
	)
}

// notifyReadyNodes sends a lifecycle event notification to all ready nodes
// in the cluster, informing about changes in the indicated role.
func notifyReadyNodes(
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
		if ready, ok := otherRole.membersByState[memberReady]; ok {
			for _, member := range ready {
				go func(m *kdv1.MemberStatus, r *roleInfo) {
					defer wgReady.Done()
					configErr := runAppConfig(cr, m.Pod, r.roleStatus.Name, role)
					if configErr != nil {
						shared.LogWarnf(
							cr,
							"failed to notify member{%s}: %v",
							m.Pod,
							configErr,
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

// runAppConfig notifies a member's app setup script, if any, about cluster
// lifecycle events. If otherRole is nil, this is a notification for the
// designated pod about its own creation. Otherwise we are notifying about new
// memmbers either being added to the otherRole (if it has members in
// creating state) or being removed (if it has members in delete_pending
// state).
func runAppConfig(
	cr *kdv1.KubeDirectorCluster,
	podName string,
	roleName string,
	otherRole *roleInfo,
) error {

	// Figure out which lifecycle event we're dealing with. If this is noi
	// the initial configure event, also collect the FQDNs of the affected
	// members.
	op := ""
	deltaFqdns := ""
	if otherRole == nil {
		op = "configure"
	} else {
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
	}

	// Compose the command args and run it.
	cmd := appPrepStartscript + " --" + op
	if op != "configure" {
		// Currently only 1 nodegroup possible in KubeDirector cluster.
		cmd = cmd + " --nodegroup 1"
		// identify the role of the changed nodes
		cmd = cmd + " --role " + otherRole.roleStatus.Name
		// and finally list the FQDNs of the changed nodes
		cmd = cmd + " --fqdns " + deltaFqdns
	}
	return executor.RunScript(
		cr,
		podName,
		"appconfig",
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
			cr.Namespace + shared.DomainBase,
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
