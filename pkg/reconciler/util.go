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
	handlerState *handlerClusterState,
) (StatusGen, bool) {
	handlerState.lock.RLock()
	defer handlerState.lock.RUnlock()
	val, ok := handlerState.clusterStatusGens[cr.UID]
	return val, ok
}

// WriteStatusGen provides threadsafe write of a status gen UID string.
// The validated flag will begin as false.
func WriteStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
	newGenUid string,
) {
	handlerState.lock.Lock()
	defer handlerState.lock.Unlock()
	handlerState.clusterStatusGens[cr.UID] = StatusGen{Uid: newGenUid}
}

// ValidateStatusGen provides threadsafe mark-validated of a status gen.
func ValidateStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
) {
	handlerState.lock.Lock()
	defer handlerState.lock.Unlock()
	val, ok := handlerState.clusterStatusGens[cr.UID]
	if ok {
		val.Validated = true
		handlerState.clusterStatusGens[cr.UID] = val
	}
}

// DeleteStatusGen provides threadsafe delete of a status gen.
func DeleteStatusGen(
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
) {
	handlerState.lock.Lock()
	defer handlerState.lock.Unlock()
	delete(handlerState.clusterStatusGens, cr.UID)
}

// ClustersUsingApp returns the list of cluster names referencing the given app.
func ClustersUsingApp(
	app string,
	handlerState *handlerClusterState,
) []string {
	var clusters []string
	handlerState.lock.RLock()
	defer handlerState.lock.RUnlock()
	// This is a relationship that needs to be query-able given either ONLY
	// the app name (this function) or ONLY cluster name (RemoveClusterForApp).
	// Since the app CR deletion/update triggers for this function are very
	// infrequent, we'll implement this app-name check by just walking the
	// list of associations.
	for clusterKey, appName := range handlerState.clusterAppTypes {
		if appName == app {
			clusters = append(clusters, clusterKey)
		}
	}
	return clusters
}

// AddClusterForApp notes that an app type is in use by this cluster.
func AddClusterAppReference(
	cr *kdv1.KubeDirectorCluster,
	handlerState *handlerClusterState,
) {
	clusterKey := cr.Namespace + "/" + cr.Name
	handlerState.lock.Lock()
	defer handlerState.lock.Unlock()
	handlerState.clusterAppTypes[clusterKey] = cr.Spec.AppID
}

// RemoveClusterForApp notes that an app type is no longer in use by this
// cluster.
func RemoveClusterAppReference(
	namespace string,
	clusterName string,
	handlerState *handlerClusterState,
) {
	clusterKey := namespace + "/" + clusterName
	handlerState.lock.Lock()
	defer handlerState.lock.Unlock()
	delete(handlerState.clusterAppTypes, clusterKey)
}
