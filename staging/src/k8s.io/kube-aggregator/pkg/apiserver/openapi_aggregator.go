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
	"strings"
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

		// We would find first spec with "/api/" endpoints and consider it the
		// base for the merge. Any other spec would be filtered for their "/apis/"
		// endpoints only.
		filterPaths := true
		if len(s.baseForMergeSpecName) == 0 {
			for path := range delegateSpec.Paths.Paths {
				if strings.HasPrefix(path, "/api/") {
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
		if serviceName == s.baseForMergeSpecName || specInfo.spec == nil {
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

// tryUpdatingServiceSpecs tries updating openAPISpecs map, and keep the map intact if the update fails.
// note that map's elements are pointers, so the update method should not change them.
func (s *openAPIAggregator) tryUpdatingServiceSpecs(updateFunc func() error) error {
	// Keep a backup value and restore it if aggregation failed.
	backupMap := s.openAPISpecs
	if err := updateFunc(); err != nil {
		s.openAPISpecs = backupMap
		return err
	}
	return nil
}

// UpdateApiServiceSpec updates the api service's OpenAPI spec. It is thread safe.
func (s *openAPIAggregator) UpdateApiServiceSpec(apiServiceName string, spec *spec.Swagger, etag string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	specInfo, existingService := s.openAPISpecs[apiServiceName]
	if !existingService {
		return fmt.Errorf("APIService %s does not exists", apiServiceName)
	}

	return s.tryUpdatingServiceSpecs(func() error {
		s.openAPISpecs[apiServiceName] = &openAPISpecInfo{
			apiService: specInfo.apiService,
			spec:       spec,
			handler:    specInfo.handler,
			etag:       etag,
		}
		return s.updateOpenAPISpec()
	})
}

// AddUpdateApiService adds or updates the api service. It is thread safe.
func (s *openAPIAggregator) AddUpdateApiService(handler http.Handler, apiService *apiregistration.APIService) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.tryUpdatingServiceSpecs(func() error {
		if specInfo, existingService := s.openAPISpecs[apiService.Name]; existingService {
			specInfo.handler = handler
			specInfo.apiService = *apiService
		} else {
			s.openAPISpecs[apiService.Name] = &openAPISpecInfo{
				apiService: *apiService,
				handler:    handler,
			}
		}
		return s.updateOpenAPISpec()
	})
}

// RemoveApiServiceSpec removes an api service from OpenAPI aggregation. It is thread safe.
func (s *openAPIAggregator) RemoveApiServiceSpec(apiServiceName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, existingService := s.openAPISpecs[apiServiceName]; !existingService {
		return nil
	}

	return s.tryUpdatingServiceSpecs(func() error {
		delete(s.openAPISpecs, apiServiceName)
		return s.updateOpenAPISpec()
	})
}

// GetApiServiceSpec returns api service spec info
func (s *openAPIAggregator) GetApiServiceInfo(apiServiceName string) (handler http.Handler, etag string, exists bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if info, existingService := s.openAPISpecs[apiServiceName]; existingService {
		return info.handler, info.etag, true
	} else {
		return nil, "", false
	}
}
