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
	"sync"

	"github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

type StatusGen struct {
	Uid       string
	Validated bool
}

type handlerClusterState struct {
	clusterStatusGens map[types.UID]StatusGen
	clusterAppTypes   map[string]string
}

type Handler struct {
	lock         sync.RWMutex
	ClusterState handlerClusterState
	GlobalConfig *v1alpha1.KubeDirectorConfig
}

type clusterState string

const (
	clusterCreating clusterState = "creating"
	clusterUpdating              = "updating"
	clusterReady                 = "ready"
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
	memberCreatePending memberState = "create_pending"
	memberCreating                  = "creating"
	memberConfigured                = "configured_internal" // not externally visible
	memberReady                     = "ready"
	memberDeletePending             = "delete_pending"
	memberDeleting                  = "deleting"
	memberConfigError               = "config_error"
)

const (
	configMetaFile     = "/etc/guestconfig/configmeta.json"
	nodePrepSrcFile    = "/root/nodeprep.tgz"
	nodePrepDestFile   = "/tmp/nodeprep.tgz"
	nodePrepInstallCmd = `cd /tmp;tar xzf nodeprep.tgz;
	chmod +x /tmp/nodeprep/install;/tmp/nodeprep/install;
	rm -rf /tmp/nodeprep;rm -f /tmp/nodeprep.tgz`
	nodePrepTestFile   = "/usr/bin/configcli"
	appPrepStartscript = "/opt/guestconfig/*/startscript"
	appPrepInitCmd     = `cd /opt/guestconfig/;
	rm -rf /opt/guestconfig/*;
	curl -L {{APP_CONFIG_URL}} -o appconfig.tgz;
	tar xzf appconfig.tgz;
	chmod +x ` + appPrepStartscript + `;
	rm -rf /opt/guestconfig/appconfig.tgz`
	appPrepConfigStatus = "/opt/guestconfig/configure.status"
	appPrepConfigRunCmd = `rm -f /opt/guestconfig/configure.*;
	touch ` + appPrepConfigStatus + `;
	nohup sh -c "` + appPrepStartscript + ` --configure
	2> /opt/guestconfig/configure.stderr
	1> /opt/guestconfig/configure.stdout;
	echo -n $? > /opt/guestconfig/configure.status" &`
)

type roleInfo struct {
	statefulSet    *appsv1.StatefulSet
	roleSpec       *kdv1.Role
	roleStatus     *kdv1.RoleStatus
	membersByState map[memberState][]*kdv1.MemberStatus
	desiredPop     int
}
