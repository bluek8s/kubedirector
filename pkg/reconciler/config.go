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

package reconciler

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// syncConfig runs the reconciliation logic for config cr. It is invoked because of a
// change in or addition of a KubeDirectorConfig resource, or a periodic
// polling to check on such a resource. Currently all we do is set the config data
// in handler structure on add/change and on deletes set config data to be nil
func syncConfig(
	event sdk.Event,
	cr *kdv1.KubeDirectorConfig,
	handler *Handler,
) error {

	// Exit early if deleting the resource.
	if event.Deleted {
		removeGlobalConfig(handler)
		return nil
	}

	addGlobalConfig(handler, cr)

	return nil
}
