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

package secret

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
	// secretType is a label placed on desired comfig maps that
	// we want to watch and propogate inside containers
	secretType = shared.KdDomainBase + "/secretType"
)

// syncSecret runs the reconciliation logic. It is invoked because of a
// change in or addition of secret instance, currently there is no
// polling for this resource. If the secret is not labeled
// with key "kubedirector.hpe.com/secretType" then no op
func (r *ReconcileSecret) syncSecret(
	reqLogger logr.Logger,
	secret *corev1.Secret,
) error {

	// Memoize state of the incoming object.
	oldSecret, _ := observer.GetSecret(secret.Namespace, secret.Name)
	if _, ok := oldSecret.Labels[secretType]; !ok {
		return nil
	}
	/* anonymous fun to check if some cluster
	   is using this config map as a connection */
	isClusterUsingSecret := func(secretName string, cluster kdv1.KubeDirectorCluster) bool {
		clusterSecrets := cluster.Spec.Connections.Secrets
		for _, clusterSecret := range clusterSecrets {
			if clusterSecret == secretName {
				return true
			}
		}
		return false
	}
	allClusters := &kdv1.KubeDirectorClusterList{}
	shared.List(context.TODO(), allClusters)
	for _, kubecluster := range allClusters.Items {
		if isClusterUsingSecret(secret.Name, kubecluster) {
			shared.LogInfof(
				reqLogger,
				&kubecluster,
				shared.EventReasonSecret,
				"connected secret {%s} has changed",
				secret.Name,
			)
			shared.LogInfof(
				reqLogger,
				secret,
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
						secret,
						shared.EventReasonSecret,
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
