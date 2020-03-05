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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/bluek8s/kubedirector/pkg/observer"
	"github.com/bluek8s/kubedirector/pkg/shared"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add validation handlers for all CRs that we currently support
var validationHandlers = map[string]admitFunc{
	"KubeDirectorApp":     admitAppCR,
	"KubeDirectorCluster": admitClusterCR,
	"KubeDirectorConfig":  admitKDConfigCR,
}

// validation handles the http portion of a request prior to dispatching the
// resource-type-specific validation handler.
func validation(
	w http.ResponseWriter,
	r *http.Request,
) {

	var admissionResponse *v1beta1.AdmissionResponse

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(
			w,
			"invalid Content-Type, expect `application/json`",
			http.StatusUnsupportedMediaType,
		)
		return
	}

	ar := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		crKind := ar.Request.Kind.Kind
		// If there is a validation handler for this CR invoke it.
		if handler, ok := validationHandlers[crKind]; ok {
			admissionResponse = handler(&ar)
		} else {
			// No validation handler for this CR. Allow to go through.
			admissionResponse = &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("could not encode response: %v", err),
			http.StatusInternalServerError,
		)
	}
	if _, err := w.Write(respBytes); err != nil {
		http.Error(
			w,
			fmt.Sprintf("could not write response: %v", err),
			http.StatusInternalServerError,
		)
	}
}

// StartValidationServer starts the admission validation server. Prior to
// invoking this function, InitValidationServer function must be called to
// set up secret (for TLS certs) k8s resource. This function runs forever.
func StartValidationServer() error {

	// Fetch our namespace
	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return err
	}

	// Fetch certificate secret information
	certSecret, err := observer.GetSecret(kdNamespace, validatorSecret)
	if err != nil {
		return fmt.Errorf(
			"failed to read secret(%s) object %v",
			validatorSecret,
			err,
		)
	}

	// extract cert information from the secret object
	certBytes, ok := certSecret.Data[appCrt]
	if !ok {
		return fmt.Errorf(
			"%s value not found in %s secret",
			appCrt,
			validatorSecret,
		)
	}
	keyBytes, ok := certSecret.Data[appKey]
	if !ok {
		return fmt.Errorf(
			"%s value not found in %s secret",
			appKey,
			validatorSecret,
		)
	}

	signingCertBytes, ok := certSecret.Data[rootCrt]
	if !ok {
		return fmt.Errorf(
			"%s value not found in %s secret",
			rootCrt,
			validatorSecret,
		)
	}

	certPool := x509.NewCertPool()
	ok = certPool.AppendCertsFromPEM(signingCertBytes)
	if !ok {
		return fmt.Errorf("failed to parse root certificate")
	}

	sCert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr: ":" + strconv.Itoa(validationPort),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{sCert},
		},
	}

	http.HandleFunc(
		validationPath,
		func(w http.ResponseWriter, r *http.Request) {
			validation(w, r)
		},
	)

	http.HandleFunc(
		healthPath,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		},
	)

	err = server.ListenAndServeTLS("", "")

	return err
}

// InitValidationServer creates secret, service and admission validation k8s
// resources. All these resources are created in the same namespace where
// KubeDirector is running.
// XXX We could/should move to using the tls module now provided by the SDK.
// However, its interface requires storing the various certs/keys in two
// secrets and a configmap, while our current method uses one secret. Since
// there are now some existing deployments of KD, we would need a migration
// strategy.
func InitValidationServer(
	ownerReference metav1.OwnerReference,
) error {

	// Fetch our namespace
	kdNamespace, err := shared.GetKubeDirectorNamespace()
	if err != nil {
		return err
	}

	// Check to see if webhook secret is already present
	certSecret, err := observer.GetSecret(kdNamespace, validatorSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Secret not found, create certs and the secret object
			certSecret, err = createCertsSecret(
				ownerReference,
				validatorSecret,
				validatorServiceName,
				kdNamespace,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to create secret(%s) resource %v",
					validatorSecret,
					err,
				)
			}
		} else {
			// Unable to read secret object
			return fmt.Errorf(
				"unable to read secret object %s: %v",
				validatorSecret,
				err,
			)
		}
	}

	signingCertBytes, ok := certSecret.Data[rootCrt]
	if !ok {
		return fmt.Errorf(
			"%s value not found in %s secret",
			rootCrt,
			validatorSecret,
		)
	}

	serviceErr := createWebhookService(
		ownerReference,
		validatorServiceName,
		kdNamespace,
	)
	if serviceErr != nil {
		return fmt.Errorf(
			"failed to create Service{%s}: %v",
			validatorServiceName,
			serviceErr,
		)
	}

	validatorErr := createAdmissionService(
		ownerReference,
		validatorWebhook,
		kdNamespace,
		validatorServiceName,
		signingCertBytes,
	)
	if validatorErr != nil {
		return fmt.Errorf(
			"failed to create validator{%s}: %v",
			validatorWebhook,
			validatorErr,
		)
	}

	return nil
}
