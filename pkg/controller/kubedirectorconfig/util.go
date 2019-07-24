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

package kubedirectorconfig

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
)

// removeGlobalConfig removes the globalConfig from handler structure
func removeGlobalConfig(r *ReconcileKubeDirectorConfig) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.globalConfig = nil
}

// addGlobalConfig adds the globalConfig CR data to handler structure
func addGlobalConfig(
	r *ReconcileKubeDirectorConfig,
	cr *kdv1.KubeDirectorConfig,
) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.globalConfig = cr
}
