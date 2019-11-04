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

import (
	"sync"
)

var (
	clustersUsingApp     map[string][]string
	clustersUsingAppLock sync.RWMutex
)

// init ...
func init() {

	clustersUsingApp = make(map[string][]string)
}

// ClustersUsingApp returns the list of clusters referencing the given app.
func ClustersUsingApp(
	appNamespace string,
	appID string,
) []string {

	key := appNamespace + "/" + appID
	clustersUsingAppLock.RLock()
	defer clustersUsingAppLock.RUnlock()
	if value, ok := clustersUsingApp[key]; ok {
		return value
	}
	return []string{}
}

// makeKey is a utility subroutine used by EnsureClusterAppReference and
// RemoveClusterAppReference.
func makeKey(
	clusterNamespace string,
	appCatalog string,
	appID string,
) string {

	if appCatalog == AppCatalogLocal {
		return clusterNamespace + "/" + appID
	}
	kdNamespace, _ := GetKubeDirectorNamespace()
	return kdNamespace + "/" + appID
}

// EnsureClusterAppReference notes that an app type is in use by this cluster.
// The cluster namespace and name are stored in a map indexed by the app CR's
// namespace+name. Note that we only expect this to be called once per cluster
// in the current design, but we will still protect against storing duplicates.
func EnsureClusterAppReference(
	clusterNamespace string,
	clusterName string,
	appCatalog string,
	appID string,
) {

	key := makeKey(clusterNamespace, appCatalog, appID)
	newElement := clusterNamespace + "/" + clusterName
	clustersUsingAppLock.Lock()
	defer clustersUsingAppLock.Unlock()
	if value, ok := clustersUsingApp[key]; ok {
		// Some clusters are already marked as using this app CR.
		if !StringInList(newElement, value) {
			// Not this one yet, so let's add it to the list.
			clustersUsingApp[key] = append(value, newElement)
		}
	} else {
		// First cluster using this app CR.
		clustersUsingApp[key] = []string{newElement}
	}
}

// RemoveClusterAppReference notes that an app type is no longer in use by
// this cluster. The cluster namespace+name is removed from the list of
// such references marked against the app's namespace+name. Note that we only
// expect this to be called when a reference to the cluster does exist; but
// if for some reason the reference does not exist this call is a NOP.
func RemoveClusterAppReference(
	clusterNamespace string,
	clusterName string,
	appCatalog string,
	appID string,
) {

	key := makeKey(clusterNamespace, appCatalog, appID)
	element := clusterNamespace + "/" + clusterName
	clustersUsingAppLock.Lock()
	defer clustersUsingAppLock.Unlock()
	if value, ok := clustersUsingApp[key]; ok {
		// Some clusters are marked as using this app CR.
		for i, e := range value {
			if e == element {
				// Found this cluster in the list.
				numElements := len(value)
				if numElements == 1 {
					// If we're the only one just delete the whole map bucket.
					delete(clustersUsingApp, key)
				} else {
					// Quick delete of our reference. Since we don't care about
					// order in this list, just copy the last element over
					// the element we want to delete, then truncate the list
					// to remove that last element.
					value[i] = value[numElements-1]
					clustersUsingApp[key] = value[:numElements-1]
				}
			}
		}
	}
}
