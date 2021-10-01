// Copyright 2021 Hewlett Packard Enterprise Development LP

package validator

import (
	"encoding/json"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// pvcPatchSpec is used to create the PATCH operation for setting the PVC
// owner references.
type pvcPatchSpec struct {
	Op    string                  `json:"op"`
	Path  string                  `json:"path"`
	Value []metav1.OwnerReference `json:"value"`
}

// mutateOwnerRefs checks PVC labels to see if it is for a kdcluster. If it
// is, it returns patches to set the ownerReferences to point to that CR.
func mutateOwnerRefs(
	pvc *corev1.PersistentVolumeClaim,
	patches []pvcPatchSpec,
) []pvcPatchSpec {

	if crName, ok := pvc.Labels[shared.ClusterLabel]; ok {
		// Label exists, let's get the cluster mentioned.
		cr, crErr := observer.GetCluster(pvc.Namespace, crName)
		if crErr == nil {
			// And set the owner refs.
			if len(cr.Labels) == 0 {
				patches = append(
					patches,
					pvcPatchSpec{
						Op:    "add",
						Path:  "/metadata/ownerReferences",
						Value: shared.OwnerReferences(cr),
					},
				)
			} else {
				patches = append(
					patches,
					pvcPatchSpec{
						Op:    "replace",
						Path:  "/metadata/ownerReferences",
						Value: shared.OwnerReferences(cr),
					},
				)
			}
		}
	}
	return patches
}

// admitPVC is the top-level PVC validation function, which invokes
// the top-specific validation subroutines and composes the admission
// response.
func admitPVC(
	ar *v1beta1.AdmissionReview,
) *v1beta1.AdmissionResponse {

	var patches []pvcPatchSpec
	var admitResponse = v1beta1.AdmissionResponse{
		Allowed: false,
	}

	// Deserialize the object.
	raw := ar.Request.Object.Raw
	pvc := corev1.PersistentVolumeClaim{}
	if jsonErr := json.Unmarshal(raw, &pvc); jsonErr != nil {
		admitResponse.Result = &metav1.Status{
			Message: "\n" + jsonErr.Error(),
		}
		return &admitResponse
	}

	// Get patches for kdcluster owner ref if necessary.
	patches = mutateOwnerRefs(&pvc, patches)

	// Apply patches.
	if len(patches) == 0 {
		admitResponse.Allowed = true
	} else {
		patchResult, patchErr := json.Marshal(patches)
		if patchErr == nil {
			admitResponse.Patch = patchResult
			patchType := v1beta1.PatchTypeJSONPatch
			admitResponse.PatchType = &patchType
			admitResponse.Allowed = true
		} else {
			admitResponse.Result = &metav1.Status{
				Message: "\n" + failedToPatchPVC,
			}
		}
	}

	return &admitResponse
}
