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

// Package validator handles dynamic admission control checks for resources.
//
// InitValidationServer and StartValidationServer are called from the main
// function of KubeDirector. This starts an independent webserver process to
// field validation requests. Then, when resources in kubedirector.hpe.com
// (KubeDirectorCluster and KubeDirectorApp) are created/changed/deleted,
// the validation function will be invoked to check the proposed operation.
package validator
