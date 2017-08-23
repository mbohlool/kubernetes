/*
Copyright 2016 The Kubernetes Authors.

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
	"time"

	"github.com/go-openapi/spec"
	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
)

const (
	successfulUpdateDelay   = time.Minute
	failedUpdateMaxExpDelay = time.Hour
)

// OpenAPIAggregationManager is the interface between this controller and OpenAPI Aggregator service.
type OpenAPIAggregationManager interface {
	AddUpdateApiService(handler http.Handler, apiService *apiregistration.APIService) error
	UpdateApiServiceSpec(apiServiceName string, spec *spec.Swagger, etag string) error
	RemoveApiServiceSpec(apiServiceName string) error
	GetApiServiceInfo(apiServiceName string) (handler http.Handler, etag string, exists bool)
}

// OpenAPIAggregationController periodically check for changes in OpenAPI specs of APIServices and update/remove
// them if necessary.
type OpenAPIAggregationController struct {
	openAPIAggregationManager OpenAPIAggregationManager
	queue                     workqueue.RateLimitingInterface
	downloader                *openAPIDownloader
}

func NewOpenAPIAggregationController(downloader *openAPIDownloader, openAPIAggregationManager OpenAPIAggregationManager) *OpenAPIAggregationController {
	c := &OpenAPIAggregationController{
		openAPIAggregationManager: openAPIAggregationManager,
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(successfulUpdateDelay, failedUpdateMaxExpDelay), "APIServiceOpenAPIAggregationControllerQueue1"),
		downloader: downloader,
	}

	return c
}

func (c *OpenAPIAggregationController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	glog.Infof("Starting OpenAPIAggregationController")
	defer glog.Infof("Shutting down OpenAPIAggregationController")

	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *OpenAPIAggregationController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *OpenAPIAggregationController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	defer c.queue.Done(key)
	if quit {
		return false
	}
	apiServiceName := key.(string)
	handler, etag, exists := c.openAPIAggregationManager.GetApiServiceInfo(apiServiceName)
	if !exists || handler == nil {
		c.queue.Forget(key)
		return true
	}
	returnSpec, newEtag, httpStatus, err := c.downloader.Download(handler, etag)
	rateLimit := false
	switch {
	case err != nil:
		utilruntime.HandleError(fmt.Errorf("loading OpenAPI spec for %q failed with : %v", apiServiceName, err))
		rateLimit = true
	case httpStatus == http.StatusNotModified:
		rateLimit = false
	case httpStatus == http.StatusNotFound || returnSpec == nil:
		rateLimit = true
	case httpStatus == http.StatusOK:
		if err := c.openAPIAggregationManager.UpdateApiServiceSpec(apiServiceName, returnSpec, newEtag); err != nil {
			utilruntime.HandleError(fmt.Errorf("aggregating OpenAPI spec for %q failed with : %v", apiServiceName, err))
			rateLimit = true
		}
	}

	if rateLimit {
		c.queue.AddRateLimited(key)
	} else {
		c.queue.Forget(key)
		c.queue.AddAfter(key, successfulUpdateDelay)
	}
	return true
}

func (c *OpenAPIAggregationController) AddAPIService(handler http.Handler, apiService *apiregistration.APIService) {
	if err := c.openAPIAggregationManager.AddUpdateApiService(handler, apiService); err != nil {
		utilruntime.HandleError(fmt.Errorf("adding %q to OpenAPIAggregationController failed with : %v", apiService.Name, err))
	}
	c.queue.AddAfter(apiService.Name, time.Second)
}

func (c *OpenAPIAggregationController) UpdateAPIService(handler http.Handler, apiService *apiregistration.APIService) {
	if err := c.openAPIAggregationManager.AddUpdateApiService(handler, apiService); err != nil {
		utilruntime.HandleError(fmt.Errorf("updating %q to OpenAPIAggregationController failed with : %v", apiService.Name, err))
	}
	key := apiService.Name
	if c.queue.NumRequeues(key) > 0 {
		// The item has failed before. Remove it from failure queue and
		// update it immediately
		c.queue.Forget(key)
		c.queue.AddAfter(key, time.Second)
	}
	// Else: The item has been succeeded before and it will be updated soon (after successfulUpdateDelay)
	// we don't add it again as it will cause a duplication of items.
}

func (c *OpenAPIAggregationController) RemoveAPIService(apiServiceName string) {
	if err := c.openAPIAggregationManager.RemoveApiServiceSpec(apiServiceName); err != nil {
		utilruntime.HandleError(fmt.Errorf("removing %q from OpenAPIAggregationController failed with : %v", apiServiceName, err))
	}
	// This will only remove it if it was failing before. If it was successful, processNextWorkItem will figure it out
	// and will not add it again to the queue.
	c.queue.Forget(apiServiceName)
}
