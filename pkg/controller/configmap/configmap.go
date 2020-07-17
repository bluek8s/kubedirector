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
	"strconv"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	// ConfigMapType is a label placed on desired comfig maps that
	// we want to watch and propogate inside containers
	configMapType = shared.KdDomainBase + "/cmType"
)

// syncConfigMap runs the reconciliation logic. It is invoked because of a
// change in or addition of configmap instance, currently there is no
// polling for this resource. If the configmap is not labeled
// with key "kubedirector.hpe.com/cmType" then no op
func (r *ReconcileConfigMap) syncConfigMap(
	reqLogger logr.Logger,
	configmap *corev1.ConfigMap,
) error {

	// Memoize state of the incoming object.
	oldMap, _ := observer.GetConfigMap(configmap.Namespace, configmap.Name)
	if _, ok := oldMap.Labels[configMapType]; !ok {
		return nil
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
				&kubecluster,
				shared.EventReasonConfigMap,
				"connected configmap {%s} has changed",
				configmap.Name,
			)
			shared.LogInfof(
				reqLogger,
				configmap,
				shared.EventReasonCluster,
				"connected to cluster {%s}; updating it",
				kubecluster.Name,
			)

			//Notify cluster by incrementing configmetaGenerator
			wait := time.Second
			maxWait := 4096 * time.Second
			for {
				updateMetaGenerator := &kubecluster
				annotations := updateMetaGenerator.Annotations
				if annotations == nil {
					annotations = make(map[string]string)
					updateMetaGenerator.Annotations = annotations
				}
				if v, ok := annotations[shared.ConnectionsIncrementor]; ok {
					newV, _ := strconv.Atoi(v)
					annotations[shared.ConnectionsIncrementor] = strconv.Itoa(newV + 1)
				} else {
					annotations[shared.ConnectionsIncrementor] = "1"
				}
				updateMetaGenerator.Annotations = annotations
				if shared.Update(context.TODO(), updateMetaGenerator) == nil {
					break
				}
				// Since update failed, get a fresh copy of this cluster to work with and
				// try update
				updateMetaGenerator, fetchErr := observer.GetCluster(kubecluster.Namespace, kubecluster.Name)
				if fetchErr != nil {
					if errors.IsNotFound(fetchErr) {
						break
					}
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

	return nil
}
