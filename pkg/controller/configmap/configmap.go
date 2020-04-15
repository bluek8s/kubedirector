// Copyright 2020 Hewlett Packard Enterprise Development LP

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmap

import (
	"context"
	"fmt"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

const (
	// configMapType is a label placed on every created statefulset, pod, and
	// service, with a value of the KubeDirectorCluster CR name.
	configMapType = shared.KdDomainBase + "/cmType"
)

var (
	// StatusGens is exported so that the validator can have access
	// to the ConfigMap StatusGens
	StatusGens = shared.NewStatusGens()
)

// syncConfigMap runs the reconciliation logic. It is invoked because of a
// change in or addition of configmap instance, currently there is no
// polling for this resource. If the configmap is not labeled
// with key ConfigMapType then no op
func (r *ReconcileConfigMap) syncConfigMap(
	reqLogger logr.Logger,
	configmap *corev1.ConfigMap,
) error {
	// Memoize state of the incoming object.
	oldMap, _ := observer.GetConfigMap(configmap.Namespace, configmap.Name)

	// Set a defer func to write new status and/or finalizers if they change.
	defer func() {

		if _, ok := oldMap.Labels[configMapType]; !ok {
			return
		}
		/* anonymous fun to check if some cluster
		   is using this config map as a connection */
		isClusterUsingConfigMap := func(cmName string, cluster kdv1.KubeDirectorCluster) bool {
			clusterModels := cluster.Spec.Connections.ConfigMaps
			for _, modelMapName := range clusterModels {
				if modelMapName == cmName {
					return true
				}
			}
			return false
		}
		allClusters := &kdv1.KubeDirectorClusterList{}
		shared.List(context.TODO(), allClusters)
		for _, kubecluster := range allClusters.Items {
			if isClusterUsingConfigMap(configmap.Name, kubecluster) {
				shared.LogInfof(
					reqLogger,
					configmap,
					shared.EventReasonConfigMap,
					"configmap {%s} is a connection to cluster {%s}, updating its configmeta",
					configmap.Name,
					kubecluster.Name,
				)
				updateMetaGenerator := &kubecluster
				updateMetaGenerator.Spec.ConnectionsGenToProcess = kubecluster.Spec.ConnectionsGenToProcess + 1
				//Notify cluster by incrementing configmetaGenerator
				wait := time.Second
				maxWait := 4096 * time.Second
				for {
					if shared.Update(context.TODO(), updateMetaGenerator) == nil {
						break
					}
					if wait > maxWait {
						shared.LogErrorf(
							reqLogger,
							fmt.Errorf("failed to update cluster"),
							configmap,
							shared.EventReasonConfigMap,
							"Unable to notify cluster {%s} of configmeta change",
							updateMetaGenerator.Name)
						break
					}
					time.Sleep(wait)
					wait = wait * 2
				}
			}
		}
	}()

	return nil
}
