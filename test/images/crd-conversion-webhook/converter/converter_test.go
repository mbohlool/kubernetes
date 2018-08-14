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
	"net/http"
	"strings"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type inMemoryResponseWriter struct {
	writeHeaderCalled bool
	header            http.Header
	respCode          int
	data              []byte
}

func newInMemoryResponseWriter() *inMemoryResponseWriter {
	return &inMemoryResponseWriter{header: http.Header{}}
}

func (r *inMemoryResponseWriter) Header() http.Header {
	return r.header
}

func (r *inMemoryResponseWriter) WriteHeader(code int) {
	r.writeHeaderCalled = true
	r.respCode = code
}

func (r *inMemoryResponseWriter) Write(in []byte) (int, error) {
	if !r.writeHeaderCalled {
		r.WriteHeader(http.StatusOK)
	}
	r.data = append(r.data, in...)
	return len(in), nil
}

func TestConverter(t *testing.T) {
	sampleObj := `kind: ConversionReview
apiVersion: apiextensions.k8s.io/v1beta1
request:
  uid: 0000-0000-0000-0000
  desiredAPIVersion: stable.example.com/v2
  objects:
    - apiVersion: stable.example.com/v1
      kind: CronTab
      metadata:
        name: my-new-cron-object
      spec:
        cronSpec: "* * * * */5"
        image: my-awesome-cron-image
      hostPort: "localhost:7070"
`
	response := newInMemoryResponseWriter()
	request, err := http.NewRequest("POST", "/convert", strings.NewReader(sampleObj))
	request.Header.Add("Content-Type", "application/json")
	if err != nil {
		t.Fatal(err)
	}
	ServeExampleConvert(response, request)
	convertReview := v1beta1.ConversionReview{}
	deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err := deserializer.Decode(response.data, nil, &convertReview); err != nil {
		t.Fatal(err)
	}
	if convertReview.Response.Result.Status != v1.StatusSuccess {
		t.Fatalf("cr conversion failed: %v", convertReview.Response)
	}
	convertedObj := unstructured.Unstructured{}
	if _, _, err := deserializer.Decode(convertReview.Response.ConvertedObjects[0].Raw, nil, &convertedObj); err != nil {
		t.Fatal(err)
	}
	if e, a := "localhost", convertedObj.Object["host"]; e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}
	if e, a := "7070", convertedObj.Object["port"]; e != a {
		t.Errorf("expected= %v, actual= %v", e, a)
	}
}
