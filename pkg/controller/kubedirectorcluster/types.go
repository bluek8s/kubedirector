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
	appPrepConfigStatus = "/opt/guestconfig/configure.status"
	appPrepConfigRunCmd = `rm -f /opt/guestconfig/configure.* &&
	echo -n %s= > ` + appPrepConfigStatus + ` && 
	nohup sh -c '` + appPrepStartscript + ` --configure 2>/opt/guestconfig/configure.stderr 1>/opt/guestconfig/configure.stdout;
	echo -n $? >> ` + appPrepConfigStatus + `' &`
	fileInjectionCommand = `mkdir -p %s && cd %s &&
	curl -L %s -o %s`
	appPrepConfigReconnectCmd = `echo -n %s= > ` + appPrepConfigStatus + ` &&
	nohup sh -c '` + appPrepStartscript + ` --reconnect 2>/opt/guestconfig/configure.stderr 1>/opt/guestconfig/configure.stdout;
	echo -n $? >> ` + appPrepConfigStatus + `' &`
)

// Support for old images/scripts that expect configcli to be in /usr/bin.
const (
	legacyLinksCmd = `ln -sf /usr/local/bin/configcli /usr/bin/bdvcli &&
	ln -sf /usr/local/bin/configcli /usr/bin/bd_vcli &&
	ln -sf /usr/local/bin/configcli /usr/bin/configcli &&
	ln -sf /usr/local/bin/ccli /usr/bin/ccli &&
	ln -sf /usr/local/bin/configmacro /usr/bin/configmacro`
)

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
