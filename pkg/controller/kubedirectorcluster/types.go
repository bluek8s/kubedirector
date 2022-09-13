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

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	appsv1 "k8s.io/api/apps/v1"
)

type clusterState string

const (
	clusterCreating clusterState = "creating"
	clusterUpdating              = "updating"
	clusterReady                 = "configured"
	// ClusterSpecModified is exported because it is actually only used by
	// the validator; declaring it here just to keep all cluster states in
	// one spot.
	ClusterSpecModified = "spec modified"
)

type clusterStateInternal int

const (
	clusterMembersChangedUnready clusterStateInternal = iota
	clusterMembersStableUnready
	clusterMembersStableReady
	clusterMembersUnknown
)

type memberState string

const (
	memberCreatePending memberState = "create pending"
	memberCreating                  = "creating"
	memberReady                     = "configured"
	memberDeletePending             = "delete pending"
	memberDeleting                  = "deleting"
	memberConfigError               = "config error"
)

var creatingMemberStates = []string{
	string(memberCreatePending),
	string(memberCreating),
}
var deletingMemberStates = []string{
	string(memberDeletePending),
	string(memberDeleting),
}

const (
	containerRunning      = "running"
	containerWaiting      = "waiting"
	containerInitializing = "initializing"
	containerUnresponsive = "unresponsive"
	containerTerminated   = "terminated"
	containerMissing      = "absent"
	containerUnknown      = "unknown"
)

const (
	configMetaFile         = "/etc/guestconfig/configmeta.json"
	configcliSrcFile       = "/home/kubedirector/configcli.tgz"
	configcliDestFile      = "/tmp/configcli.tgz"
	configcliInstallCmdFmt = `cd /tmp && tar xzf configcli.tgz &&
	chmod u+x /tmp/configcli-*/install && /tmp/configcli-*/install %[1]s &&
	rm -rf /tmp/configcli-* && rm -f /tmp/configcli.tgz &&
	ln -sf %[2]s/bin/configcli %[2]s/bin/bdvcli &&
	ln -sf %[2]s/bin/configcli %[2]s/bin/bd_vcli`
	configcliTestFile       = shared.ConfigCliLoc + "/bin/configcli"
	configcliLegacyTestFile = shared.ConfigCliLegacyLoc + "/bin/configcli"
	appPrepStartscript      = "/opt/guestconfig/*/startscript"
	appPrepInitCmdFmt       = `mkdir -p /opt/guestconfig &&
	chmod 700 /opt/guestconfig &&
	cd /opt/guestconfig &&
	rm -rf /opt/guestconfig/* &&
	curl -L %s -o appconfig.tgz &&
	tar xzf appconfig.tgz &&
	chmod u+x ` + appPrepStartscript + ` &&
	rm -rf /opt/guestconfig/appconfig.tgz`
	appPrepConfigStatus      = "/opt/guestconfig/configure.status"
	appPrepConfigStdout      = "/opt/guestconfig/configure.stdout"
	appPrepConfigStderr      = "/opt/guestconfig/configure.stderr"
	appPrepConfigTemplateCmd = `echo -n %s= > ` + appPrepConfigStatus + ` &&
	nohup sh -c '` + appPrepStartscript + ` --%s 2>` + appPrepConfigStderr + ` 1>` + appPrepConfigStdout + `;
	echo -n $? >> ` + appPrepConfigStatus + `' &`
	fileInjectionCommand = `mkdir -p %s && cd %s &&
	curl -L %s -o %s`
	legacyLinksCmd = `ln -sf /usr/local/bin/configcli /usr/bin/bdvcli &&
	ln -sf /usr/local/bin/configcli /usr/bin/bd_vcli &&
	ln -sf /usr/local/bin/configcli /usr/bin/configcli &&
	ln -sf /usr/local/bin/ccli /usr/bin/ccli &&
	ln -sf /usr/local/bin/configmacro /usr/bin/configmacro`
)

// ConfigArg is enum of possible startscript arguments
type ConfigArg string

const (
	// Configure arg signals the pod is ready to startscript configuring
	ConfigureNotification ConfigArg = "configure"
	// Reconnect arg signals to reset the pod connection version
	ReconnectNotification ConfigArg = "reconnect"
	// PodUpgraded signals the pod app version was upgraded
	PodUpgradedNotification ConfigArg = "pod_upgraded"
	// PodReverted signals the pod app version was reverted after unsuccessful upgrade
	PodRevertedNotification ConfigArg = "pod_reverted"
	// RoleUpgraded signals that all pods of the current role were upgraded
	RoleUpgradedNotification ConfigArg = "role_upgraded"
	// RoleReverted signals that all already upgraded pods of the current role were reverted
	RoleRevertedNotification ConfigArg = "role_reverted"
	// ClusterUpgraded signals that all roles finished their upgrade processes
	ClusterUpgradedNotification ConfigArg = "cluster_upgraded"
	// ClusterReverted signals that all roles finished their rollback processes
	ClusterRevertedNotification ConfigArg = "cluster_reverted"
)

var cmdCache = make(map[ConfigArg]*string)

// GetAppConfigCmd gets cached startscript command template for the specified `arg`
// and substitutes containerId into this template.
// The possible `arg` values currently are: Configure, Reconnect, PodUpgraded, PodReverted */
func GetAppConfigCmd(
	containerID string,
	arg ConfigArg,
) string {

	cmdTemplate := cmdCache[arg]
	if cmdTemplate == nil {
		cmd := fmt.Sprintf(appPrepConfigTemplateCmd, "%s", arg)
		// Clean all outputs before executing command with --run argument
		if arg == ConfigureNotification {
			cmd = `rm -f /opt/guestconfig/configure.* && ` + cmd
		}
		cmdCache[arg] = &cmd
		return GetAppConfigCmd(containerID, arg)
	}

	return fmt.Sprintf(*cmdTemplate, containerID)
}

const (
	zeroPortsService = "n/a"
)

type roleInfo struct {
	statefulSet    *appsv1.StatefulSet
	roleSpec       *kdv1.Role
	roleStatus     *kdv1.RoleStatus
	membersByState map[memberState][]*kdv1.MemberStatus
	desiredPop     int
}
