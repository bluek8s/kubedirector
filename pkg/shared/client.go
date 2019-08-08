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
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"os"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// config is a config to talk to the apiserver.
	config *rest.Config

	// client is a k8s client to perform K8s CRUD operations.
	client k8sClient.Client

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
		v1.EventSource{Component: operatorName},
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

// Client getter ...
func Client() k8sClient.Client {
	return client
}

// SetClient setter ...
func SetClient(c k8sClient.Client) {
	client = c
}

// ClientSet getter ...
func ClientSet() kubernetes.Interface {
	return clientSet
}
