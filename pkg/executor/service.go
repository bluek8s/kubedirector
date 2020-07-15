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

package executor

import (
	"context"

	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/catalog"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// CreateHeadlessService creates in k8s the "cluster service" used for
// intra-cluster network communication and for defining the virtual cluster's
// DNS subdomain. Cluster service name is an important part of DNS identity,
// so if we had already used a name previously and are re-creating the service,
// re-use that same name instead of generating a new one.
func CreateHeadlessService(
	cr *kdv1.KubeDirectorCluster,
) (*corev1.Service, error) {

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       cr.Namespace,
			OwnerReferences: ownerReferences(cr),
			Labels:          labelsForService(cr, nil),
			Annotations:     annotationsForCluster(cr),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector: map[string]string{
				HeadlessServiceLabel: cr.Name,
			},
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name: "port",
					Port: 8888, // not used
				},
			},
		},
	}
	if cr.Status.ClusterService == "" {
		service.ObjectMeta.GenerateName = headlessSvcNamePrefix
	} else {
		service.ObjectMeta.Name = cr.Status.ClusterService
	}
	err := shared.Create(context.TODO(), service)

	return service, err
}

// UpdateHeadlessService examines the current cluster service in k8s and may
// take steps to reconcile it to the desired spec.
func UpdateHeadlessService(
	cr *kdv1.KubeDirectorCluster,
	service *corev1.Service,
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
// LoadBalancer service. If there are no ports to configure for this service,
// no service object will be created and the function will return (nil, nil).
func CreatePodService(
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	podName string,
) (*corev1.Service, error) {

	serviceType := shared.ServiceType(*cr.Spec.ServiceType)

	portInfoList, portsErr := catalog.PortsForRole(cr, role.Name)
	if portsErr != nil {
		return nil, portsErr
	}
	if len(portInfoList) == 0 {
		return nil, nil
	}
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcNamePrefix + podName,
			Namespace:       cr.Namespace,
			OwnerReferences: ownerReferences(cr),
			Labels:          labelsForService(cr, role),
			Annotations:     annotationsForCluster(cr),
		},
		Spec: corev1.ServiceSpec{
			Selector:                 map[string]string{statefulSetPodLabel: podName},
			Type:                     serviceType,
			PublishNotReadyAddresses: true,
		},
	}
	for _, portInfo := range portInfoList {
		servicePort := corev1.ServicePort{
			Port: portInfo.Port,
			Name: createPortNameForService(portInfo),
		}
		service.Spec.Ports = append(service.Spec.Ports, servicePort)
	}
	createErr := shared.Create(context.TODO(), service)
	return service, createErr
}

// UpdatePodService examines a current per-member service in k8s and may take
// steps to reconcile it to the desired spec.
// TBD: Currently this function handles changes only for serviceType, and is
// only called if the service is known to already exist. If port-changing is
// supported in the future, either this function or its caller must take care
// of possibly transitioning to and from the "no ports" state which will
// involve deleting or creating the service object rather than just modifying.
func UpdatePodService(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *kdv1.Role,
	podName string,
	service *corev1.Service,
) error {

	reqServiceType := shared.ServiceType(*cr.Spec.ServiceType)

	// Compare cluster CR's service type against created service
	if reqServiceType == service.Spec.Type {
		return nil
	}

	shared.LogInfof(
		reqLogger,
		cr,
		shared.EventReasonMember,
		"modifying serviceType from %s to %s for service{%s}",
		service.Spec.Type,
		reqServiceType,
		service.Name,
	)

	if (service.Spec.Type == corev1.ServiceTypeNodePort ||
		service.Spec.Type == corev1.ServiceTypeLoadBalancer) &&
		(reqServiceType == corev1.ServiceTypeClusterIP ||
			reqServiceType == corev1.ServiceTypeLoadBalancer) {

		shared.LogInfof(
			reqLogger,
			cr,
			shared.EventReasonMember,
			"deleting service {%s}",
			service.Name,
		)

		deleteError := DeletePodService(
			reqLogger,
			cr.Namespace,
			service.Name,
		)

		if deleteError != nil {
			shared.LogInfof(
				reqLogger,
				cr,
				shared.EventReasonMember,
				"waiting for service of serviceType %s to be created by reconciler",
				reqServiceType,
			)

			return deleteError
		}
	} else {
		service.Spec.Type = reqServiceType
		return UpdateService(reqLogger, cr, service)
	}
	return nil
}

// DeletePodService deletes a per-member service from k8s.
func DeletePodService(
	reqLogger logr.Logger,
	namespace string,
	serviceName string,
) error {

	toDelete := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}

	return shared.Delete(context.TODO(), toDelete)
}

// UpdateService updates a service
func UpdateService(
	reqLogger logr.Logger,
	obj runtime.Object,
	service *corev1.Service,
) error {

	err := shared.Update(context.TODO(), service)
	if err == nil {
		return nil
	}
	// See https://github.com/bluek8s/kubedirector/issues/194
	// Migrate Client().Update() calls back to Patch() calls.
	if !errors.IsConflict(err) {
		shared.LogErrorf(
			reqLogger,
			err,
			obj,
			shared.EventReasonCluster,
			"failed to update service {%v}",
			service,
		)
		return err
	}

	// If there was a resourceVersion conflict then fetch a more
	// recent version of the object and attempt to update that.
	currentService := &corev1.Service{}
	err = shared.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: service.Namespace,
			Name:      service.Name,
		},
		currentService,
	)
	if err != nil {
		shared.LogErrorf(
			reqLogger,
			err,
			obj,
			shared.EventReasonMember,
			"failed to retrieve service{%s}",
			service.Name,
		)
		return err
	}

	currentService.Spec.Type = service.Spec.Type
	err = shared.Update(context.TODO(), currentService)
	if err != nil {
		shared.LogErrorf(
			reqLogger,
			err,
			obj,
			shared.EventReasonMember,
			"failed to update service{%s} with {%v}",
			service.Name,
			currentService,
		)
	}
	return err
}
