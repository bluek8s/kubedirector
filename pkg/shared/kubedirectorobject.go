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

package shared

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// KubeDirectorFinalizerID is added to KubeDirector objects' finalizers
	// to prevent them from being deleted before we can clean up.
	KubeDirectorFinalizerID = "kubedirector.hpe.com/cleanup"
)

// KubeDirectorObject is an interface that most KubeDirector CRs implement.
// Currently it's used to add/remove the KubeDirector finalizer from
// KubeDirector resources.
type KubeDirectorObject interface {
	runtime.Object
	metav1.Object
}

// HasFinalizer checks whether the KubeDirector finalizer is among the CR's
// finalizers list.
func HasFinalizer(
	cr KubeDirectorObject,
) bool {

	finalizers := cr.GetFinalizers()
	for _, f := range finalizers {
		if f == KubeDirectorFinalizerID {
			return true
		}
	}
	return false
}

// RemoveFinalizer removes the KubeDirector finalizer from the CR's finalizers
// list (if it is in there).
func RemoveFinalizer(
	cr KubeDirectorObject,
) {

	finalizers := cr.GetFinalizers()
	for i, f := range finalizers {
		if f == KubeDirectorFinalizerID {
			cr.SetFinalizers(append(finalizers[:i], finalizers[i+1:]...))
			return
		}
	}
}

// EnsureFinalizer adds the KubeDirector finalizer into the CR's finalizers
// list (if it is not in there).
func EnsureFinalizer(
	cr KubeDirectorObject,
) {

	finalizers := cr.GetFinalizers()
	for _, f := range finalizers {
		if f == KubeDirectorFinalizerID {
			return
		}
	}
	cr.SetFinalizers(append(finalizers, KubeDirectorFinalizerID))
}
