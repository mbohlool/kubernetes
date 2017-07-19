/*
Copyright 2017 The Kubernetes Authors.

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

package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
	"k8s.io/kube-openapi/pkg/aggregator"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"

	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
)

type openAPIAggregator struct {
	openAPISpecs map[string]*openAPISpecInfo

	// provided for dynamic OpenAPI spec
	openAPIService *handler.OpenAPIService

	// Aggregator's openapi spec.
	aggregatorsOpenAPISpec *spec.Swagger

	// Local delegate's openapi spec.
	localDelegatesOpenAPISpec *spec.Swagger
}

func newOpenAPIAggregator(delegateHandler http.Handler, webServices []*restful.WebService, config *common.Config, pathHandler common.PathHandler) (s *openAPIAggregator, err error) {
	s = &openAPIAggregator{}
	s.localDelegatesOpenAPISpec, err = loadOpenAPISpec(delegateHandler)
	if err != nil {
		return nil, err
	}
	// Install Aggregator's OpenAPI Handler with an initial spec
	s.aggregatorsOpenAPISpec, err = builder.BuildOpenAPISpec(
		webServices, config)
	if err != nil {
		return nil, err
	}
	// Remove any non-API endpoints from aggregator's spec. aggregatorsOpenAPISpec
	// is the source of truth for al non-api endpoints.
	aggregator.FilterSpecByPaths(s.aggregatorsOpenAPISpec, []string{"/apis/"})
	s.openAPIService, err = handler.RegisterOpenAPIService(
		s.localDelegatesOpenAPISpec, "/swagger.json", pathHandler)
	if err != nil {
		return nil, err
	}
	if err = s.updateOpenAPISpec(); err != nil {
		return nil, err
	}
	return s, nil
}

type openAPISpecInfo struct {
	serviceName string
	spec        *spec.Swagger
	priority    int32
}

type byPriority []openAPISpecInfo

func (a byPriority) Len() int      { return len(a) }
func (a byPriority) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool {
	if a[i].priority != a[j].priority {
		// Sort by priority, higher first
		return a[i].priority > a[j].priority
	} else {
		// Sort by service name, lower first.
		return a[i].serviceName < a[j].serviceName
	}
}

// updateOpenAPISpec aggregates all OpenAPI specs.  It is not thread-safe.
func (s *openAPIAggregator) updateOpenAPISpec() error {
	if s.openAPIService == nil {
		return nil
	}
	specToServe, err := aggregator.CloneSpec(s.localDelegatesOpenAPISpec)
	if err != nil {
		return err
	}
	if err := aggregator.MergeSpecs(specToServe, s.aggregatorsOpenAPISpec); err != nil {
		return fmt.Errorf("cannot merge local delegate spec with aggregator spec: %s", err.Error())
	}
	specs := []openAPISpecInfo{}
	for name, specInfo := range s.openAPISpecs {
		specs = append(specs, openAPISpecInfo{name, specInfo.spec, specInfo.priority})
	}
	sort.Sort(byPriority(specs))
	for _, specInfo := range specs {
		if err := aggregator.MergeSpecs(specToServe, specInfo.spec); err != nil {
			return err
		}
	}
	return s.openAPIService.UpdateSpec(specToServe)
}

type inMemoryResponseWriter struct {
	header   http.Header
	respCode int
	data     []byte
}

func (r *inMemoryResponseWriter) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}

func (r *inMemoryResponseWriter) WriteHeader(code int) {
	r.respCode = code
}

func (r *inMemoryResponseWriter) Write(in []byte) (int, error) {
	r.data = append(r.data, in...)
	return len(in), nil
}

func (r *inMemoryResponseWriter) ErrorMessage() string {
	s := fmt.Sprintf("ResponseCode: %d" , r.respCode)
	if r.data != nil {
		s += fmt.Sprintf(", Body: %s", string(r.data))
	}
	return s
}

// inMemoryResponseWriter checks response code first. If response code is http OK then it will unmarshal the response
// into json object v.
func (r *inMemoryResponseWriter) jsonUnmarshal(v interface{}) error {
	switch r.respCode {
	case http.StatusOK:
		if err := json.Unmarshal(r.data, v); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("failed to retrive openAPI spec, http error: %s", r.ErrorMessage())
	}
}

func loadOpenAPISpec(handler http.Handler) (*spec.Swagger, error) {
	req, err := http.NewRequest("GET", "/swagger.json", nil)
	if err != nil {
		return nil, err
	}
	writer := inMemoryResponseWriter{}
	handler.ServeHTTP(&writer, req)
	openApiSpec := &spec.Swagger{}
	if err := writer.jsonUnmarshal(openApiSpec); err != nil {
		return nil, err
	}
	return openApiSpec, nil
}

func max(i, j int32) int32 {
	if i > j {
		return i
	}
	return j
}

func (s *openAPIAggregator) loadApiServiceSpec(handler http.Handler, apiService *apiregistration.APIService) error {
	if apiService.Spec.Service == nil {
		return nil
	}
	openApiSpec, err := loadOpenAPISpec(handler)
	if err != nil {
		return err
	}
	aggregator.FilterSpecByPaths(openApiSpec, []string{"/apis/" + apiService.Spec.Group + "/"})
	s.openAPISpecs[apiService.Name] = &openAPISpecInfo{
		serviceName: apiService.Name,
		spec:        openApiSpec,
		priority:    max(apiService.Spec.VersionPriority, apiService.Spec.GroupPriorityMinimum),
	}
	err = s.updateOpenAPISpec()
	if err != nil {
		delete(s.openAPISpecs, apiService.Name)
		return err
	}
	return nil
}
