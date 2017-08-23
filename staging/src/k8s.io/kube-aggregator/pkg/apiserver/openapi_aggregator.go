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
	"k8s.io/apiserver/pkg/server"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
	"k8s.io/kube-openapi/pkg/aggregator"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"
	"strings"
	"sync"
)

const (
	aggregatorUser                = "system:aggregator"
	specDownloadTimeout           = 60 * time.Second
	localDelegateChainNamePattern = "k8s_internal_local_delegation_chain_%d"

	// A randomly generated UUID to differentiate local and remote eTags.
	locallyGeneratedEtagPrefix = "\"6E8F849B434D4B98A569B9D7718876E9-"
)

type openAPIAggregator struct {
	// rwMutex protects All members of this struct.
	mutex sync.Mutex

	// Map of API Services' OpenAPI specs by their name
	openAPISpecs map[string]*openAPISpecInfo

	// openAPISpecs[baseForMergeSpecName] is the spec that should be used
	// for the basis of merge. That means we would clone this spec first
	// and then merge others into it. The spec would have `/api` endpoints
	// and all other information such as Swagger.Info will be used for final
	// spec.
	baseForMergeSpecName string

	// provided for dynamic OpenAPI spec
	openAPIService *handler.OpenAPIService

	contextMapper request.RequestContextMapper

	openAPIAggregationController *OpenAPIAggregationController
}

var _ OpenAPIAggregationManager = openAPIAggregator{}

// This function is not thread safe as it only being called on startup.
func (s *openAPIAggregator) addLocalSpec(spec *spec.Swagger, localHandler http.Handler, name, etag string) {
	localApiService := apiregistration.APIService{}
	localApiService.Name = name
	s.openAPISpecs[name] = &openAPISpecInfo{
		etag:       etag,
		apiService: localApiService,
		handler:    localHandler,
		spec:       spec,
	}
}

