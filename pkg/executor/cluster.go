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

package executor

import (
	"context"
	"fmt"
	"strings"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/exec"
)

// UpdateClusterStatus propagates status changes back to k8s. Roles or members
// in the status that have been marked for deletion (by having certain fields
// set to emptystring) will be removed before the writeback.
func UpdateClusterStatus(
	cr *kdv1.KubeDirectorCluster,
	statusBackupShouldExist bool,
	statusBackup *kdv1.KubeDirectorStatusBackup,
) error {

	// Before writing back, remove any RoleStatus where StatefulSet is
	// emptystring, and remove any MemberStatus where Pod is emptystring.
	compact(&(cr.Status.Roles))

	// First sync the backup status CR. This includes deleting it if it is
	// not supposed to exist.
	if statusBackupShouldExist {
		if statusBackup != nil {
			// Overwrite
			statusBackup.Spec.StatusBackup = cr.Status
			updateErr := shared.Update(context.TODO(), statusBackup)
			if updateErr != nil {
				return updateErr
			}
		} else {
			// Create
			statusBackup := &kdv1.KubeDirectorStatusBackup{
				TypeMeta: metav1.TypeMeta{
					Kind:       "KubeDirectorStatusBackup",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            cr.Name,
					Namespace:       cr.Namespace,
					OwnerReferences: shared.OwnerReferences(cr),
				},
				Spec: kdv1.KubeDirectorStatusBackupSpec{
					StatusBackup: cr.Status,
				},
			}
			createErr := shared.Create(context.TODO(), statusBackup)
			if createErr != nil {
				return createErr
			}
		}
	} else {
		if statusBackup != nil {
			// Best-effort delete.
			shared.Delete(context.TODO(), statusBackup)
		}
	}

	// OK finally let's update the status subresource.
	return shared.StatusUpdate(context.TODO(), cr)
}

// BackupAnnotationNeedsReconcile checks that the annotation exists and
// has the correct value in the in-memory CR.
func BackupAnnotationNeedsReconcile(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	statusBackupShouldExist bool,
) bool {

	desiredValue := "true"
	if !statusBackupShouldExist {
		desiredValue = "false"
	}
	if cr.Annotations != nil {
		if annValue, ok := cr.Annotations[shared.StatusBackupAnnotation]; ok {
			if annValue == desiredValue {
				return false
			}
		}
	}
	return true
}

// SetBackupAnnotation sets the annotation to the desired value. If the
// CR in K8s is successfully updated, the annotations of the in-memory CR
// (passed to this function) will also be updated to match.
func SetBackupAnnotation(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	statusBackupShouldExist bool,
) error {

	desiredValue := "true"
	if !statusBackupShouldExist {
		desiredValue = "false"
	}
	patchedCR := *cr
	patchedCR.Annotations = make(map[string]string)
	for key, value := range cr.Annotations {
		patchedCR.Annotations[key] = value
	}
	patchedCR.Annotations[shared.StatusBackupAnnotation] = desiredValue
	patchErr := shared.Patch(
		context.TODO(),
		cr,
		&patchedCR,
	)
	if patchErr == nil {
		cr.Annotations = patchedCR.Annotations
	}
	return patchErr
}

// UpdateClusterStatusBackupOwner handles reconciliation only of the owner ref.
func UpdateClusterStatusBackupOwner(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	statusBackup *kdv1.KubeDirectorStatusBackup,
) error {

	if statusBackup == nil {
		return nil
	}
	if shared.OwnerReferencesPresent(cr, statusBackup.OwnerReferences) {
		return nil
	}
	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonNoEvent,
		"repairing owner ref on statusbackup{%s}",
		statusBackup.Name,
	)
	// We're just going to nuke any existing owner refs. (A bit more
	// discussion of this in UpdateStatefulSetNonReplicas comments.)
	patchedRes := *statusBackup
	patchedRes.OwnerReferences = shared.OwnerReferences(cr)
	return shared.Patch(
		context.TODO(),
		statusBackup,
		&patchedRes,
	)
}

// compact edits the input slice of role statuses so that any elements that
// have an empty string StatefulSet field are removed from the slice. Also
// compactMembers is invoked on the Pod field of the non-removed elements.
func compact(
	r *[]kdv1.RoleStatus,
) {

	numRoles := len(*r)
	numRemovedRoles := 0
	for i := 0; i < numRoles; i++ {
		// Is this role status marked for removal?
		if (*r)[i].StatefulSet == "" {
			// Is there a subsequent role we can compact into this slot?
			didCompact := false
			for j := i + 1; j < numRoles; j++ {
				if (*r)[j].StatefulSet != "" {
					(*r)[i] = (*r)[j]
					(*r)[j].StatefulSet = ""
					didCompact = true
					break
				}
			}
			if !didCompact {
				// Didn't find any subsequent non-empty slots. Time to stop
				// looping.
				numRemovedRoles = (numRoles - i)
				break
			}
		}
		// Compact the member status list.
		compactMembers(&((*r)[i].Members))
	}
	*r = (*r)[:numRoles-numRemovedRoles]
}

