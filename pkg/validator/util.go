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

package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluek8s/kubedirector/pkg/cert"
	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/bluek8s/kubedirector/pkg/triple"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/api/admissionregistration/v1beta1"
	v1auth "k8s.io/api/authentication/v1"
	sar "k8s.io/api/authorization/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	corevalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsvalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// createWebhookService creates our webhook Service resource if it does not
// exist.
func createWebhookService(
	ownerReference metav1.OwnerReference,
	serviceName string,
	namespace string,
) error {

	var createService = false
	currentService, err := observer.GetService(namespace, serviceName)
	if err == nil {
		// service already present, no need to do anything
		createService = false
	} else {
		if errors.IsNotFound(err) {
			createService = true
		} else {
			return err
		}
	}

	// create service resource that refers to KubeDirector pod
	kdName, _ := k8sutil.GetOperatorName()
	serviceLabels := map[string]string{"name": kdName}
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			Name:            serviceName,
			Labels:          map[string]string{"webhook": kdName},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Spec: v1.ServiceSpec{
			Selector: serviceLabels,
			Ports: []v1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(validationPort),
				},
			},
		},
	}
	// For a service, delete-and-recreate has slightly less messy semantics
	// than update.
	if !createService {
		shared.Delete(context.TODO(), currentService)
	}
	return shared.Create(context.TODO(), service)
}

// createAdmissionService creates our MutatingWebhookConfiguration resource
// if it does not exist.
func createAdmissionService(
	validatorWebhook string,
	namespace string,
	serviceName string,
	signingCert []byte,
) error {

	var createValidator = false
	currentWebhook, err := observer.GetValidatorWebhook(validatorWebhook)
	if err == nil {
		// validator object already present, no need to do anything
		createValidator = false
	} else {
		if errors.IsNotFound(err) {
			createValidator = true
		} else {
			return err
		}
	}

	hardFailurePolicy := v1beta1.Fail
	softFailurePolicy := v1beta1.Ignore
	sideEffectsNone := v1beta1.SideEffectClassNone

	// Webhook handler with a "fail" failure policy; these operations
	// will NOT be allowed even when the handler is down.
	// Use the v1beta1 version until our K8s version support floor is 1.16 or
	// better.
	// Also note that until we raise our K8s support floor to 1.15, we can't
	// use any properties in v1beta1.MutatingWebhook that were not also
	// present in the old v1beta1.Webhook.
	hardWebhookHandler := v1beta1.MutatingWebhook{
		Name: "hard-" + webhookHandlerName,
		ClientConfig: v1beta1.WebhookClientConfig{
			Service: &v1beta1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      shared.StrPtr(validationPath),
			},
			CABundle: signingCert,
		},
		Rules: []v1beta1.RuleWithOperations{
			// For kubedirectorclusters and kubedirectorconfigs, we don't
			// actually do any delete validation, but if our whole operator is
			// down (most likely failure case) the object won't go away
			// because the reconciler won't remove its finalizer. And you
			// can't manually remove the finalizer without doing an update. So
			// let's head all of that off by just registering for Delete
			// (with Fail failure policy) for those resources too.
			{
				Operations: []v1beta1.OperationType{
					v1beta1.Create,
					v1beta1.Update,
					v1beta1.Delete,
				},
				Rule: v1beta1.Rule{
					APIGroups:   []string{"kubedirector.hpe.com"},
					APIVersions: []string{"v1beta1"},
					Resources: []string{
						"kubedirectorconfigs",
						"kubedirectorapps",
						"kubedirectorclusters",
					},
				},
			},
		},
		FailurePolicy: &hardFailurePolicy,
		SideEffects:   &sideEffectsNone,
	}

	// Webhook handler with an "ignore" failure policy; these operations
	// WILL be allowed even when the handler is down.
	// Use the v1beta1 version until our K8s version support floor is 1.16 or
	// better.
	// Also note that until we raise our K8s support floor to 1.15, we can't
	// use any properties in v1beta1.MutatingWebhook that were not also
	// present in the old v1beta1.Webhook.
	softWebhookHandler := v1beta1.MutatingWebhook{
		Name: "soft-" + webhookHandlerName,
		ClientConfig: v1beta1.WebhookClientConfig{
			Service: &v1beta1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      shared.StrPtr(validationPath),
			},
			CABundle: signingCert,
		},
		Rules: []v1beta1.RuleWithOperations{
			{
				Operations: []v1beta1.OperationType{
					v1beta1.Create,
				},
				Rule: v1beta1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"persistentvolumeclaims"},
				},
			},
		},
		FailurePolicy: &softFailurePolicy,
		SideEffects:   &sideEffectsNone,
	}

	validator := &v1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validatorWebhook,
		},
		Webhooks: []v1beta1.MutatingWebhook{hardWebhookHandler, softWebhookHandler},
	}

	if createValidator {
		return shared.Create(context.TODO(), validator)
	}
	// Overwrite the existing webhook. We'll do an update just so we don't
	// create a window where the webhook doesn't exist.
	validator.ResourceVersion = currentWebhook.ResourceVersion
	return shared.Update(context.TODO(), validator)
}

