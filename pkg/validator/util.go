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
	"fmt"
	"strings"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
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
					TargetPort: intstr.FromInt(443),
				},
			},
		},
	}
	return sdk.Create(service)
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

	webhookHandler := v1beta1.Webhook{
		Name: webhookHandlerName,
		Rules: []v1beta1.RuleWithOperations{{
			Operations: []v1beta1.OperationType{
				v1beta1.Create,
				v1beta1.Update,
				v1beta1.Delete,
			},
			Rule: v1beta1.Rule{
				APIGroups:   []string{"kubedirector.bluedata.io"},
				APIVersions: []string{"v1alpha1"},
				Resources:   []string{"*"},
			},
		}},
		ClientConfig: v1beta1.WebhookClientConfig{
			Service: &v1beta1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      shared.StrPtr(validationPath),
			},
			CABundle: signingCert,
		},
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
		Webhooks: []v1beta1.Webhook{webhookHandler},
	}

	return sdk.Create(validator)
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
		"cluster.local",
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

	result := sdk.Create(secret)

	return secret, result
}
