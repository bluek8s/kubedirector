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

package kubedirectorcluster

import (
	kdv1 "github.com/bluek8s/kubedirector/pkg/apis/kubedirector/v1beta1"
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// serviceShouldBeReconciled captures whether members in a given state should
// have their associated individual service processed.
var serviceShouldBeReconciled = map[memberState]bool{
	memberCreatePending: true,
	memberCreating:      true,
	memberReady:         true,
	memberConfigError:   true,
	memberDeletePending: false,
	memberDeleting:      false,
}

// syncClusterService is responsible for dealing with the per-member services.
// It and syncMemberServices are the only functions in this file that are
// invoked from another file (from the syncCluster function in cluster.go).
// Managing service changes may result in operations on k8s services. This
// function will also modify the status data structures to record the service
// name.
func syncClusterService(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) error {

	// If we already have the cluster service name stored,
	// look it up to see if it still exists.
	clusterService, queryErr := queryService(
		reqLogger,
		cr,
		cr.Status.ClusterService,
	)
	if queryErr != nil {
		return queryErr
	}
	if clusterService == nil {
		// We don't have an existing service, and we do need one.
		if cr.Status.ClusterService != "" {
			shared.LogInfo(
				reqLogger,
				cr,
				shared.EventReasonCluster,
				"re-creating missing cluster service",
			)
		}
		createErr := handleClusterServiceCreate(reqLogger, cr)
		if createErr != nil {
			return createErr
		}
	} else {
		// We have an existing service so just reconcile its config.
		handleClusterServiceConfig(reqLogger, cr, clusterService)
	}
	return nil
}

// syncMemberServices is responsible for dealing with the per-member services.
// It and syncClusterService are the only functions in this file that are
// invoked from another file (from syncCluster in cluster.go). Managing
// service changes may result in operations on k8s services. This function
// will also modify the status data structures to record the service names.
func syncMemberServices(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	roles []*roleInfo,
) error {

	for _, role := range roles {
		if role.roleStatus != nil {
			for i := 0; i < len(role.roleStatus.Members); i++ {
				serviceErr := handleMemberService(
					reqLogger,
					cr,
					role,
					&(role.roleStatus.Members[i]),
				)
				if serviceErr != nil {
					return serviceErr
				}
			}
		}
	}

	return nil
}

// handleClusterServiceCreate will create a cluster "headless" service and
// store its name in the cluster status. Failure to create this service will
// be a reconciler-stopping error.
func handleClusterServiceCreate(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
) error {

	clusterService, createErr := executor.CreateHeadlessService(cr)
	if createErr != nil {
		// Not much to do if we can't create it... we'll just keep trying
		// on every run through the reconciler.
		shared.LogError(
			reqLogger,
			createErr,
			cr,
			shared.EventReasonCluster,
			"failed to create cluster service",
		)
		cr.Status.ClusterService = ""
		return createErr
	}
	cr.Status.ClusterService = clusterService.Name
	return nil
}

// handleClusterServiceConfig checks an existing cluster "headless" service to
// see if any of its important properties need to be reconciled. Failure to
// reconcile will not be treated as a reconciler-stopping error; we'll just try
// again next time.
func handleClusterServiceConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	clusterService *corev1.Service,
) {

	updateErr := executor.UpdateHeadlessService(cr, clusterService)
	if updateErr != nil {
		shared.LogErrorf(
			reqLogger,
			updateErr,
			cr,
			shared.EventReasonCluster,
			"failed to update Service{%s}",
			cr.Status.ClusterService,
		)
	}
}

// handleMemberService makes sure that the per-member service exists if it
// should. (If it should not, we don't worry about it here... member syncing
// will clean it up.) If the service is created, it will store this service
// name in the member status. Failure to create a service as needed will be a
// reconciler-stopping error.
func handleMemberService(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
) error {

	if serviceShouldBeReconciled[memberState(member.State)] {
		if member.Service == zeroPortsService {
			// TBD: Currently nothing to do if no ports on the service. This
			// will change in the future if/when handleMemberServiceConfig
			// supports modification of an existing service's ports.
			return nil
		}
		memberService, queryErr := queryService(
			reqLogger,
			cr,
			member.Service,
		)
		if queryErr != nil {
			return queryErr
		}
		if memberService == nil {
			if member.Service != "" && member.Service != zeroPortsService {
				shared.LogInfof(
					reqLogger,
					cr,
					shared.EventReasonMember,
					"re-creating missing service for member{%s} in role{%s}",
					member.Pod,
					role.roleStatus.Name,
				)
			}
			// Need to create a service.
			createErr := handleMemberServiceCreate(
				reqLogger,
				cr,
				role,
				member,
			)
			if createErr != nil {
				return createErr
			}
		} else {
			// We have an existing service so just reconcile its config.
			handleMemberServiceConfig(
				reqLogger,
				cr,
				role,
				member,
				memberService,
			)
		}
	}
	return nil
}

// handleMemberServiceCreate will create a per-member service and store its
// name in the member status. Failure to create this service will be a
// reconciler-stopping error. In the special case of having no ports to configure,
// no service object will be created, and the service element of the member
// status will be assigned the special constant defined by zeroPortsService.
func handleMemberServiceCreate(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
) error {

	memberService, createErr := executor.CreatePodService(
		cr,
		role.roleSpec,
		member.Pod,
	)
	if createErr != nil {
		// Not much to do if we can't create it... we'll just keep trying
		// on every run through the reconciler.
		shared.LogErrorf(
			reqLogger,
			createErr,
			cr,
			shared.EventReasonMember,
			"failed to create member service for member{%s} in role{%s}",
			member.Pod,
			role.roleStatus.Name,
		)
		member.Service = ""
		return createErr
	}
	if memberService == nil {
		member.Service = zeroPortsService
	} else {
		member.Service = memberService.Name
	}
	return nil
}

// handleMemberServiceConfig checks an existing per-member service to see if
// any of its important properties need to be reconciled. Failure to reconcile
// will not be treated as a reconciler-stopping error; we'll just try again next
// time.
func handleMemberServiceConfig(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
	memberService *corev1.Service,
) {

	executor.UpdatePodService(
		reqLogger,
		cr,
		role.roleSpec,
		member.Pod,
		memberService,
	)
}

// queryService is a generalized lookup subroutine for finding either
// a cluster "headless" service or a per-member service. It will return
// nil for the Service pointer if the object does not exist.
func queryService(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorCluster,
	serviceName string,
) (*corev1.Service, error) {

	var service *corev1.Service
	if serviceName == "" || serviceName == zeroPortsService {
		service = nil
	} else {
		serviceFound, queryErr := observer.GetService(
			cr.Namespace,
			serviceName,
		)
		if queryErr == nil {
			service = serviceFound
		} else {
			if errors.IsNotFound(queryErr) {
				service = nil
			} else {
				shared.LogErrorf(
					reqLogger,
					queryErr,
					cr,
					shared.EventReasonNoEvent,
					"failed to query Service{%s}",
					serviceName,
				)
				return nil, queryErr
			}
		}
	}
	return service, nil
}
