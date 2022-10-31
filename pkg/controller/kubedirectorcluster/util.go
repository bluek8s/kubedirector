// Copyright 2022 Hewlett Packard Enterprise Development LP

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
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// updateSchedulingErrorMessage updates MemberStateDetails with SchedulingErrorMessage
func updateSchedulingErrorMessage(
	pod *corev1.Pod,
	memberStatus *kdv1.MemberStatus,
) {

	if memberStatus.StateDetail.LastKnownContainerState == containerMissing {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled {
				if condition.Reason == corev1.PodReasonUnschedulable {
					memberStatus.StateDetail.SchedulingErrorMessage = &condition.Message
				}
			}
		}
	}
}

// RunConfigScript executes an app configuration script
// with different notifications which describe the current cluster state
func RunConfigScript(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	roleName string,
	podName string,
	configArg ConfigArg,
	expectedContainerID string,
	loggingErr bool,
) error {

	cmd := GetAppConfigCmd(expectedContainerID, configArg)
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
	if loggingErr && cmdErr != nil {
		shared.LogErrorf(
			reqLogger,
			cmdErr,
			cr,
			shared.EventReasonMember,
			"failed to run startscript with --{%s} in member{%s} in role{%s}",
			string(configArg),
			podName,
			roleName,
		)
	}
	return cmdErr
}

// QueueNotify prepares the info for handling a lifecycle event to a currently
// ready node, and adds the info to the node's notification queue. We are
// notifying about new members either being added to the modifiedRole (if it
// has members in creating state) or being removed (if it has members in
// delete pending state).
func QueueNotify(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	podName string,
	roleName string,
	// This function should return the operation argument as first value and updated FQDNs as second value
	evalOpFqdnsFn func() (string, string),
) {

	op, deltaFqdns := evalOpFqdnsFn()
	if deltaFqdns == "" && (op == "addnodes" || op == "delnodes") {
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
		roleName,
		"--fqdns",
		deltaFqdns,
	}
	notifyDesc := kdv1.NotificationDesc{
		Arguments: arguments,
	}

	for i, role := range cr.Status.Roles {
		if role.Name == roleName {
			for j, member := range cr.Status.Roles[i].Members {
				if member.Pod == podName {
					(*cr).Status.Roles[i].Members[j].StateDetail.PendingNotifyCmds = append(
						(*cr).Status.Roles[i].Members[j].StateDetail.PendingNotifyCmds, &notifyDesc)
					break
				}
			}
		}
	}
}

// FqdnsList generates a comma-separated list of FQDNs given a list of members.
func FqdnsList(
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
		if (members[i].State != memberCreating) && (members[i].StateDetail.LastConfiguredContainer == "") ||
			(members[i].State == memberReady) && (members[i].StateDetail.LastConfiguredContainer != "") {
			fqdns = append(fqdns, getMemberFqdn(members[i]))
		}
	}

	return strings.Join(fqdns, ",")
}
