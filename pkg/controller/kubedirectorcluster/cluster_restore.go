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

package kubedirectorcluster

import (
	"context"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
)

// handleRestore runs the special reconciliation logic for a kdcluster that
// is in the process of being restored from backup.
func (r *ReconcileKubeDirectorCluster) handleRestore(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) error {

	// Memoize state of the incoming object.
	oldStatus := cr.Status.DeepCopy()

	// Make sure we have a Status object to work with.
	if cr.Status == nil {
		cr.Status = &kdv1.KubeDirectorClusterStatus{}
		cr.Status.Roles = make([]kdv1.RoleStatus, 0)
	}

	// Set a defer func to write new status if it changes. Simplified
	// version of the func from "normal" reconciliation, this one dealing
	// only with status updates.
	defer func() {
		// Bail out if nothing has changed. Note that if we are deleting we
		// don't care if status has changed.
		statusChanged := false
		if cr.DeletionTimestamp == nil {
			statusChanged = !equality.Semantic.DeepEqual(cr.Status, oldStatus)
		}
		if !statusChanged {
			return
		}
		// Write back the status. Don't exit this reconciler until we
		// succeed (will block other reconcilers for this resource).
		wait := time.Second
		maxWait := 4096 * time.Second
		for {
			// Do the status write.
			cr.Status.GenerationUID = uuid.New().String()
			ClusterStatusGens.WriteStatusGen(cr.UID, cr.Status.GenerationUID)
			updateErr := executor.UpdateClusterStatus(cr, false, nil)
			// If this succeeded, no need to do it again on next iteration.
			if updateErr == nil {
				return
			}
			// Update failed. If the cluster has been deleted, that's ok...
			// otherwise we'll try again.
			currentCluster, currentClusterErr := observer.GetCluster(
				cr.Namespace,
				cr.Name,
			)
			if currentClusterErr != nil {
				if errors.IsNotFound(currentClusterErr) {
					return
				}
			} else {
				if currentCluster.DeletionTimestamp != nil {
					return
				}
				// If we got a conflict error, update the CR with its current
				// form, restore our desired status, and try again immediately.
				if errors.IsConflict(updateErr) {
					currentCluster.Status = cr.Status
					*cr = *currentCluster
					continue
				}
			}
			if wait < maxWait {
				wait = wait * 2
			}
			shared.LogErrorf(
				reqLogger,
				updateErr,
				cr,
				shared.EventReasonCluster,
				"trying status update again in %v; failed",
				wait,
			)
			time.Sleep(wait)
		}
	}()

	// If we don't have a restoreProgress object in the status yet, set it
	// up.
	if cr.Status.RestoreProgress == nil {
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"being restored: setting initial restore-progress flags",
		)
		cr.Status.RestoreProgress = &kdv1.RestoreProgress{
			AwaitingApp:       true,
			AwaitingStatus:    true,
			AwaitingResources: true,
		}
	}

	// OK let's look for the resources we depend on. Note that it's possible
	// (tho hopefully unlikely) for a flag to flop from true to false and
	// back to true if a resource appears and then disappears. Only when all
	// are simultaneously false will we auto-switch back to normal reconciling.

	checkAppRestored(reqLogger, cr)

	checkStatusRestored(reqLogger, cr)

	if cr.Status.RestoreProgress.AwaitingStatus == false {
		checkResourcesRestored(reqLogger, cr)
	}

	// If all "waiting" flags are false we can try to resume reconcile.
	checkRestoreDone(reqLogger, cr)

	return nil
}

// checkAppRestored looks to see if the appropriate kdapp is present, and
// sets the restoreProgress.awaitingApp flag accordingly.
func checkAppRestored(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	_, err := catalog.FindApp(cr)
	catalogStr := "auto"
	if cr.Spec.AppCatalog != nil {
		catalogStr = *(cr.Spec.AppCatalog)
	}

	if err == nil {
		if cr.Status.RestoreProgress.AwaitingApp {
			cr.Status.RestoreProgress.AwaitingApp = false
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"being restored: kdapp %s (%s catalog) is present",
				cr.Spec.AppID,
				catalogStr,
			)
		}
	} else {
		cr.Status.RestoreProgress.AwaitingApp = true
		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"being restored: awaiting kdapp %s (%s catalog)",
			cr.Spec.AppID,
			catalogStr,
		)
	}
}

