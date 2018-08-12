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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/util/webhook"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

var _ webhook.ClientConfigInterface = &Webhook{}

func (w *Webhook) GetName() string {
	return w.Name
}

func (w *Webhook) GetURL() *string {
	return w.ClientConfig.URL
}

func (w *Webhook) GetCABundle() []byte {
	return w.ClientConfig.CABundle
}

func (w *Webhook) GetServiceName() *string {
	if w.ClientConfig.Service != nil {
		return &w.ClientConfig.Service.Name
	}
	return nil
}

func (w *Webhook) GetServiceNamespace() *string {
	if w.ClientConfig.Service != nil {
		return &w.ClientConfig.Service.Namespace
	}
	return nil
}

func (w *Webhook) GetServicePath() *string {
	if w.ClientConfig.Service != nil {
		return w.ClientConfig.Service.Path
	}
	return nil
}

func (w *Webhook) GetCacheKey() (string, error) {
	cacheKey, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(cacheKey), nil
}
