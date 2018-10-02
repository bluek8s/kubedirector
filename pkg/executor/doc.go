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

// Package executor manages objects within k8s that implement virtual clusters.
//
// For the most part, the exported functions in this package will create,
// update, or delete individual native k8s objects that make up parts of the
// virtual cluster. The exceptions are in guest.go, where the exported
// functions handle operations within a cluster member's OS.
package executor