// checkStatusRestored looks to see if the appropriate kdstatusbackup is
// present, and if it is, copy its contents to this kdcluster's status.
// Set the restoreProgress.awaitingApp flag accordingly.
func checkStatusRestored(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	// If we've already restored the status, do nothing. This is one case where
	// it's not possible for the flag to flip from false to true.
	if cr.Status.RestoreProgress.AwaitingStatus == false {
		return
	}

	statusBackup, err := observer.GetStatusBackup(
		cr.Namespace,
		cr.Name,
	)

	if err == nil {
		if cr.Status.RestoreProgress.AwaitingStatus {
			cr.Status.RestoreProgress.AwaitingStatus = false
			shared.LogInfo(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"being restored: kdstatusbackup is present",
			)
			statusBackup.Spec.StatusBackup.RestoreProgress = cr.Status.RestoreProgress
			cr.Status = statusBackup.Spec.StatusBackup
		}
	} else {
		cr.Status.RestoreProgress.AwaitingStatus = true
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"being restored: awaiting kdstatusbackup",
		)
	}
}

// checkResourcesRestored looks to see if resources named in the status,
// which KD is responsible for directly creating, all exist. Set the
// restoreProgress.awaitingResources flag accordingly.
func checkResourcesRestored(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	// We need to check for the cluster (headless) service, the statefulsets,
	// and the per-member services associated with statefulset pods.
	// Everything else will be created by the statefulset controller.

	resourcesPresent := clusterServiceExists(reqLogger, cr)
	resourcesPresent = resourcesPresent && roleResourcesExist(reqLogger, cr)

	if resourcesPresent {
		if cr.Status.RestoreProgress.AwaitingResources {
			cr.Status.RestoreProgress.AwaitingResources = false
			shared.LogInfo(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"being restored: component resources are present",
			)
		}
	} else {
		cr.Status.RestoreProgress.AwaitingResources = true
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"being restored: awaiting component resources",
		)
	}
}

// clusterServiceExists looks to see if a headless service named in the
// status exists.
func clusterServiceExists(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) bool {

	clusterService := cr.Status.ClusterService
	if clusterService != "" {
		_, serviceErr := observer.GetService(
			cr.Namespace,
			clusterService,
		)
		if serviceErr != nil {
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"being restored: cluster service %s does not exist",
				clusterService,
			)
			return false
		}
	}
	return true
}

// roleResourcesExist looks to see if a statefulsets named in the
// status exist, along with the necessary per-member services.
func roleResourcesExist(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) bool {

	for _, roleStatus := range cr.Status.Roles {
		if roleStatus.StatefulSet != "" {
			_, statefulSetErr := observer.GetStatefulSet(
				cr.Namespace,
				roleStatus.StatefulSet,
			)
			if statefulSetErr != nil {
				shared.LogInfof(
					reqLogger,
					cr,
					shared.EventReasonCluster,
					"being restored: statefulset %s does not exist",
					roleStatus.StatefulSet,
				)
				return false
			}
		}
		for _, memberStatus := range roleStatus.Members {
			memberService := memberStatus.Service
			if memberService != "" && memberService != zeroPortsService {
				_, serviceErr := observer.GetService(
					cr.Namespace,
					memberService,
				)
				if serviceErr != nil {
					shared.LogInfof(
						reqLogger,
						cr,
						shared.EventReasonCluster,
						"being restored: member service %s does not exist",
						memberService,
					)
					return false
				}
			}
		}
	}
	return true
}

// checkRestoreDone will try to clear the being-restored label if all the
// "waiting" flags are false. If that fails it will populate the error field
// of the restore progress object.
func checkRestoreDone(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) {

	if cr.Status.RestoreProgress.AwaitingApp ||
		cr.Status.RestoreProgress.AwaitingStatus ||
		cr.Status.RestoreProgress.AwaitingResources {
		return
	}

	// Let's not deep-copy the whole CR; we just need to modify Labels.
	patchedCR := *cr
	patchedCR.Labels = make(map[string]string)
	for key, value := range cr.Labels {
		if key != shared.RestoringLabel {
			patchedCR.Labels[key] = value
		}
	}
	patchErr := shared.Patch(
		context.TODO(),
		cr,
		&patchedCR,
	)
	if patchErr == nil {
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonCluster,
			"resuming reconciliation",
		)
		return
	}
	// Don't want to use LogError here because it generates a stacktrace.
	// We don't really care about that, just about the message.
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonCluster,
		"failed to resume reconciliation: %s",
		patchErr.Error(),
	)
	cr.Status.RestoreProgress.Error = patchErr.Error()
}
