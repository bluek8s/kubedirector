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
	"sync"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
)

var (
	globalConfig     *kdv1.KubeDirectorConfig
	globalConfigLock sync.RWMutex
)

// GetRequiredSecretPrefix returns a string that must prefix-match a
// secret name in order to allow that secret to be mounted by us. May be
// emptystring if no match required.
func GetRequiredSecretPrefix() string {

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	if globalConfig != nil && globalConfig.Spec.RequiredSecretPrefix != nil {
		return *globalConfig.Spec.RequiredSecretPrefix
	}
	return ""
}

// GetNativeSystemdSupport extracts the flag definition from the
// globalConfig CR data if present, otherwise returns false
func GetNativeSystemdSupport() bool {

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	if globalConfig != nil && globalConfig.Spec.NativeSystemdSupport != nil {
		return *globalConfig.Spec.NativeSystemdSupport
	}
	return false
}

// GetDefaultStorageClass extracts the default storage class from the
// globalConfig CR data if present, otherwise returns an empty string
func GetDefaultStorageClass() string {

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	if globalConfig != nil && globalConfig.Spec.StorageClass != nil {
		return *globalConfig.Spec.StorageClass
	}
	return ""
}

// GetDefaultServiceType extracts the default service type from the
// globalConfig CR data if present, otherwise returns the default
// value (NodePort).
func GetDefaultServiceType() string {

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	if globalConfig != nil && globalConfig.Spec.ServiceType != nil {
		return *globalConfig.Spec.ServiceType
	}
	return DefaultServiceType
}

// GetSvcClusterDomainBase extracts the default svc cluster domain
// from the globalConfig CR data if present, otherwise returns the default
// value (NodePort).
func GetSvcClusterDomainBase() string {

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()
	if globalConfig != nil && globalConfig.Spec.ClusterSvcDomainBase != nil {
		return *globalConfig.Spec.ClusterSvcDomainBase
	}
	return DefaultSvcDomainBase
}

// RemoveGlobalConfig removes the current globalConfig
func RemoveGlobalConfig() {

	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()
	globalConfig = nil
}

// AddGlobalConfig adds the globalConfig CR data
func AddGlobalConfig(config *kdv1.KubeDirectorConfig) {

	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()
	globalConfig = config
}
