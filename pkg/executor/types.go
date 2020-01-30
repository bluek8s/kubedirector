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

package executor

import (
	"io"

	"github.com/bluek8s/kubedirector/pkg/shared"
)

const (
	// ClusterLabel is a label placed on every created statefulset, pod, and
	// service, with a value of the KubeDirectorCluster CR name.
	// XXX change this, add group
	ClusterLabel = "kubedirectorcluster"
	// ClusterRoleLabel is a label placed on every created pod, and
	// (non-headless) service, with a value of the relevant role ID.
	// XXX change this, add group
	ClusterRoleLabel     = "role"
	headlessServiceLabel = shared.KdDomainBase + "/" + "headless"
	statefulSetPodLabel  = "statefulset.kubernetes.io/pod-name"
	storageClassName     = "volume.beta.kubernetes.io/storage-class"
	// AppContainerName is the name of KubeDirector app containers.
	AppContainerName = "app"
	// PvcNamePrefix (along with a hyphen) is prepended to the name of each
	// member PVC name that is auto-created for a statefulset.
	PvcNamePrefix         = "p"
	svcNamePrefix         = "s-"
	statefulSetNamePrefix = "kdss-"
	headlessSvcNamePrefix = "kdhs-"
	initContainerName     = "init"
	execShell             = "bash"
	configMetaFile        = "/etc/guestconfig/configmeta.json"
	cgroupFSVolume        = "/sys/fs/cgroup"
	systemdFSVolume       = "/sys/fs/cgroup/systemd"
	tmpFSVolSize          = "20Gi"
	kubedirectorInit      = "/etc/kubedirector.init"
)

// Streams for stdin, stdout, stderr of executed commands
type Streams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}
