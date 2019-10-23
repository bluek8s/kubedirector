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
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// KubeDirectorFinalizerID is added to KubeDirector objects' finalizers
	// to prevent them from being deleted before we can clean up.
	KubeDirectorFinalizerID = "kubedirector.bluedata.io/cleanup"
)

// KubeDirectorObject is an interface that most KubeDirector CRs implement.
// Currently it's used to add/remove the KubeDirector finalizer from
// KubeDirector resources.
type KubeDirectorObject interface {
	GetFinalizers() []string
	SetFinalizers(finalizers []string)
	runtime.Object
}

// RemoveFinalizer removes the KubeDirector finalizer from the CR's finalizers
// list (if it is in there).
func RemoveFinalizer(
	cr KubeDirectorObject,
) error {

	found := false
	finalizers := cr.GetFinalizers()
	for i, f := range finalizers {
		if f == KubeDirectorFinalizerID {
			cr.SetFinalizers(append(finalizers[:i], finalizers[i+1:]...))
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	// See https://github.com/bluek8s/kubedirector/issues/194
	// Migrate Client().Update() calls back to Patch() calls.
	return Client().Update(context.TODO(), cr)
}

// EnsureFinalizer adds the KubeDirector finalizer into the CR's finalizers
// list (if it is not in there).
func EnsureFinalizer(
	cr KubeDirectorObject,
) error {

	found := false
	finalizers := cr.GetFinalizers()
	for _, f := range finalizers {
		if f == KubeDirectorFinalizerID {
			found = true
			break
		}
	}
	if found {
		return nil
	}

	cr.SetFinalizers(append(finalizers, KubeDirectorFinalizerID))

	// See https://github.com/bluek8s/kubedirector/issues/194
	// Migrate Client().Update() calls back to Patch() calls.
	return Client().Update(context.TODO(), cr)
}
