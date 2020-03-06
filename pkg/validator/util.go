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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createWebhookService creates our webhook Service resource if it does not
// exist.
func createWebhookService(
	ownerReference metav1.OwnerReference,
	serviceName string,
	namespace string,
) error {

	var createService = false
	_, err := observer.GetService(namespace, serviceName)
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

	if !createService {
		return nil
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
	return shared.Create(context.TODO(), service)
}

// createAdmissionService creates our MutatingWebhookConfiguration resource
// if it does not exist.
func createAdmissionService(
	ownerReference metav1.OwnerReference,
	validatorWebhook string,
	namespace string,
	serviceName string,
	signingCert []byte,
) error {

	var createValidator = false
	_, err := observer.GetValidatorWebhook(validatorWebhook)
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

	if !createValidator {
		return nil
	}

	// Webhook handler with a "fail" failure policy; these operations
	// will NOT be allowed even when the handler is down.
	// Use the v1beta1 version until our K8s version support floor is 1.16 or
	// better.
	failurePolicy := v1beta1.Fail
	// Also note that until we raise our K8s support floor to 1.15, we can't
	// use any properties in v1beta1.MutatingWebhook that were not also
	// present in the old v1beta1.Webhook.
	webhookHandler := v1beta1.MutatingWebhook{
		Name: webhookHandlerName,
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
		FailurePolicy: &failurePolicy,
	}

	validator := &v1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            validatorWebhook,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Webhooks: []v1beta1.MutatingWebhook{webhookHandler},
	}

	return shared.Create(context.TODO(), validator)
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
