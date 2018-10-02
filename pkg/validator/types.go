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

package validator

import (
	"github.com/bluek8s/kubedirector/pkg/reconciler"
	"k8s.io/api/admission/v1beta1"
)

// admitFunc is used as the type for all the callback validators
type admitFunc func(*v1beta1.AdmissionReview, *reconciler.Handler) *v1beta1.AdmissionResponse
type checkFunc func() error

const (
	validatorServiceName = "kubedirector-validator"
	validatorWebhook     = "kubedirector-webhook"
	validatorSecret      = "kubedirector-validator-secret"
	webhookHandlerName   = "validate-cr.kubedirector.bluedata.io"
	validationPath       = "/validate"

	appCrt  = "app.crt"
	appKey  = "app.pem"
	rootCrt = "ca.crt"

	invalidAppMessage  = "Invalid app(%s). This app resource ID has not been registered."
	invalidCardinality = "Invalid member count for role(%s). Specified member count:%d Role cardinality:%s"
	invalidRole        = "Invalid role(%s) in app(%s) specified. Valid roles: \"%s\""
	unconfiguredRole   = "Active role(%s) in app(%s) must have its configuration included in the roles list."

	invalidNodeRoleID     = "Invalid role_id(%s) in role_services section in config section. Valid roles: \"%s\""
	invalidSelectedRoleID = "Invalid role_id(%s) in selected_roles array in config section. Valid roles: \"%s\""
	invalidServiceID      = "Invalid service_id(%s) in role_services section in config section. Valid services: \"%s\""

	defaultMemberErr = "One or more role member counts are unspecified, but KubeDirector failed to apply the apptype-defined defaults."
)
