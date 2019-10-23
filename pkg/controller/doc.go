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

// Package controller implements the reconciliation logic for custom resources.
//
// The reconciliation logic will make use of the other packages (observer,
// catalog, executor, etc.) to determine the current state of the relevant
// resources and adjust them to match the provided spec. The reconciliation
// code responsible for a particular type of custom resource is segregated
// into its own subdirectory.
package controller
