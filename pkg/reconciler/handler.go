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
	"context"
	"sync"

	"github.com/bluek8s/kubedirector/pkg/apis/kubedirector.bluedata.io/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/types"
)

// NewHandler creates the object that will handle reconciling all types of
// resources that we are watching.
func NewHandler() *Handler {
	return &Handler{
		ClusterState: handlerClusterState{
			lock:              sync.RWMutex{},
			clusterStatusGens: make(map[types.UID]StatusGen),
		},
	}
}

// Handle dispatches reconciliation to handler based on object type. Currently
// only KubeDirectorCluster is handled.
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.KubeDirectorCluster:
		return syncCluster(event, o, &(h.ClusterState))
	}
	return nil
}
