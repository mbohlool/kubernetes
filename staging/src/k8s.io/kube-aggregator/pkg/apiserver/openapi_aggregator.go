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
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
	"k8s.io/kube-openapi/pkg/aggregator"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"
)

const (
	aggregatorUser      = "system:aggregator"
	specDownloadTimeout = 60 * time.Second
	localDelegateName   = ""
)

type openAPIAggregator struct {
	// Map of API Services' OpenAPI specs by their name
	openAPISpecs map[string]*openAPISpecInfo

	// provided for dynamic OpenAPI spec
	openAPIService *handler.OpenAPIService

	contextMapper request.RequestContextMapper

	openAPIAggregationController *OpenAPIAggregationController
}

func (s *openAPIAggregator) addLocalSpec(spec *spec.Swagger, localHandler http.Handler, name, etag string) {
	localApiService := apiregistration.APIService{}
	localApiService.Name = name
	localApiService.Spec.Service = nil
	s.openAPISpecs[name] = &openAPISpecInfo{
		etag:       etag,
		apiService: localApiService,
		handler:    localHandler,
		spec:       spec,
	}
}

func buildAndRegisterOpenAPIAggregator(delegateHandler http.Handler, webServices []*restful.WebService, config *common.Config, pathHandler common.PathHandler, contextMapper request.RequestContextMapper) (s *openAPIAggregator, err error) {
	s = &openAPIAggregator{
		openAPISpecs:  map[string]*openAPISpecInfo{},
		contextMapper: contextMapper,
	}
	s.openAPIAggregationController = NewOpenAPIAggregationController(s)

	delegateSpec, etag, _, err := s.downloadOpenAPISpec(delegateHandler, "")
	if err != nil {
		return nil, err
	}
	s.addLocalSpec(delegateSpec, delegateHandler, localDelegateName, etag)

	// Build Aggregator's spec
	aggregatorOpenAPISpec, err := builder.BuildOpenAPISpec(
		webServices, config)
	if err != nil {
		return nil, err
	}
	// Remove any non-API endpoints from aggregator's spec. aggregatorOpenAPISpec
	// is the source of truth for all non-api endpoints.
	aggregator.FilterSpecByPaths(aggregatorOpenAPISpec, []string{"/apis/"})

	s.addLocalSpec(aggregatorOpenAPISpec, nil, "", "")

	// Build initial spec to serve.
	specToServe, err := s.buildOpenAPISpec()
	if err != nil {
		return nil, err
	}

	// Install handler
	s.openAPIService, err = handler.RegisterOpenAPIService(
		specToServe, "/swagger.json", pathHandler)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// openAPISpecInfo is used to store OpenAPI spec with its priority.
// It can be used to sort specs with their priorities.
type openAPISpecInfo struct {
	apiService apiregistration.APIService
	spec       *spec.Swagger
	handler    http.Handler
	etag       string
}

// byPriority can be used in sort.Sort to sort specs with their priorities.
type byPriority struct {
	specs           []openAPISpecInfo
	groupPriorities map[string]int32
}

func (a byPriority) Len() int      { return len(a.specs) }
func (a byPriority) Swap(i, j int) { a.specs[i], a.specs[j] = a.specs[j], a.specs[i] }
func (a byPriority) Less(i, j int) bool {
	// All local specs will come first
	if a.specs[i].apiService.Spec.Service == nil {
		return true
	}
	var iPriority, jPriority int32
	if a.specs[i].apiService.Spec.Group == a.specs[j].apiService.Spec.Group {
		iPriority = a.specs[i].apiService.Spec.VersionPriority
		jPriority = a.specs[i].apiService.Spec.VersionPriority
	} else {
		iPriority = a.groupPriorities[a.specs[i].apiService.Spec.Group]
		jPriority = a.groupPriorities[a.specs[j].apiService.Spec.Group]
	}
	if iPriority != jPriority {
		// Sort by priority, higher first
		return iPriority > jPriority
	}
	// Sort by service name.
	return a.specs[i].apiService.Name < a.specs[j].apiService.Name
}

func sortByPriority(specs []openAPISpecInfo) {
	b := byPriority{
		specs:           specs,
		groupPriorities: map[string]int32{},
	}
	for _, spec := range specs {
		if spec.apiService.Spec.Service == nil {
			continue
		}
		if pr, found := b.groupPriorities[spec.apiService.Spec.Group]; !found || spec.apiService.Spec.GroupPriorityMinimum > pr {
			b.groupPriorities[spec.apiService.Spec.Group] = spec.apiService.Spec.GroupPriorityMinimum
		}
	}
	sort.Sort(b)
}

// buildOpenAPISpec aggregates all OpenAPI specs.  It is not thread-safe.
func (s *openAPIAggregator) buildOpenAPISpec() (specToReturn *spec.Swagger, err error) {
	localDelegateSpec, exists := s.openAPISpecs[localDelegateName]
	if !exists {
		return nil, fmt.Errorf("localDelegate spec is missing")
	}
	specToReturn, err = aggregator.CloneSpec(localDelegateSpec.spec)
	if err != nil {
		return nil, err
	}
	specs := []openAPISpecInfo{}
	for serviceName, specInfo := range s.openAPISpecs {
		if serviceName == localDelegateName {
			continue
		}
		specs = append(specs, openAPISpecInfo{specInfo.apiService, specInfo.spec, specInfo.handler, specInfo.etag})
	}
	sortByPriority(specs)
	for _, specInfo := range specs {
		if err := aggregator.MergeSpecs(specToReturn, specInfo.spec); err != nil {
			return nil, err
		}
	}
	return specToReturn, nil
}

// updateOpenAPISpec aggregates all OpenAPI specs.  It is not thread-safe.
func (s *openAPIAggregator) updateOpenAPISpec() error {
	if s.openAPIService == nil {
		return nil
	}
	specToServe, err := s.buildOpenAPISpec()
	if err != nil {
		return err
	}
	return s.openAPIService.UpdateSpec(specToServe)
}

// inMemoryResponseWriter is a http.Writer that keep the response in memory.
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

func (r *inMemoryResponseWriter) String() string {
	s := fmt.Sprintf("ResponseCode: %d", r.respCode)
	if r.data != nil {
		s += fmt.Sprintf(", Body: %s", string(r.data))
	}
	if r.header != nil {
		s += fmt.Sprintf(", Header: %s", r.header)
	}
	return s
}

func (s *openAPIAggregator) handlerWithUser(handler http.Handler, info user.Info) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ctx, ok := s.contextMapper.Get(req); ok {
			s.contextMapper.Update(req, request.WithUser(ctx, info))
		}
		handler.ServeHTTP(w, req)
	})
}

