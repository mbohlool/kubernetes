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

package apiextensions

import (
	"k8s.io/apimachinery/pkg/util/webhook"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

var _ webhook.ClientConfigInterface = &WebhookClientConfig{}

func (c *WebhookClientConfig) GetURL() *string {
	return c.URL
}

func (c *WebhookClientConfig) GetCABundle() []byte {
	return c.CABundle
}

func (c *WebhookClientConfig) GetServiceName() *string {
	if c.Service != nil {
		return &c.Service.Name
	}
	return nil
}

func (c *WebhookClientConfig) GetServiceNamespace() *string {
	if c.Service != nil {
		return &c.Service.Namespace
	}
	return nil
}

func (c *WebhookClientConfig) GetServicePath() *string {
	if c.Service != nil {
		return c.Service.Path
	}
	return nil
}

func (c *WebhookClientConfig) GetCacheKey() (string, error) {
	cacheKey, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(cacheKey), nil
}