// This function is not thread safe as it only being called on startup.
func buildAndRegisterOpenAPIAggregator(delegationTarget server.DelegationTarget, webServices []*restful.WebService,
	config *common.Config, pathHandler common.PathHandler, contextMapper request.RequestContextMapper) (s *openAPIAggregator, err error) {
	s = &openAPIAggregator{
		openAPISpecs:  map[string]*openAPISpecInfo{},
		contextMapper: contextMapper,
		openAPIAggregationController: NewOpenAPIAggregationController(s),
	}

	i := 0
	// Build Aggregator's spec
	aggregatorOpenAPISpec, err := builder.BuildOpenAPISpec(
		webServices, config)
	if err != nil {
		return nil, err
	}
	aggregator.FilterSpecByPaths(aggregatorOpenAPISpec, []string{"/apis/"})

	// Reserving non-name spec for aggregator's Spec.
	s.addLocalSpec(aggregatorOpenAPISpec, nil, fmt.Sprintf(localDelegateChainNamePattern, i), "")
	i++
	for delegate := delegationTarget; delegate != nil; delegate = delegate.NextDelegate() {
		handler := delegate.UnprotectedHandler()
		if handler == nil {
			continue
		}
		delegateSpec, etag, _, err := s.downloadOpenAPISpec(handler, "")
		if err != nil {
			return nil, err
		}
		if delegateSpec == nil {
			continue
		}
		specName := fmt.Sprintf(localDelegateChainNamePattern, i)

		// We would find first spec with "/api/" endpoints and consider it the
		// base for the merge. Any other spec would be filtered for their "/apis/"
		// endpoints only.
		filterPaths := true
		if len(s.baseForMergeSpecName) == 0 {
			for path := range delegateSpec.Paths.Paths {
				if strings.HasPrefix(path,"/api/") {
					s.baseForMergeSpecName = specName
					filterPaths = false
					break
				}
			}
		}
		if filterPaths {
			aggregator.FilterSpecByPaths(delegateSpec, []string{"/apis/"})
		}
		s.addLocalSpec(delegateSpec, handler, specName, etag)
		i++
	}

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
	// WARNING: This will result in not following priorities for local APIServices.
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

// buildOpenAPISpec aggregates all OpenAPI specs.  It is not thread-safe. The caller is responsible to hold proper locks.
func (s *openAPIAggregator) buildOpenAPISpec() (specToReturn *spec.Swagger, err error) {
	localDelegateSpec, exists := s.openAPISpecs[s.baseForMergeSpecName]
	if !exists {
		return nil, fmt.Errorf("Cannot find base spec for merging.")
	}
	specToReturn, err = aggregator.CloneSpec(localDelegateSpec.spec)
	if err != nil {
		return nil, err
	}
	specs := []openAPISpecInfo{}
	for serviceName, specInfo := range s.openAPISpecs {
		if serviceName == s.baseForMergeSpecName {
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

// updateOpenAPISpec aggregates all OpenAPI specs.  It is not thread-safe. The caller is responsible to hold proper locks.
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
	return fmt.Sprintf("%s%X\"", locallyGeneratedEtagPrefix, sha512.Sum512(data))
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

	// Only pass eTag if it is not generated locally
	if len(etag) > 0 && !strings.HasPrefix(etag, locallyGeneratedEtagPrefix) {
		req.Header.Add("If-None-Match", etag)
	}

	writer := newInMemoryResponseWriter()
	handler.ServeHTTP(writer, req)

	switch writer.respCode {
	case http.StatusNotModified:
		if len(etag) == 0 {
			return nil, etag, http.StatusNotModified, fmt.Errorf("http.StatusNotModified is not allowed in absence of etag")
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
		newEtag = writer.Header().Get("Etag")
		if len(newEtag) == 0 {
			newEtag = etagFor(writer.data)
			if len(etag) > 0 && strings.HasPrefix(etag, locallyGeneratedEtagPrefix) {
				// The function call with an etag and server does not report an etag.
				// That means this server does not support etag and the etag that passed
				// to the function generated previously by us. Just compare etags and
				// return StatusNotModified if they are the same.
				if etag == newEtag {
					return nil, etag, http.StatusNotModified, nil
				}
			}
		}
		return openApiSpec, etag, http.StatusOK, nil
	default:
		return nil, "", 0, fmt.Errorf("failed to retrive openAPI spec, http error: %s", writer.String())
	}
}

// LoadApiServiceSpec loads OpenAPI spec for the given API Service and then updates aggregator's spec.  It is thread safe.
func (s *openAPIAggregator) LoadApiServiceSpec(handler http.Handler, apiService *apiregistration.APIService) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if apiService.DisableOpenAPIAggregation != nil && *apiService.DisableOpenAPIAggregation {
		return nil
	}

	// Ignore local services
	if apiService.Spec.Service == nil {
		return nil
	}

	openApiSpec, etag, _, err := s.downloadOpenAPISpec(handler, "")
	if err != nil {
		return err
	}
	if openApiSpec == nil {
		// API service does not provide an OpenAPI spec.
		s.removeApiServiceSpec(apiService.Name)
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

// RemoveApiServiceSpec removes an api service from OpenAPI aggregation. It is thread safe.
func (s *openAPIAggregator) RemoveApiServiceSpec(apiServiceName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.removeApiServiceSpec(apiServiceName)
}

// removeApiServiceSpec is the internal version of RemoveApiServiceSpec that does not hold any lock and safe to call
// from a method that holds the lock.
func (s *openAPIAggregator) removeApiServiceSpec(apiServiceName string) error {
	if _, existingService := s.openAPISpecs[apiServiceName]; !existingService {
		return nil
	}
	oldSpecs := s.openAPISpecs
	delete(s.openAPISpecs, apiServiceName)
	err := s.updateOpenAPISpec()
	if err != nil {
		s.openAPISpecs = oldSpecs
		return err
	}
	return nil
}

// UpdateApiServiceSpec updates and aggregation OpenAPI spec for given service name. It is thread safe.
func (s *openAPIAggregator) UpdateApiServiceSpec(apiServiceName string) (shouldBeRateLimited, deleted bool, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	specInfo, existingService := s.openAPISpecs[apiServiceName]
	if !existingService {
		return false, true, nil
	}
	// If this is a local spec with no handler, that means it is a static spec.
	if specInfo.handler == nil {
		return false, false, nil
	}
	spec, etag, httpStatus, err := s.downloadOpenAPISpec(specInfo.handler, specInfo.etag)
	switch {
	case err != nil:
		return true, false, err
	case httpStatus == http.StatusNotModified:
		return false, false, nil
	case httpStatus == http.StatusNotFound || spec == nil:
		return true, false, nil
	}

	oldSpec := specInfo.spec
	specInfo.spec = spec
	if err := s.updateOpenAPISpec(); err != nil {
		specInfo.spec = oldSpec
		return true, false, err
	}
	specInfo.etag = etag
	return true, false, nil
}
