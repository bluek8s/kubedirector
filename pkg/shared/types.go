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

package shared

const (
	// DomainBase contains the initial segments used to build FQDNs
	// for cluster members
	DomainBase = ".svc.cluster.local"

	// KubeDirectorNamespaceEnvVar is the constant for env variable MY_NAMESPACE
	// which is the namespace of the kubedirector pod.
	KubeDirectorNamespaceEnvVar = "MY_NAMESPACE"

	// KubeDirectorGlobalConfig is the name of the kubedirector config CR
	KubeDirectorGlobalConfig = "kd-global-config"

	// KubeDirectorFinalizerID is added to kubedirector objects finalizers
	// to prevent them from being deleted before we can clean up.
	KubeDirectorFinalizerID = "kubedirector.bluedata.io/cleanup"
)

// Event reason constants for recording events
const (
	EventReasonNoEvent = ""
	EventReasonCluster = "Cluster"
	EventReasonRole    = "Role"
	EventReasonMember  = "Member"
)
