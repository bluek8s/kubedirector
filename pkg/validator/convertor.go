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
	"github.com/prometheus/common/log"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	cr *v1beta1.ConversionReview,
) *v1beta1.ConversionResponse {

	var convertedObjects []runtime.RawExtension
	var conversionResponse = v1beta1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result: metav1.Status{
			Message: metav1.StatusSuccess,
		},
	}

	log.Info("Am I getting lucky?")
	return &conversionResponse
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

	log.Info("Body: ", body)

	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	convertReview := v1beta1.ConversionReview{}
	_, gvk, err := serializer.Decode(body, nil, &convertReview)
	if err == nil {
		convertReview.Response = convert(&convertReview)
		convertReview.Response.UID = convertReview.Request.UID
	}

	log.Info("Kind of request is: ", gvk)

	// reset the request, it is not needed in a response.
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

	/*
		cr := v1beta1.ConversionReview{}
		if err := json.Unmarshal(body, &cr); err != nil {
			conversionResponse = &v1beta1.ConversionResponse{
				Result: metav1.Status{
					Message: err.Error(),
				},
			}
		} else {
			conversionResponse = convert(&cr)
		}

		cr.Request = &v1beta1.ConversionRequest{}

		conversionReview := v1beta1.ConversionReview{}
		if conversionResponse != nil {
			conversionReview.Response = conversionResponse
			if cr.Request != nil {
				conversionReview.Response.UID = cr.Request.UID
			}
		}

		respBytes, err := json.Marshal(conversionReview)
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
	*/

}
