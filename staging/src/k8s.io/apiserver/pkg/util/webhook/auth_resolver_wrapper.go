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

package webhook

import (
	"net/http"

	restclient "k8s.io/client-go/rest"
)

// CreateWebhookAuthResolverWrapper creates a auth wrapper using the loopback and proxyTransport configs.
func CreateWebhookAuthResolverWrapper(LoopbackClientConfig *restclient.Config, proxyTransport *http.Transport) AuthenticationInfoResolverWrapper {
	return func(delegate AuthenticationInfoResolver) AuthenticationInfoResolver {
		return &AuthenticationInfoResolverDelegator{
			ClientConfigForFunc: func(server string) (*restclient.Config, error) {
				if server == "kubernetes.default.svc" {
					return LoopbackClientConfig, nil
				}
				return delegate.ClientConfigFor(server)
			},
			ClientConfigForServiceFunc: func(serviceName, serviceNamespace string) (*restclient.Config, error) {
				if serviceName == "kubernetes" && serviceNamespace == "default" {
					return LoopbackClientConfig, nil
				}
				ret, err := delegate.ClientConfigForService(serviceName, serviceNamespace)
				if err != nil {
					return nil, err
				}
				if proxyTransport != nil && proxyTransport.DialContext != nil {
					ret.Dial = proxyTransport.DialContext
				}
				return ret, err
			},
		}
	}
}
