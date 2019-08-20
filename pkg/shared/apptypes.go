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

import (
	"strings"
	"sync"
)

var (
	appTypes     map[string]string
	appTypesLock sync.RWMutex
)

// init ...
func init() {

	appTypes = make(map[string]string)
}

// ClustersUsingApp returns the list of cluster names referencing the given app.
func ClustersUsingApp(
	appNamespace string,
	appID string,
) []string {

	var clusters []string
	appTypesLock.RLock()
	defer appTypesLock.RUnlock()
	// This is a relationship that needs to be query-able given either ONLY
	// the app name (in this function) or ONLY the cluster name (in
	// removeClusterAppReference). Since the app CR deletion/update triggers
	// for this function are very infrequent, we'll implement this app-name
	// check by just walking the list of associations. It's also nice to go
	// ahead and gather all the offending cluster CR names to report back to
	// the client.
	for clusterKey, appName := range appTypes {
		// Extract app namespace from clusterKey
		keyComponents := strings.Split(clusterKey, "/")
		clusterAppNamespace := keyComponents[0]
		clusterNamespace := keyComponents[1]
		clusterName := keyComponents[2]
		if appName == appID && clusterAppNamespace == appNamespace {
			clusters = append(clusters, clusterNamespace+"/"+clusterName)
		}
	}
	return clusters
}

// EnsureClusterAppReference notes that an app type is in use by this cluster.
func EnsureClusterAppReference(
	appNamespace string,
	clusterNamespace string,
	clusterName string,
	appID string,
) {

	clusterKey := appNamespace + "/" + clusterNamespace + "/" + clusterName
	appTypesLock.Lock()
	defer appTypesLock.Unlock()
	appTypes[clusterKey] = appID
}

// RemoveClusterAppReference notes that an app type is no longer in use by
// this cluster.
func RemoveClusterAppReference(
	appNamespace string,
	clusterNamespace string,
	clusterName string,
) {

	clusterKey := appNamespace + "/" + clusterNamespace + "/" + clusterName
	appTypesLock.Lock()
	defer appTypesLock.Unlock()
	delete(appTypes, clusterKey)
}
