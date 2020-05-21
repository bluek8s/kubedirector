// Copyright 2020 Hewlett Packard Enterprise Development LP

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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/munnerz/goautoneg"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

type mediaType struct {
	Type, SubType string
}

var scheme = runtime.NewScheme()
var serializers = map[mediaType]runtime.Serializer{
	{"application", "json"}: json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false),
	{"application", "yaml"}: json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme),
}

func getInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func getOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}

	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}

	return nil
}

func convert(
	convertRequest *v1beta1.ConversionRequest,
) *v1beta1.ConversionResponse {

	var convertedObjects []runtime.RawExtension
	for _, obj := range convertRequest.Objects {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(obj.Raw); err != nil {
			return &v1beta1.ConversionResponse{
				Result: metav1.Status{
					Message: fmt.Sprintf("failed to unmarshall object (%v) with error: %v", string(obj.Raw), err),
					Status:  metav1.StatusFailure,
				},
			}
		}

		fromVersion := cr.GetAPIVersion()
		toVersion := convertRequest.DesiredAPIVersion
		convertedObject := cr.DeepCopy()

		if fromVersion == toVersion {
			return &v1beta1.ConversionResponse{
				Result: metav1.Status{
					Message: fmt.Sprintf("conversion from a version to itself should not call the webhook: %s", toVersion),
					Status:  metav1.StatusFailure,
				},
			}
		}

		switch fromVersion {
		case "kubedirector.hpe.com/v1beta1":
			switch toVersion {
			case "kubedirector.hpe.com/v1beta2":
				spec := convertedObject.Object["spec"]
				delete(convertedObject.Object, "spec")
				spec.(map[string]interface{})["defaultEventList"] = []string{"configure", "addnodes", "delnodes"}
				convertedObject.Object["spec"] = spec
			default:
				return &v1beta1.ConversionResponse{
					Result: metav1.Status{
						Message: fmt.Sprintf("unexpected conversion version %q", toVersion),
						Status:  metav1.StatusFailure,
					},
				}
			}
		case "kubedirector.hpe.com/v1beta2":
			switch toVersion {
			case "kubedirector.hpe.com/v1beta1":
				spec := convertedObject.Object["spec"]
				delete(convertedObject.Object, "spec")
				delete(spec.(map[string]interface{}), "defaultEventList")

				roles := spec.(map[string]interface{})["roles"]
				delete(spec.(map[string]interface{}), "roles")
				for _, roleConfig := range roles.([]interface{}) {
					delete(roleConfig.(map[string]interface{}), "eventList")
				}

				spec.(map[string]interface{})["roles"] = roles
				convertedObject.Object["spec"] = spec
			default:
				return &v1beta1.ConversionResponse{
					Result: metav1.Status{
						Message: fmt.Sprintf("unexpected conversion version %q", toVersion),
						Status:  metav1.StatusFailure,
					},
				}
			}
		default:
			return &v1beta1.ConversionResponse{
				Result: metav1.Status{
					Message: fmt.Sprintf("unexpected conversion version %q", toVersion),
					Status:  metav1.StatusFailure,
				},
			}
		}

		convertedObject.SetAPIVersion(convertRequest.DesiredAPIVersion)
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedObject})
	}

	return &v1beta1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
	}
}

func convertor(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	convertReview := v1beta1.ConversionReview{}
	_, _, err := serializer.Decode(body, nil, &convertReview)
	if err != nil {
		msg := fmt.Sprintf("failed to deserialize body (%v) with error %v", string(body), err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	convertReview.Response = convert(convertReview.Request)
	convertReview.Response.UID = convertReview.Request.UID
	convertReview.Request = &v1beta1.ConversionRequest{}

	accept := r.Header.Get("Accept")
	outSerializer := getOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid accept header `%s`", accept)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = outSerializer.Encode(&convertReview, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
