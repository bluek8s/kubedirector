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

package executor

import (
	"encoding/json"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/types"
)

// statusPatchSpec is used to create PATCH operation input for modifying a
// virtual cluster CR's status.
type statusPatchSpec struct {
	Op    string             `json:"op"`
	Path  string             `json:"path"`
	Value kdv1.ClusterStatus `json:"value"`
}

// finalizersPatchSpec is used to create PATCH operation input for modifying a
// virtual cluster CR's finalizers list.
type finalizersPatchSpec struct {
	Op    string   `json:"op"`
	Path  string   `json:"path"`
	Value []string `json:"value"`
}

// UpdateStatus propagates status changes back to k8s. Roles or members in
// the status that have been marked for deletion (by having certain fields
// set to emptystring) will be removed before the writeback.
func UpdateStatus(
	cr *kdv1.KubeDirectorCluster,
) error {

	// Before writing back, remove any RoleStatus where StatefulSet is
	// emptystring, and remove any MemberStatus where Pod is emptystring.
	compact(&(cr.Status.Roles))

	// TBD: We should probably write to the status sub-resource. That's only
	// available in 1.11 (beta feature) and later though. So for now let's
	// just modify the status property of the CR.
	statusPatch := []statusPatchSpec{
		{
			Op:    "replace",
			Path:  "/status",
			Value: *(cr.Status),
		},
	}
	patchAttempt := func(p []statusPatchSpec) error {
		statusPatchBytes, err := json.Marshal(statusPatch)
		if err == nil {
			err = sdk.Patch(cr, types.JSONPatchType, statusPatchBytes)
		}
		return err
	}
	patchErr := patchAttempt(statusPatch)
	if patchErr != nil {
		// If replace doesn't work, try add. (First time update.)
		statusPatch[0].Op = "add"
		patchErr = patchAttempt(statusPatch)
	}
	if patchErr != nil {
		shared.LogErrorf(
			cr,
			true,
			shared.EventReasonCluster,
			"failed to update status: %v",
			patchErr,
		)
	}
	return patchErr
}

// RemoveFinalizer removes the KubeDirector finalizer from the CR's finalizers
// list (if it is in there).
func RemoveFinalizer(
	cr *kdv1.KubeDirectorCluster,
) error {

	found := false
	for i, f := range cr.Finalizers {
		if f == finalizerId {
			cr.Finalizers = append(cr.Finalizers[:i], cr.Finalizers[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	finalizersPatch := []finalizersPatchSpec{
		{
			Op:    "replace",
			Path:  "/metadata/finalizers",
			Value: cr.Finalizers,
		},
	}
	finalizersPatchBytes, patchErr := json.Marshal(finalizersPatch)
	if patchErr == nil {
		patchErr = sdk.Patch(cr, types.JSONPatchType, finalizersPatchBytes)
	}
	if patchErr != nil {
		shared.LogErrorf(
			cr,
			true,
			shared.EventReasonCluster,
			"failed to remove finalizer: %v",
			patchErr,
		)
	}
	return patchErr
}

// EnsureFinalizer adds the KubeDirector finalizer into the CR's finalizers
// list (if it is not in there).
func EnsureFinalizer(
	cr *kdv1.KubeDirectorCluster,
) error {

	found := false
	for _, f := range cr.Finalizers {
		if f == finalizerId {
			found = true
			break
		}
	}
	if found {
		return nil
	}
	cr.Finalizers = append(cr.Finalizers, finalizerId)

	var finalizersPatch []finalizersPatchSpec
	if len(cr.Finalizers) == 1 {
		finalizersPatch = []finalizersPatchSpec{
			{
				Op:    "add",
				Path:  "/metadata/finalizers",
				Value: cr.Finalizers,
			},
		}

	} else {
		finalizersPatch = []finalizersPatchSpec{
			{
				Op:    "replace",
				Path:  "/metadata/finalizers",
				Value: cr.Finalizers,
			},
		}
	}
	finalizersPatchBytes, patchErr := json.Marshal(finalizersPatch)
	if patchErr == nil {
		patchErr = sdk.Patch(cr, types.JSONPatchType, finalizersPatchBytes)
	}
	if patchErr != nil {
		shared.LogErrorf(
			cr,
			true,
			shared.EventReasonCluster,
			"failed to add finalizer: %v",
			patchErr,
		)
	}
	return patchErr
}

// compact edits the input slice of role statuses so that any elements that
// have an emptystring StatefulSet field are removed from the slice. Also
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