// compactMembers edits the input slice of member statuses so that any
// elements that have an emptystring Pod field are removed from the slice.
func compactMembers(
	m *[]kdv1.MemberStatus,
) {

	numMembers := len(*m)
	numRemovedMembers := 0
	for i := 0; i < numMembers; i++ {
		// Is this members status marked for removal?
		if (*m)[i].Pod == "" {
			// Is there a subsequent member we can compact into this slot?
			didCompact := false
			for j := i + 1; j < numMembers; j++ {
				if (*m)[j].Pod != "" {
					(*m)[i] = (*m)[j]
					(*m)[j].Pod = ""
					didCompact = true
					break
				}
			}
			if !didCompact {
				// Didn't find any subsequent non-empty slots. Time to stop
				// looping.
				numRemovedMembers = (numMembers - i)
				break
			}
		}
	}
	*m = (*m)[:numMembers-numRemovedMembers]
}

// UpdateStorageInitProgress parses rsync output (if available) and sets the
// memberStatus.StateDetail.StorageInitProgress field
func UpdateStorageInitProgress(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	memberStatus *kdv1.MemberStatus,
	initContainerStatus corev1.ContainerStatus,
) {
	// If the rsync progress file does not exist, this indicates that we're
	// not using rsync. If the file check itself fails, the init container is
	// probably not ready yet (e.g. image pull is ongoing).
	var rsyncStatusStrB strings.Builder
	progressBarFile := fmt.Sprintf("/mnt%s", kubedirectorInitProgressBar)
	fileExists, fileExistsErr := IsFileExists(
		reqLogger,
		cr,
		cr.Namespace,
		(*memberStatus).Pod,
		initContainerStatus.ContainerID,
		initContainerName,
		progressBarFile,
	)
	if fileExistsErr != nil {
		message := initContainerNotReady
		memberStatus.StateDetail.StorageInitProgress = &message
		return
	} else if !fileExists {
		message := initProgressNotAvailable
		memberStatus.StateDetail.StorageInitProgress = &message
		return
	}
	// Progress file exists. We want to get the last whole line (as demarcated
	// by carriage returns). This is complicated by the fact that rsync is
	// still writing the file, and we don't really want to rely on any
	// knowledge of the line format, so we can't say if the last line really
	// is a complete line. So we'll use the second-to-last line. Given the
	// rate at which these lines update (which is much faster than our
	// reconciliation period anyway), that's fine.
	// Also, that progress log file is potentially large. So we don't want to
	// make a copy of it for processing, and we don't want to read it all into
	// memory here. Instead we'll "tail" a chunk of it back to us here (that
	// is more than big enough to contain two complete lines) and then look
	// for the second-to-last complete line in that output.
	ioStreams := &Streams{
		Out: &rsyncStatusStrB,
	}
	tailExecErr := ExecCommand(
		reqLogger,
		cr,
		cr.Namespace,
		(*memberStatus).Pod,
		initContainerStatus.ContainerID,
		initContainerName,
		[]string{"tail", "-c", "1024", progressBarFile},
		ioStreams,
	)
	if tailExecErr != nil {
		_, iscoe := tailExecErr.(exec.CodeExitError)
		if iscoe {
			// Non-zero exit status for the command. Could be a problem with
			// "tail" in the container? ... who knows. Not going to dig into
			// it deeply here.
			shared.LogError(
				reqLogger,
				tailExecErr,
				cr,
				shared.EventReasonCluster,
				"error status on attempt to pull last line(s) from rsync progress file",
			)
			message := initProgressNotAvailable
			memberStatus.StateDetail.StorageInitProgress = &message
		}
		// At this point we have just a failure to run the command at all.
		// Since we already successfully did the file check above, the most
		// likely reason for this is that the init is done. Don't set a
		// progress message.
		return
	}
	// OK let's pull some set of lines out of what we received.
	lines := strings.Split(rsyncStatusStrB.String(), "\r")
	numLines := len(lines)
	// If there's not two lines, we can't start reporting yet.
	if numLines < 2 {
		message := initProgressPending
		memberStatus.StateDetail.StorageInitProgress = &message
		return
	}
	// Otherwise, we have progress to report.
	lastFullLine := strings.TrimSpace(lines[numLines-2])
	memberStatus.StateDetail.StorageInitProgress = &lastFullLine
}
