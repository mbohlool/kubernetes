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
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"

	"k8s.io/apiserver/pkg/server"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
	"k8s.io/kube-openapi/pkg/aggregator"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/handler"
)

const (
	aggregatorUser                = "system:aggregator"
	specDownloadTimeout           = 60 * time.Second
	localDelegateChainNamePattern = "k8s_internal_local_delegation_chain_%d"

	// A randomly generated UUID to differentiate local and remote eTags.
	locallyGeneratedEtagPrefix = "\"6E8F849B434D4B98A569B9D7718876E9-"
)

type openAPIAggregator struct {
	// mutex protects All members of this struct.
	rwMutex sync.RWMutex

	// Map of API Services' OpenAPI specs by their name
	openAPISpecs map[string]*openAPISpecInfo

	// provided for dynamic OpenAPI spec
	openAPIService *handler.OpenAPIService
}

var _ OpenAPIAggregationManager = &openAPIAggregator{}

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
func buildAndRegisterOpenAPIAggregator(downloader *openAPIDownloader, delegationTarget server.DelegationTarget, webServices []*restful.WebService,
	config *common.Config, pathHandler common.PathHandler) (s *openAPIAggregator, err error) {
	s = &openAPIAggregator{
		openAPISpecs: map[string]*openAPISpecInfo{},
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
		delegateSpec, etag, _, err := downloader.Download(handler, "")
		if err != nil {
			return nil, err
		}
		if delegateSpec == nil {
			continue
		}
		specName := fmt.Sprintf(localDelegateChainNamePattern, i)

		aggregator.FilterSpecByPaths(delegateSpec, []string{"/apis/", "/api/", "/logs/", "/version/"})
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

	// Specification of this API Service. If null then the spec is not loaded yet.
	spec    *spec.Swagger
	handler http.Handler
	etag    string
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
	baseForMergeSpecName := fmt.Sprintf(localDelegateChainNamePattern, 0)
	localDelegateSpec, exists := s.openAPISpecs[baseForMergeSpecName]
	if !exists {
		return nil, fmt.Errorf("Cannot find base spec for merging.")
	}
	specToReturn, err = aggregator.CloneSpec(localDelegateSpec.spec)
	if err != nil {
		return nil, err
	}
	specs := []openAPISpecInfo{}
	for serviceName, specInfo := range s.openAPISpecs {
		if serviceName == baseForMergeSpecName || specInfo.spec == nil {
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

// tryUpdatingServiceSpecs tries updating openAPISpecs map with specified specInfo, and keeps the map intact
// if the update fails.
func (s *openAPIAggregator) tryUpdatingServiceSpecs(specInfo *openAPISpecInfo) error {
	orgSpecInfo, exists := s.openAPISpecs[specInfo.apiService.Name]
	s.openAPISpecs[specInfo.apiService.Name] = specInfo
	if err := s.updateOpenAPISpec(); err != nil {
		if exists {
			s.openAPISpecs[specInfo.apiService.Name] = orgSpecInfo
		} else {
			delete(s.openAPISpecs, specInfo.apiService.Name)
		}
		return err
	}
	return nil
}

// tryDeleteServiceSpecs tries delete specified specInfo from openAPISpecs map, and keeps the map intact
// if the update fails.
func (s *openAPIAggregator) tryDeleteServiceSpecs(apiServiceName string) error {
	orgSpecInfo, exists := s.openAPISpecs[apiServiceName]
	if !exists {
		return nil
	}
	delete(s.openAPISpecs, apiServiceName)
	if err := s.updateOpenAPISpec(); err != nil {
		s.openAPISpecs[apiServiceName] = orgSpecInfo
		return err
	}
	return nil
}

// UpdateApiServiceSpec updates the api service's OpenAPI spec. It is thread safe.
func (s *openAPIAggregator) UpdateApiServiceSpec(apiServiceName string, spec *spec.Swagger, etag string) error {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	specInfo, existingService := s.openAPISpecs[apiServiceName]
	if !existingService {
		return fmt.Errorf("APIService %q does not exists", apiServiceName)
	}

	return s.tryUpdatingServiceSpecs(&openAPISpecInfo{
		apiService: specInfo.apiService,
		spec:       spec,
		handler:    specInfo.handler,
		etag:       etag,
	})
}

// AddUpdateApiService adds or updates the api service. It is thread safe.
func (s *openAPIAggregator) AddUpdateApiService(handler http.Handler, apiService *apiregistration.APIService) error {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	newSpec := &openAPISpecInfo{
		apiService: *apiService,
		handler:    handler,
	}
	if specInfo, existingService := s.openAPISpecs[apiService.Name]; existingService {
		newSpec.etag = specInfo.etag
		newSpec.spec = specInfo.spec
	}
	return s.tryUpdatingServiceSpecs(newSpec)
}

// RemoveApiServiceSpec removes an api service from OpenAPI aggregation. If it does not exist, no error is returned.
// It is thread safe.
func (s *openAPIAggregator) RemoveApiServiceSpec(apiServiceName string) error {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	if _, existingService := s.openAPISpecs[apiServiceName]; !existingService {
		return nil
	}

	return s.tryDeleteServiceSpecs(apiServiceName)
}

// GetApiServiceSpec returns api service spec info
func (s *openAPIAggregator) GetApiServiceInfo(apiServiceName string) (handler http.Handler, etag string, exists bool) {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	if info, existingService := s.openAPISpecs[apiServiceName]; existingService {
		return info.handler, info.etag, true
	} else {
		return nil, "", false
	}
}