// createCertsSecret creates a self-signed certificate and stores it as a
// secret resource in Kubernetes.
func createCertsSecret(
	ownerReference metav1.OwnerReference,
	secretName string,
	serviceName string,
	namespace string,
) (*v1.Secret, error) {

	// Create a signing certificate
	caKeyPair, err := triple.NewCA(fmt.Sprintf("%s-ca", serviceName))
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	// Create app certs signed through the certificate created above
	apiServerKeyPair, err := triple.NewServerKeyPair(
		caKeyPair,
		strings.Join([]string{serviceName, namespace, "svc"}, "."),
		serviceName,
		namespace,
		[]string{},
		[]string{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create server key pair: %v", err)
	}

	// create an opaque secret resource with certificate(s) created above
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			appCrt:  cert.EncodeCertPEM(apiServerKeyPair.Cert),
			appKey:  cert.EncodePrivateKeyPEM(apiServerKeyPair.Key),
			rootCrt: cert.EncodeCertPEM(caKeyPair.Cert),
		},
	}

	result := shared.Create(context.TODO(), secret)

	return secret, result
}

// validateLabelsAndAnnotations is a common subroutine for validating a set of
// pod/service labels/annotations; used both for cluster and for config.
// Return an indicator of whether there were any errors, along with the
// updated errors list.
func validateLabelsAndAnnotations(
	path *field.Path,
	podLabels map[string]string,
	podAnnotations map[string]string,
	serviceLabels map[string]string,
	serviceAnnotations map[string]string,
	valErrors []string,
) ([]string, bool) {

	anyError := false
	labelErrors := appsvalidation.ValidateLabels(
		podLabels,
		path.Child("podLabels"),
	)
	annotationErrors := corevalidation.ValidateAnnotations(
		podAnnotations,
		path.Child("podAnnotations"),
	)
	serviceLabelErrors := appsvalidation.ValidateLabels(
		serviceLabels,
		path.Child("serviceLabels"),
	)
	serviceAnnotationErrors := corevalidation.ValidateAnnotations(
		serviceAnnotations,
		path.Child("serviceAnnotations"),
	)
	if (len(labelErrors) != 0) ||
		(len(annotationErrors) != 0) ||
		(len(serviceLabelErrors) != 0) ||
		(len(serviceAnnotationErrors) != 0) {
		anyError = true
		for _, labelErr := range labelErrors {
			valErrors = append(valErrors, labelErr.Error())
		}
		for _, annotationErr := range annotationErrors {
			valErrors = append(valErrors, annotationErr.Error())
		}
		for _, serviceLabelErr := range serviceLabelErrors {
			valErrors = append(valErrors, serviceLabelErr.Error())
		}
		for _, serviceAnnotationErr := range serviceAnnotationErrors {
			valErrors = append(valErrors, serviceAnnotationErr.Error())
		}
	}

	return valErrors, anyError
}

// createSubjectAccessReview is a utility function to validate if a user is allowed to access
// a resource in a namespace. It constructs SubjectAccessReviewSpec using the information
// provided by the caller and makes the SAR request to API Server. It returns an error string
// to the caller.
func createSubjectAccessReview(
	userInfo v1auth.UserInfo,
	resourceNamespace string,
	resourceName string,
	objectName string,
	verb string,
) (errStr string) {

	// Convert k8s.io/api/authentication/v1".ExtraValue -> k8s.io/api/authorization/v1".ExtraValue
	xtra := make(map[string]sar.ExtraValue)
	for k, v := range userInfo.Extra {
		xtra[k] = sar.ExtraValue(v)
	}
	sar := &sar.SubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1",
		},
		Spec: sar.SubjectAccessReviewSpec{
			ResourceAttributes: &sar.ResourceAttributes{
				Namespace: resourceNamespace,
				Verb:      verb,
				Resource:  resourceName,
				Name:      objectName,
			},
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			UID:    userInfo.UID,
			Extra:  xtra,
		},
	}
	err := shared.Create(context.TODO(), sar)
	if err != nil {
		errStr = err.Error()
	} else {
		if sar.Status.Denied {
			errStr = sar.Status.Reason
		}
	}

	return
}
