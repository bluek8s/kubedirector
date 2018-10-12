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

package executor

import (
	"io"
)

const (
	headlessServiceLabel = "kubedirector.bluedata.io/headless"
	statefulSetPodLabel  = "statefulset.kubernetes.io/pod-name"
	storageClassName     = "volume.beta.kubernetes.io/storage-class"
	appContainerName     = "app"
	pvcName              = "pvc"
	headlessServiceName  = "svc"
	initContainerName    = "init"
	execShell            = "bash"
	configMetaFile       = "/etc/guestconfig/configmeta.json"
	cgroupFsVolume       = "/sys/fs/cgroup"
	systemdFsVolume      = "/sys/fs/cgroup/systemd"
	tmpFsVolSize         = "20Gi"
	finalizerId          = "kubedirector.bluedata.io/cleanup"
)

type streams struct {
	in     io.Reader
	out    io.Writer
	errOut io.Writer
}
