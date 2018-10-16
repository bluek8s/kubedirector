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
	"github.com/bluek8s/kubedirector/pkg/executor"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// serviceShouldExist captures whether members in a given state should have
// an associated individual service.
var serviceShouldExist = map[memberState]bool{
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
	cr *kdv1.KubeDirectorCluster,
) error {

	// If we already have the cluster service name stored, look it up to see
	// if it still exists.
	clusterService, queryErr := queryService(
		cr,
		cr.Status.ClusterService,
	)
	if queryErr != nil {
		return queryErr
	}
	if clusterService == nil {
		// We don't have an existing service, and we do need one.
		if cr.Status.ClusterService != "" {
			shared.LogWarn(
				cr,
				true,
				shared.EventReasonCluster,
				"re-creating missing cluster service",
			)
		}
		createErr := handleClusterServiceCreate(cr)
		if createErr != nil {
			return createErr
		}
	} else {
		// We have an existing service so just reconcile its config.
		handleClusterServiceConfig(cr, clusterService)
	}
	return nil
}

// syncMemberServices is responsible for dealing with the per-member services.
// It and syncClusterService are the only functions in this file that are
// invoked from another file (from syncCluster in cluster.go). Managing
// service changes may result in operations on k8s services. This function
// will also modify the status data structures to record the service names.
func syncMemberServices(
	cr *kdv1.KubeDirectorCluster,
	roles []*roleInfo,
) error {

	for _, role := range roles {
		if role.roleStatus != nil {
			for i := 0; i < len(role.roleStatus.Members); i++ {
				serviceErr := handleMemberService(
					cr,
					role,
					&(role.roleStatus.Members[i]))
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
// be a handler-stopping error.
func handleClusterServiceCreate(
	cr *kdv1.KubeDirectorCluster,
) error {
	clusterService, createErr := executor.CreateHeadlessService(cr)
	if createErr != nil {
		// Not much to do if we can't create it... we'll just keep trying
		// on every run through the handler.
		shared.LogErrorf(
			cr,
			true,
			shared.EventReasonCluster,
			"failed to create cluster service: %v",
			createErr,
		)
		cr.Status.ClusterService = ""
		return createErr
	}
	cr.Status.ClusterService = clusterService.Name
	return nil
}

// handleClusterServiceConfig checks an existing cluster "headless" service to
// see if any of its important properties need to be reconciled. Failure to
// reconcile will not be treated as a handler-stopping error; we'll just try
// again next time.
func handleClusterServiceConfig(
	cr *kdv1.KubeDirectorCluster,
	clusterService *v1.Service,
) {

	updateErr := executor.UpdateHeadlessService(cr, clusterService)
	if updateErr != nil {
		shared.LogWarnf(
			cr,
			true,
			shared.EventReasonCluster,
			"failed to update Service{%s}: %v",
			cr.Status.ClusterService,
			updateErr,
		)
	}
}

// handleMemberService makes sure that the per-member service exists if it
// should. (If it should not, we don't worry about it here... member syncing
// will clean it up.) If the service is created, it will store this service
// name in the member status. Failure to create a service as needed will be a
// handler-stopping error.
func handleMemberService(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
) error {

	if serviceShouldExist[memberState(member.State)] {
		memberService, queryErr := queryService(
			cr,
			member.Service,
		)
		if queryErr != nil {
			return queryErr
		}
		if memberService == nil {
			if member.Service != "" {
				shared.LogWarnf(
					cr,
					true,
					shared.EventReasonCluster,
					"re-creating missing service for member{%s}",
					member.Pod,
				)
			}
			// Need to create a service.
			createErr := handleMemberServiceCreate(
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
// handler-stopping error.
func handleMemberServiceCreate(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
) error {
	memberService, createErr := executor.CreatePodService(
		cr,
		role.roleSpec,
		member.Pod)
	if createErr != nil {
		// Not much to do if we can't create it... we'll just keep trying
		// on every run through the handler.
		shared.LogErrorf(
			cr,
			true,
			shared.EventReasonMember,
			"failed to create member service for member{%s}: %v",
			member.Pod,
			createErr,
		)
		member.Service = ""
		return createErr
	}
	member.Service = memberService.Name
	return nil
}

// handleMemberServiceConfig checks an existing per-member service to see if
// any of its important properties need to be reconciled. Failure to reconcile
// will not be treated as a handler-stopping error; we'll just try again next
// time.
func handleMemberServiceConfig(
	cr *kdv1.KubeDirectorCluster,
	role *roleInfo,
	member *kdv1.MemberStatus,
	memberService *v1.Service,
) {

	updateErr := executor.UpdatePodService(
		cr,
		role.roleSpec,
		member.Pod,
		memberService,
	)
	if updateErr != nil {
		shared.LogWarnf(
			cr,
			true,
			shared.EventReasonMember,
			"failed to update Service{%s}: %v",
			member.Service,
			updateErr,
		)
	}
}

// queryService is a generalized lookup subroutine for finding either
// a cluster "headless" service or a per-member service. It will return
// nil for the Service pointer if the object does not exist.
func queryService(
	cr *kdv1.KubeDirectorCluster,
	serviceName string,
) (*v1.Service, error) {

	var service *v1.Service
	if serviceName == "" {
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
					cr,
					false,
					"",
					"failed to query Service{%s}: %v",
					serviceName,
					queryErr,
				)
				return nil, queryErr
			}
		}
	}
	return service, nil
}
