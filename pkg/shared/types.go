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

package shared

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sClient struct {
	Clientset    kubernetes.Interface
	ClientConfig *rest.Config
}

const (
	DomainBase = ".svc.cluster.local"

	// KubeDirectorNamespaceEnvVar is the constant for env variable MY_NAMESPACE
	// which is the namespace of the kubedirector pod.
	KubeDirectorNamespaceEnvVar = "MY_NAMESPACE"
)
