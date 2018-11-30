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
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
)

// ReadStatusGen provides threadsafe read of a status gen UID string and
// validated flag.
func ReadStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) (StatusGen, bool) {
	handler.lock.RLock()
	defer handler.lock.RUnlock()
	val, ok := handler.ClusterState.clusterStatusGens[cr.UID]
	return val, ok
}

// writeStatusGen provides threadsafe write of a status gen UID string.
// The validated flag will begin as false.
func writeStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
	newGenUid string,
) {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	handler.ClusterState.clusterStatusGens[cr.UID] = StatusGen{Uid: newGenUid}
}

// ValidateStatusGen provides threadsafe mark-validated of a status gen.
func ValidateStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	val, ok := handler.ClusterState.clusterStatusGens[cr.UID]
	if ok {
		val.Validated = true
		handler.ClusterState.clusterStatusGens[cr.UID] = val
	}
}

// deleteStatusGen provides threadsafe delete of a status gen.
func deleteStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	delete(handler.ClusterState.clusterStatusGens, cr.UID)
}

// ClustersUsingApp returns the list of cluster names referencing the given app.
func ClustersUsingApp(
	app string,
	handler *Handler,
) []string {
	var clusters []string
	handler.lock.RLock()
	defer handler.lock.RUnlock()
	// This is a relationship that needs to be query-able given either ONLY
	// the app name (in this function) or ONLY the cluster name (in
	// removeClusterAppReference). Since the app CR deletion/update triggers
	// for this function are very infrequent, we'll implement this app-name
	// check by just walking the list of associations. It's also nice to go
	// ahead and gather all the offending cluster CR names to report back to
	// the client.
	for clusterKey, appName := range handler.ClusterState.clusterAppTypes {
		if appName == app {
			clusters = append(clusters, clusterKey)
		}
	}
	return clusters
}

// ensureClusterAppReference notes that an app type is in use by this cluster.
func ensureClusterAppReference(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) {
	clusterKey := cr.Namespace + "/" + cr.Name
	handler.lock.Lock()
	defer handler.lock.Unlock()
	handler.ClusterState.clusterAppTypes[clusterKey] = cr.Spec.AppID
}

// removeClusterAppReference notes that an app type is no longer in use by
// this cluster.
func removeClusterAppReference(
	cr *kdv1.KubeDirectorCluster,
	handler *Handler,
) {
	clusterKey := cr.Namespace + "/" + cr.Name
	handler.lock.Lock()
	defer handler.lock.Unlock()
	delete(handler.ClusterState.clusterAppTypes, clusterKey)
}

func removeGlobalConfig(handler *Handler) {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	handler.GlobalConfig = nil
}

func addGlobalConfig(
	handler *Handler,
	cr *kdv1.KubeDirectorConfig,
) {
	handler.lock.Lock()
	defer handler.lock.Unlock()
	handler.GlobalConfig = cr
}
