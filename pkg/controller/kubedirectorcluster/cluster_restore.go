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
		if cr.Status.RestoreProgress.AwaitingApp == true {
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
		if cr.Status.RestoreProgress.AwaitingStatus == true {
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