func etagFor(data []byte) string {
	return fmt.Sprintf("\"%X\"", sha512.Sum512(data))
}

// downloadOpenAPISpec downloads openAPI spec from /swagger.json endpoint of the given handler.
// httpStatus is only valid if err == nil
func (s *openAPIAggregator) downloadOpenAPISpec(handler http.Handler, etag string) (returnSpec *spec.Swagger, newEtag string, httpStatus int, err error) {
	handler = s.handlerWithUser(handler, &user.DefaultInfo{Name: aggregatorUser})
	handler = request.WithRequestContext(handler, s.contextMapper)
	handler = http.TimeoutHandler(handler, specDownloadTimeout, "request timed out")

	req, err := http.NewRequest("GET", "/swagger.json", nil)
	if err != nil {
		return nil, "", 0, err
	}
	if len(etag) > 0 {
		req.Header.Add("If-None-Match", etag)
	}
	writer := newInMemoryResponseWriter()
	handler.ServeHTTP(writer, req)

	switch writer.respCode {
	case http.StatusNotModified:
		if len(etag) == 0 {
			return nil, etag, http.StatusNotModified, fmt.Errorf("http.StatusNotModified is not allowed in absense of etag")
		}
		return nil, etag, http.StatusNotModified, nil
	case http.StatusNotFound:
		// Gracefully skip 404, assuming the server won't provide any spec
		return nil, "", http.StatusNotFound, nil
	case http.StatusOK:
		openApiSpec := &spec.Swagger{}
		if err := json.Unmarshal(writer.data, openApiSpec); err != nil {
			return nil, "", 0, err
		}
		etag = writer.Header().Get("Etag")
		if len(etag) == 0 {
			etag = etagFor(writer.data)
		}
		return openApiSpec, etag, http.StatusOK, nil
	default:
		return nil, "", 0, fmt.Errorf("failed to retrive openAPI spec, http error: %s", writer.String())
	}
}

// loadApiServiceSpec loads OpenAPI spec for the given API Service and then updates aggregator's spec.
func (s *openAPIAggregator) loadApiServiceSpec(handler http.Handler, apiService *apiregistration.APIService) error {

	// Ignore local services
	if apiService.Spec.Service == nil {
		return nil
	}

	openApiSpec, etag, _, err := s.downloadOpenAPISpec(handler, "")
	if err != nil {
		return err
	}
	if openApiSpec == nil {
		// API service does not provide an OpenAPI spec, ignore it.
		return nil
	}
	aggregator.FilterSpecByPaths(openApiSpec, []string{"/apis/" + apiService.Spec.Group + "/"})

	_, existingService := s.openAPISpecs[apiService.Name]
	s.openAPISpecs[apiService.Name] = &openAPISpecInfo{
		apiService: *apiService,
		spec:       openApiSpec,
		handler:    handler,
		etag:       etag,
	}

	err = s.updateOpenAPISpec()
	if err != nil {
		delete(s.openAPISpecs, apiService.Name)
		return err
	}
	if !existingService {
		s.openAPIAggregationController.enqueue(apiService.Name)
	}

	return nil
}

func (s *openAPIAggregator) removeApiServiceSpec(apiServiceName string) error {
	if _, existingService := s.openAPISpecs[apiServiceName]; existingService {
		oldSpecs := s.openAPISpecs
		delete(s.openAPISpecs, apiServiceName)
		err := s.updateOpenAPISpec()
		if err != nil {
			s.openAPISpecs = oldSpecs
			return err
		}
	}
	return nil
}

func (s *openAPIAggregator) UpdateApiServiceSpec(apiServiceName string) (changed, deleted bool, err error) {
	specInfo, existingService := s.openAPISpecs[apiServiceName]
	if !existingService {
		return false, true, nil
	}
	// If this is a local spec with no handler, that means it is a static spec.
	if specInfo.handler == nil {
		return false, false, nil
	}
	spec, etag, err := s.downloadOpenAPISpec(specInfo.handler, specInfo.etag)
	if err != nil {
		return false, false, err
	}
	if spec == nil {
		return false, false, nil
	}
	oldSpec := specInfo.spec
	specInfo.spec = spec
	if err := s.updateOpenAPISpec(); err != nil {
		specInfo.spec = oldSpec
		return false, false, err
	}
	specInfo.etag = etag
	return true, false, nil
}
