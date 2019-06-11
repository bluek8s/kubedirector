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
	defaultServiceType   = "LoadBalancer"
	defaultNativeSystemd = false

	appCrt  = "app.crt"
	appKey  = "app.pem"
	rootCrt = "ca.crt"

	invalidAppMessage   = "Invalid app(%s). This app resource ID has not been registered."
	invalidAppNamespace = "Invalid app_namespace(%s). Namespace (if present) must be either cluster's namespace or kubedirector's namespace."
	invalidCardinality  = "Invalid member count for role(%s). Specified member count:%d Role cardinality:%s"
	invalidRole         = "Invalid role(%s) in app(%s) specified. Valid roles: \"%s\""
	unconfiguredRole    = "Active role(%s) in app(%s) must have its configuration included in the roles array."

	modifiedProperty = "The %s property is read-only."
	modifiedRole     = "Role(%s) properties other than the members count cannot be modified while role members exist."

	invalidNodeRoleID     = "Invalid role_id(%s) in role_services array in config section. Valid roles: \"%s\""
	invalidSelectedRoleID = "Invalid element(%s) in selected_roles array in config section. Valid roles: \"%s\""
	invalidServiceID      = "Invalid service_id(%s) in role_services array in config section. Valid services: \"%s\""

	nonUniqueRoleID       = "Each id in the roles array must be unique."
	nonUniqueServiceID    = "Each id in the services array must be unique."
	nonUniqueSelectedRole = "Each element of selected_roles array in config section must be unique."
	nonUniqueServiceRole  = "Each role_id in role_services array in config section must be unique."

	noDefaultImage = "Role(%s) has no specified image, and no top-level default image is specified."

	noURLScheme = "The endpoint for service(%s) must include a url_scheme value because is_dashboard is true."

	failedToPatch = "Internal error: failed to populate default values for unspecified properties."

	invalidStorageClass = "Unable to fetch storageClass object with the provided name(%s)."

	invalidRoleStorageClass = "Unable to fetch storageClassName(%s) for role(%s)."
	noDefaultStorageClass   = "storageClassName is not specified for one or more roles, and no default storage class is available."
	badDefaultStorageClass  = "storageClassName is not specified for one or more roles, and default storage class (%s) is not available on the system."

	invalidResource = "Specified resource(\"%s\") value(\"%s\") for role(\"%s\") is invalid. Minimum value must be \"%s\"."
)
