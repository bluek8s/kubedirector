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
	corev1 "k8s.io/api/core/v1"
)

// ServiceType is a utility function that converts serviceType string to
// Kubedirector-Plus supported service types as a corev1.ServiceType
// returns corev1.ServiceTypeNodePort if crServicetype is not ClusterIP or LoadBalancer
func ServiceType(
	crServicetype string,
) corev1.ServiceType {

	switch crServicetype {
	case string(corev1.ServiceTypeClusterIP):
		return corev1.ServiceTypeClusterIP
	case string(corev1.ServiceTypeLoadBalancer):
		return corev1.ServiceTypeLoadBalancer
	}

	return corev1.ServiceTypeNodePort
}
