/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package converter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// convertFunc is the user defined function for any conversion. The code in this file is a
// template that can be use for any CRD conversion given this function.
type convertFunc func(Object *unstructured.Unstructured, version string) (*unstructured.Unstructured, *metav1.Status)

// toConversionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toConversionResponse(err error) *v1beta1.ConversionResponse {
	return &v1beta1.ConversionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func statusWithMessage(msg string, params ...string) *metav1.Status {
	return &metav1.Status{
		Message: fmt.Sprintf(msg, params),
	}
}

// doConversion converts the requested object given the conversion function and returns a conversion response.
// failures will be reported as Reason in the conversion response.
func doConversion(convertRequest *v1beta1.ConversionRequest, convert convertFunc) *v1beta1.ConversionResponse {
	cr := unstructured.Unstructured{}
	if err := cr.UnmarshalJSON(convertRequest.Object.Raw); err != nil {
		glog.Error(err)
		return toConversionResponse(err)
	}
	var convertedCR *unstructured.Unstructured
	var status *metav1.Status
	if convertRequest.IsList {
		listCR, err := cr.ToList()
		if err != nil {
			glog.Error(err)
			return toConversionResponse(err)
		}
		convertedList := listCR.DeepCopy()
		for i := 0; i < len(convertedList.Items); i++ {
			item, status := convert(&convertedList.Items[i], convertRequest.APIVersion)
			if status != nil {
				glog.Error(status.String())
				return &v1beta1.ConversionResponse{
					Result: status,
				}
			}
			item.SetAPIVersion(convertRequest.APIVersion)
			convertedList.Items[i] = *item
		}
		convertedCR = &unstructured.Unstructured{}
		convertedCR.SetUnstructuredContent(convertedList.UnstructuredContent())
	} else {
		convertedCR, status = convert(&cr, convertRequest.APIVersion)
		if status != nil {
			glog.Error(status.String())
			return &v1beta1.ConversionResponse{
				Result: status,
			}
		}
	}
	convertedCR.SetAPIVersion(convertRequest.APIVersion)
	return &v1beta1.ConversionResponse{
		ConvertedObject: &runtime.RawExtension{
			Object: convertedCR,
		},
	}
}

func serve(w http.ResponseWriter, r *http.Request, convert convertFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := fmt.Errorf("contentType=%s, expect application/json", contentType)
		glog.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	glog.V(2).Info(fmt.Sprintf("handling request: %v", body))
	convertReview := v1beta1.ConversionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &convertReview); err != nil {
		glog.Error(err)
		convertReview.Response = toConversionResponse(err)
	} else {
		convertReview.Response = doConversion(convertReview.Request, convert)
	}
	glog.V(2).Info(fmt.Sprintf("sending response: %v", convertReview.Response))

	convertReview.Response.UID = convertReview.Request.UID
	// reset the request, it is not needed in a response.
	convertReview.Request = &v1beta1.ConversionRequest{}

	resp, err := json.Marshal(convertReview)
	if err != nil {
		glog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
		return
	}
}

func ServeExampleConvert(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertExampleCRD)
}
