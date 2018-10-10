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
	"strconv"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateHeadlessService creates in k8s the "cluster service" used for
// intra-cluster network communication and for defining the virtual cluster's
// DNS subdomain. Cluster service name is an important part of DNS identity,
// so if we had already used a name previously and are re-creating the service,
// re-use that same name instead of generating a new one.
func CreateHeadlessService(
	cr *kdv1.KubeDirectorCluster,
) (*v1.Service, error) {

	name := headlessServiceName
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       cr.Namespace,
			OwnerReferences: ownerReferences(cr),
			Labels:          labelsForService(cr),
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				headlessServiceLabel: name,
			},
			Ports: []v1.ServicePort{
				{
					Name: "port",
					Port: 8888, // not used
				},
			},
		},
	}
	if cr.Status.ClusterService == "" {
		service.ObjectMeta.GenerateName = name + "-"
	} else {
		service.ObjectMeta.Name = cr.Status.ClusterService
	}
	err := sdk.Create(service)

	return service, err
}

// UpdateHeadlessService examines the current cluster service in k8s and may
// take steps to reconcile it to the desired spec.
func UpdateHeadlessService(
	cr *kdv1.KubeDirectorCluster,
	service *v1.Service,
) error {

	// TBD: We could compare the service against the expected service
	// (generated from the CR) and if there is a deviance in properties that
	// we need/expect to be under our control, correct them here. Not going
	// to tackle that at first.

	return nil
}

// CreatePodService creates in k8s a service that exposes the designated
// service endpoints of a virtual cluster member. Depending on the app type
// definition, this will be either a NodePort service (default) or a
// LoadBalancer service.
func CreatePodService(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	podName string,
) (*v1.Service, error) {

	var serviceType v1.ServiceType
	if cr.Spec.ServiceType == "" || cr.Spec.ServiceType == string(v1.ServiceTypeNodePort) {
		serviceType = v1.ServiceTypeNodePort
	} else {
		serviceType = v1.ServiceTypeLoadBalancer
	}
	ports, portsErr := catalog.PortsForRole(cr, role.Name)
	if portsErr != nil {
		return nil, portsErr
	}
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName(podName),
			Namespace:       cr.Namespace,
			OwnerReferences: ownerReferences(cr),
			Labels:          labelsForService(cr),
		},
		Spec: v1.ServiceSpec{
			Selector: labelsForPod(cr, role, podName),
			Type:     serviceType,
		},
	}
	for _, port := range ports {
		servicePort := v1.ServicePort{
			Port: port,
			Name: servicePortName(strconv.Itoa(int(port))),
		}
		service.Spec.Ports = append(service.Spec.Ports, servicePort)
	}
	createErr := sdk.Create(service)
	return service, createErr
}

// UpdatePodService examines a current per-member service in k8s and may take
// steps to reconcile it to the desired spec.
func UpdatePodService(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	podName string,
	service *v1.Service,
) error {

	// TBD: We could compare the service against the expected service
	// (generated from the CR) and if there is a deviance in properties that
	// we need/expect to be under our control, correct them here. Not going
	// to tackle that at first.

	return nil
}

// DeletePodService deletes a per-member service from k8s.
func DeletePodService(
	namespace string,
	serviceName string,
) error {

	toDelete := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}

	return sdk.Delete(toDelete)
}

// serviceName is a utility function for generating the name of a service
// from a given base string.
func serviceName(
	baseName string,
) string {

	return "svc-" + baseName
}

// servicePortName is a utility function for generating the name of a service
// port from a given base string.
func servicePortName(
	baseName string,
) string {

	return "port-" + baseName
}
