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
