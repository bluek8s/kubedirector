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
	"context"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus propagates status changes back to k8s. Roles or members in
// the status that have been marked for deletion (by having certain fields
// set to emptystring) will be removed before the writeback.
func UpdateStatus(
	cr *kdv1.KubeDirectorCluster,
	client k8sclient.Client,
) error {

	prevCr := &kdv1.KubeDirectorCluster{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: cr.Namespace,
			Name: cr.Name,
		},
		prevCr,
	)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to retrieve cluster: %v",
			err,
		)
		return err
	}

	// Before writing back, remove any RoleStatus where StatefulSet is
	// emptystring, and remove any MemberStatus where Pod is emptystring.
	compact(&(cr.Status.Roles))

	// TBD: We should probably write to the status sub-resource. That's only
	// available in 1.11 (beta feature) and later though. So for now let's
	// just modify the status property of the CR.

	// TODO: Can we just use the cr that was passed in?
	prevCr.Status = cr.Status
	err = client.Status().Update(context.TODO(), prevCr)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to update status: %v",
			err,
		)
	}
	return err
}

// RemoveFinalizer removes the KubeDirector finalizer from the CR's finalizers
// list (if it is in there).
func RemoveFinalizer(
	cr *kdv1.KubeDirectorCluster,
	client k8sclient.Client,
) error {

	found := false
	for i, f := range cr.Finalizers {
		if f == finalizerID {
			cr.Finalizers = append(cr.Finalizers[:i], cr.Finalizers[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	// TODO: Can we just use the cr that was passed in?
	prevCr := &kdv1.KubeDirectorCluster{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: cr.Namespace,
			Name: cr.Name,
		},
		prevCr,
	)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to retrieve cluster: %v",
			err,
		)
		return err
	}

	prevCr.Finalizers = cr.Finalizers
	err = client.Update(context.TODO(), prevCr)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to remove finalizer: %v",
			err,
		)
	}
	return err
}

// EnsureFinalizer adds the KubeDirector finalizer into the CR's finalizers
// list (if it is not in there).
func EnsureFinalizer(
	cr *kdv1.KubeDirectorCluster,
	client k8sclient.Client,
) error {

	found := false
	for _, f := range cr.Finalizers {
		if f == finalizerID {
			found = true
			break
		}
	}
	if found {
		return nil
	}

	cr.Finalizers = append(cr.Finalizers, finalizerID)

	// TODO: Can we just use the cr that was passed in?
	prevCr := &kdv1.KubeDirectorCluster{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: cr.Namespace,
			Name: cr.Name,
		},
		prevCr,
	)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to retrieve cluster: %v",
			err,
		)
		return err
	}

	prevCr.Finalizers = cr.Finalizers
	err = client.Update(context.TODO(), prevCr)
	if err != nil {
		shared.LogErrorf(
			cr,
			shared.EventReasonCluster,
			"failed to add finalizer: %v",
			err,
		)
	}
	return err
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
