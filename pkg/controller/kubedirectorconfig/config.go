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

package kubedirectorconfig

import (
	"context"
	"fmt"
	"reflect"
	"time"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
)

type configState string

const (
	configCreating configState = "creating"
	configReady    configState = "ready"
)

var (
	// StatusGens is exported so that the validator can have access
	// to the KubeDirectorConfig CR StatusGens
	StatusGens = shared.NewStatusGens()
)

// syncConfig runs the reconciliation logic. It is invoked because of a
// change in or addition of a KubeDirectorConfig instance, or a periodic
// polling to check on such a resource.
func (r *ReconcileKubeDirectorConfig) syncConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorConfig,
) error {

	// Memoize state of the incoming object.
	hadFinalizer := shared.HasFinalizer(cr)
	oldStatus := cr.Status.DeepCopy()

	// Make sure we have a Status object to work with.
	if cr.Status == nil {
		cr.Status = &kdv1.KubeDirectorConfigStatus{}
	}

	// Set a defer func to write new status and/or finalizers if they change.
	defer func() {
		nowHasFinalizer := shared.HasFinalizer(cr)
		// Bail out if nothing has changed. Note that if we are deleting we
		// don't care if status has changed.
		statusChanged := false
		if (cr.DeletionTimestamp == nil) || nowHasFinalizer {
			statusChanged = !reflect.DeepEqual(cr.Status, oldStatus)
		}
		finalizersChanged := (hadFinalizer != nowHasFinalizer)
		if !(statusChanged || finalizersChanged) {
			return
		}
		// Write back the status. Don't exit this reconciler until we
		// succeed (will block other reconcilers for this resource).
		wait := time.Second
		maxWait := 4096 * time.Second
		for {
			// If status has changed, write it back.
			var updateErr error
			if statusChanged {
				cr.Status.GenerationUID = uuid.New().String()
				StatusGens.WriteStatusGen(cr.UID, cr.Status.GenerationUID)
				updateErr = shared.StatusUpdate(context.TODO(), cr)
				// If this succeeded, no need to do it again on next iteration
				// if we're just cycling because of a failure to update the
				// finalizer.
				if updateErr == nil {
					statusChanged = false
				}
			}
			// If any necessary status update worked, let's also update
			// finalizers if necessary. To be safe, don't include the status
			// stanza in this write.
			if (updateErr == nil) && finalizersChanged {
				// See https://github.com/bluek8s/kubedirector/issues/194
				// Migrate Client().Update() calls back to Patch() calls.
				crWithoutStatus := cr.DeepCopy()
				crWithoutStatus.Status = nil
				updateErr = shared.Update(context.TODO(), crWithoutStatus)
			}
			// Bail out if we're done.
			if updateErr == nil {
				return
			}
			// Some necessary update failed. If the config has been deleted,
			// that's ok... otherwise we'll try again.
			currentConfig, currentConfigErr := observer.GetKDConfig(cr.Name)
			if currentConfigErr != nil {
				shared.LogErrorf(
					reqLogger,
					currentConfigErr,
					cr,
					shared.EventReasonConfig,
					"get current config failed",
				)
				if errors.IsNotFound(currentConfigErr) {
					return
				}
			} else {
				// If we got a conflict error, update the CR with its current
				// form, restore our desired status/finalizers, and try again
				// immediately.
				if errors.IsConflict(updateErr) {
					currentConfig.Status = cr.Status
					currentHasFinalizer := shared.HasFinalizer(currentConfig)
					if currentHasFinalizer {
						if !nowHasFinalizer {
							shared.RemoveFinalizer(currentConfig)
						}
					} else {
						if nowHasFinalizer {
							shared.EnsureFinalizer(currentConfig)
						}
					}
					*cr = *currentConfig
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
				shared.EventReasonConfig,
				"trying status update again in %v; failed",
				wait,
			)
			time.Sleep(wait)
		}
	}()

	// We use a finalizer to maintain config state consistency.
	doExit, finalizerErr := r.handleFinalizers(reqLogger, cr)
	if finalizerErr != nil {
		return finalizerErr
	}
	if doExit {
		return nil
	}

	// For a new KD config just update the status state/gen.
	shouldProcessCR, processErr := r.handleNewConfig(reqLogger, cr)
	if processErr != nil {
		return processErr
	}
	if !shouldProcessCR {
		return nil
	}

	if cr.Status.State == string(configCreating) {
		cr.Status.State = string(configReady)
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonConfig,
			"stable",
		)
	}

	shared.AddGlobalConfig(cr)
	return nil
}

// handleNewConfig looks in the cache for the last-known status generation
// UID for this CR. If there is one, make sure the UID is what we expect, and
// if so return true to keep processing the CR. If there is not any last-known
// UID, this is either a new CR or one that was created before this KD came up.
// In the former case, where the CR status itself has no generation UID: set
// the config state to creating (this will also trigger population of the
// generation UID) and return false to cause this handler to exit; we'll pick
// up further processing in the next handler. In the latter case, sync up our
// internal state with the visible state of the CR and return true to continue
// processing.
func (r *ReconcileKubeDirectorConfig) handleNewConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorConfig,
) (bool, error) {

	// Have we seen this config before?
	incoming := cr.Status.GenerationUID
	lastKnown, ok := StatusGens.ReadStatusGen(cr.UID)
	if ok {
		// Yep we've already done processing for this config previously.
		// Sanity check that the UID is what we expect... it REALLY should be,
		// but if there is a bug/race in the client code or unexpected behavior
		// of the K8s API consistency then it might not be.
		if lastKnown.UID == incoming {
			return true, nil
		}
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonNoEvent,
			"ignoring config CR with stale status UID; will retry",
		)
		mismatchErr := fmt.Errorf(
			"incoming UID %s != last known UID %s",
			incoming,
			lastKnown.UID,
		)
		return false, mismatchErr
	}
	if incoming == "" {
		// This is an actual newly-created config, so kick off the processing.
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonConfig,
			"new",
		)
		cr.Status.State = string(configCreating)
		return false, nil
	}
	// This config has been processed before but we're not aware of it yet.
	// Probably KD has been restarted. Make us aware of this config.
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonNoEvent,
		"unknown with incoming gen uid %s",
		incoming,
	)
	StatusGens.WriteValidatedStatusGen(cr.UID, incoming)
	return true, nil
}

// handleFinalizers will, if deletion has been requested, try to do any
// cleanup and then remove our finalizer from the in-memory CR. If deletion
// has NOT been requested then it will add our finalizer to the in-memory CR
// if it is absent.
func (r *ReconcileKubeDirectorConfig) handleFinalizers(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorConfig,
) (bool, error) {

	if cr.DeletionTimestamp != nil {
		// If a deletion has been requested, while ours (or other) finalizers
		// existed on the CR, go ahead and remove our finalizer.
		shared.RemoveFinalizer(cr)
		shared.LogInfo(
			reqLogger,
			cr,
			shared.EventReasonConfig,
			"greenlighting for deletion",
		)
		// Also clear the status gen from our cache.
		StatusGens.DeleteStatusGen(cr.UID)
		return true, nil
	}

	// If our finalizer doesn't exist on the CR, put it in there.
	shared.EnsureFinalizer(cr)

	return false, nil
}
