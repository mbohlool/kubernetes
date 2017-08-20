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

type OpenAPIAggregationManager interface {
	UpdateApiServiceSpec(apiServiceName string) (changed, deleted bool, err error)
}

type OpenAPIAggregationController struct {
	openAPIAggregationManager OpenAPIAggregationManager
	queue                     workqueue.RateLimitingInterface
}

func NewOpenAPIAggregationController(openAPIAggregationManager OpenAPIAggregationManager) *OpenAPIAggregationController {
	c := &OpenAPIAggregationController{
		openAPIAggregationManager: openAPIAggregationManager,
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(time.Minute, time.Hour), "APIServiceOpenAPIAggregationController"),
	}

	return c
}

func (c *OpenAPIAggregationController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	glog.Infof("Starting OpenAPIAggregationController")
	defer glog.Infof("Shutting down OpenAPIAggregationController")

	go wait.Until(c.runWorker, time.Minute, stopCh)

	<-stopCh
}

func (c *OpenAPIAggregationController) runWorker() {
	keys := []string{}
	defer func() {
		for _, key := range keys {
			c.queue.Done(key)
		}
	}()

	for key, shutdown := c.queue.Get(); !shutdown; {
		keys = append(keys, key.(string))
		shouldBeRateLimited, deleted, err := c.openAPIAggregationManager.UpdateApiServiceSpec(key.(string))

		if err != nil {
			utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
		}

		if !deleted {
			if shouldBeRateLimited {
				// rate limit both updated or failed calls
				c.queue.AddRateLimited(key)
			} else {
				c.queue.Add(key)
			}
		}
	}
}

func (c *OpenAPIAggregationController) enqueue(key string) {
	c.queue.Add(key)
}
