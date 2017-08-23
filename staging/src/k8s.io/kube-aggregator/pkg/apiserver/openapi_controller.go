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
	"time"

	"github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

const (
	successfullUpdateDelay  = time.Minute
	failedUpdateMaxExpDelay = time.Hour
)

// OpenAPIAggregationManager is the interface between this controller and OpenAPI Aggregator service.
type OpenAPIAggregationManager interface {
	UpdateApiServiceSpec(apiServiceName string) (changed, deleted bool, err error)
}

// OpenAPIAggregationController periodically check for changes in OpenAPI specs of APIServices and update/remove
// them if necessary.
type OpenAPIAggregationController struct {
	openAPIAggregationManager OpenAPIAggregationManager
	queue                     workqueue.RateLimitingInterface
}

func NewOpenAPIAggregationController(openAPIAggregationManager OpenAPIAggregationManager) *OpenAPIAggregationController {
	c := &OpenAPIAggregationController{
		openAPIAggregationManager: openAPIAggregationManager,
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(successfullUpdateDelay, failedUpdateMaxExpDelay), "APIServiceOpenAPIAggregationControllerQueue1"),
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
	shouldBeRateLimited, deleted, err := c.openAPIAggregationManager.UpdateApiServiceSpec(key.(string))

	if err != nil {
		utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
	}

	if !deleted {
		if shouldBeRateLimited {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
			c.queue.AddAfter(key, successfullUpdateDelay)
		}
	}
	return true
}

func (c *OpenAPIAggregationController) AddAPIService(key string) {
	c.queue.AddAfter(key, time.Second)
}

func (c *OpenAPIAggregationController) UpdateAPIService(key string) {
	if c.queue.NumRequeues(key) > 0 {
		// The item has failed before. Remove it from failure queue and
		// update it immediately
		c.queue.Forget(key)
		c.queue.AddAfter(key, time.Second)
	}
	// Else: The item has been succeeded before and it will be updated soon (after successfullUpdateDelay)
	// we don't add it again as it will cause a duplication of items.
}
