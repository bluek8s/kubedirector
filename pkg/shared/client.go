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
	"os"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// config is a config to talk to the apiserver.
	config *rest.Config

	// client is a k8s client to perform K8s CRUD operations. Will be set to
	// the split client (caching reads) after manager startup.
	client k8sClient.Client

	// directClient is always a non-caching client.
	directClient k8sClient.Client

	// clientSet is a REST API client that will be used for actions not
	//     supported through the operator SDK.
	clientSet kubernetes.Interface

	// eventRecorder will be used to publish events for a cr
	eventRecorder record.EventRecorder

	log = logf.Log.WithName("kubedirector")
)

// init ...
func init() {

	config = getConfigFromServiceAccount()
	client = getClient(config)
	directClient = getClient(config)
	clientSet = getClientSet(config)
	eventRecorder = getEventRecorder()
}

// getClientSet creates a k8s REST API client from the given config.
func getClientSet(
	config *rest.Config,
) kubernetes.Interface {

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "getClientSet")
		os.Exit(1)
	}
	return clientset
}

// getConfigFromServiceAccount generates a client config using the local
// service account credentials.
func getConfigFromServiceAccount() *rest.Config {

	config, err := k8sConfig.GetConfig()
	if err != nil {
		log.Error(err, "getConfigFromServiceAccount")
		os.Exit(1)
	}
	return config
}

// eventRecorder returns an EventRecorder type that can be
// used to post Events to different object's lifecycles.
func getEventRecorder() record.EventRecorder {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: ClientSet().CoreV1().Events(""),
		},
	)
	operatorName, _ := k8sutil.GetOperatorName()
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: operatorName},
	)
	return recorder
}

// getClient creates a k8s client from the given config.
func getClient(
	config *rest.Config,
) k8sClient.Client {

	client, err := k8sClient.New(config, k8sClient.Options{})
	if err != nil {
		log.Error(err, "getClient")
		os.Exit(1)
	}
	return client
}

// Config getter ...
func Config() *rest.Config {

	return config
}

// SetClient setter ...
func SetClient(
	c k8sClient.Client,
) {

	client = c
}

// ClientSet getter ...
func ClientSet() kubernetes.Interface {
	return clientSet
}

// isNotFoundInCache is a utility subroutine to check whether an error
// returned from the split client is of type ErrCacheNotStarted or is a
// not-found error.
func isNotFoundInCache(
	e error,
) bool {

	if errors.IsNotFound(e) {
		return true
	}
	_, ok := e.(*cache.ErrCacheNotStarted)
	return ok
}

// Create uses the split client. Should write back directly to K8s, but we'll
// use the split client in case it ever wants to use the knowledge that we
// are changing the object.
func Create(
	ctx context.Context,
	obj runtime.Object,
) error {

	return client.Create(ctx, obj)
}

// Get will first try a GET through the split client. If this returns 404,
// it will try the direct client.
// Cf. https://github.com/bluek8s/kubedirector/issues/267
func Get(
	ctx context.Context,
	key types.NamespacedName,
	obj runtime.Object,
) error {

	getErr := client.Get(ctx, key, obj)
	if (getErr == nil) || (!isNotFoundInCache(getErr)) {
		return getErr
	}
	return directClient.Get(ctx, key, obj)
}

// List uses the split client. Currently we don't have usecases where we
// would need to fall back to the direct client if the list has zero items,
// and it would be somewhat involved to examine the list object here to
// determine the zero-items case. We do however want to fall back to the
// direct client if isNotFoundInCache is true.
func List(
	ctx context.Context,
	list runtime.Object,
	opts ...k8sClient.ListOption,
) error {

	listErr := client.List(ctx, list, opts...)
	if (listErr == nil) || (!isNotFoundInCache(listErr)) {
		return listErr
	}
	return directClient.List(ctx, list, opts...)
}

// Update uses the split client. Should write back directly to K8s, but we'll
// use the split client in case it ever wants to use the knowledge that we
// are changing the object.
func Update(
	ctx context.Context,
	obj runtime.Object,
) error {

	return client.Update(ctx, obj)
}

// StatusUpdate uses the split client. Should write back directly to K8s, but
// we'll use the split client in case it ever wants to use the knowledge that
// we are changing the object.
func StatusUpdate(
	ctx context.Context,
	obj runtime.Object,
) error {

	return client.Status().Update(ctx, obj)
}

// Delete uses the split client. Should write back directly to K8s, but
// we'll use the split client in case it ever wants to use the knowledge that
// we are deleting the object.
func Delete(
	ctx context.Context,
	obj runtime.Object,
	opts ...k8sClient.DeleteOption,
) error {

	return client.Delete(ctx, obj, opts...)
}
