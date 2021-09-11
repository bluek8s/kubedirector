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

package shared

const (
	// DefaultSvcDomainBase contains the initial segments used to build FQDNs
	// for cluster members
	DefaultSvcDomainBase = ".svc.cluster.local"

	// KubeDirectorNamespaceEnvVar is the constant for env variable MY_NAMESPACE
	// which is the namespace of the kubedirector pod.
	KubeDirectorNamespaceEnvVar = "MY_NAMESPACE"

	// KubeDirectorGlobalConfig is the name of the kubedirector config CR
	KubeDirectorGlobalConfig = "kd-global-config"

	// KdDomainBase is the prefix for label and annotation keys.
	KdDomainBase = "kubedirector.hpe.com"

	// RestoringLabel is the label placed on a kdcluster while it and objects
	// it depends on are being restored from a backup.
	RestoringLabel = KdDomainBase + "/restoring"

	// StatusBackupAnnotation is the annotation placed on a kdcluster when
	// writing status, to indicate whether or not a status backup exists.
	StatusBackupAnnotation = KdDomainBase + "/status-backup-exists"

	// DefaultServiceType - default service type if not specified in
	// the configCR
	DefaultServiceType = "LoadBalancer"

	// DefaultNamingScheme - default naming scheme if not specified in
	// the configCR
	DefaultNamingScheme = "UID"
)

// Event reason constants for recording events
const (
	EventReasonNoEvent   = ""
	EventReasonCluster   = "Cluster"
	EventReasonRole      = "Role"
	EventReasonMember    = "Member"
	EventReasonConfig    = "Config"
	EventReasonConfigMap = "ConfigMap"
	EventReasonSecret    = "Secret"
)

// Settings for appCatalog
const (
	AppCatalogLocal  = "local"
	AppCatalogSystem = "system"
)

// Used by configmap, secret and cluster reconciler to update connection
// changes
const (
	ConnectionsIncrementor = KdDomainBase + "/connUpdateCounter"
)

// Used as a counter for number of times the hash of connections changes, which is an indicator of the number of times the connections change
const (
	HashChangeIncrementor = KdDomainBase + "/hashChangeCounter"
)

// connUpdateCounter is updated whenever a connection object is updated/created.
// hashchangrincrementor is updated whenever the hash of connected object changes. It includes connection object CRUD changes.
