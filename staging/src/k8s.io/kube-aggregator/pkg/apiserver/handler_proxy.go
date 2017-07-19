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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/golang/glog"

	"github.com/go-openapi/spec"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericfeatures "k8s.io/apiserver/pkg/features"
	genericrest "k8s.io/apiserver/pkg/registry/generic/rest"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	apiregistrationapi "k8s.io/kube-aggregator/pkg/apis/apiregistration"
	"k8s.io/kube-openapi/pkg/aggregator"
)

// proxyHandler provides a http.Handler which will proxy traffic to locations
// specified by items implementing Redirector.
type proxyHandler struct {
	contextMapper genericapirequest.RequestContextMapper

	// localDelegate is used to satisfy local APIServices
	localDelegate http.Handler

	// proxyClientCert/Key are the client cert used to identify this proxy. Backing APIServices use
	// this to confirm the proxy's identity
	proxyClientCert []byte
	proxyClientKey  []byte
	proxyTransport  *http.Transport

	// Endpoints based routing to map from cluster IP to routable IP
	serviceResolver ServiceResolver

	handlingInfo atomic.Value

	groupPath string

	disableOpenAPI bool
}

type proxyHandlingInfo struct {
	// local indicates that this APIService is locally satisfied
	local bool

	// restConfig holds the information for building a roundtripper
	restConfig *restclient.Config
	// transportBuildingError is an error produced while building the transport.  If this
	// is non-nil, it will be reported to clients.
	transportBuildingError error
	// proxyRoundTripper is the re-useable portion of the transport.  It does not vary with any request.
	proxyRoundTripper http.RoundTripper
	// serviceName is the name of the service this handler proxies to
	serviceName string
	// namespace is the namespace the service lives in
	serviceNamespace string

	servicePriority int32
	// OpenAPI spec of the service, if available.
	openAPISpec *spec.Swagger
}

func max(x, y int32) int32 {
	if x > y {
		return x
	}
	return y
}

func (r *proxyHandler) rewriteRequest(req *http.Request, serviceNamespace, serviceName string) (*http.Request, error) {
	// write a new location based on the existing request pointed at the target service
	location := &url.URL{}
	location.Scheme = "https"
	rloc, err := r.serviceResolver.ResolveEndpoint(serviceNamespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("missing route (%s)", err.Error())
	}
	location.Host = rloc.Host
	location.Path = req.URL.Path
	location.RawQuery = req.URL.Query().Encode()

	// WithContext creates a shallow clone of the request with the new context.
	newReq := req.WithContext(context.Background())
	newReq.Header = utilnet.CloneHeader(req.Header)
	newReq.URL = location
	return newReq, nil
}

func (r *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	value := r.handlingInfo.Load()
	if value == nil {
		r.localDelegate.ServeHTTP(w, req)
		return
	}
	handlingInfo := value.(proxyHandlingInfo)
	if handlingInfo.local {
		if r.localDelegate == nil {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		r.localDelegate.ServeHTTP(w, req)
		return
	}

	if handlingInfo.transportBuildingError != nil {
		http.Error(w, handlingInfo.transportBuildingError.Error(), http.StatusInternalServerError)
		return
	}

	ctx, ok := r.contextMapper.Get(req)
	if !ok {
		http.Error(w, "missing context", http.StatusInternalServerError)
		return
	}
	user, ok := genericapirequest.UserFrom(ctx)
	if !ok {
		http.Error(w, "missing user", http.StatusInternalServerError)
		return
	}

	newReq, err := r.rewriteRequest(req, handlingInfo.serviceNamespace, handlingInfo.serviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	location := newReq.URL
	if handlingInfo.proxyRoundTripper == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	// we need to wrap the roundtripper in another roundtripper which will apply the front proxy headers
	proxyRoundTripper, upgrade, err := maybeWrapForConnectionUpgrades(handlingInfo.restConfig, handlingInfo.proxyRoundTripper, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	proxyRoundTripper = transport.NewAuthProxyRoundTripper(user.GetName(), user.GetGroups(), user.GetExtra(), proxyRoundTripper)

	// if we are upgrading, then the upgrade path tries to use this request with the TLS config we provide, but it does
	// NOT use the roundtripper.  Its a direct call that bypasses the round tripper.  This means that we have to
	// attach the "correct" user headers to the request ahead of time.  After the initial upgrade, we'll be back
	// at the roundtripper flow, so we only have to muck with this request, but we do have to do it.
	if upgrade {
		transport.SetAuthProxyHeaders(newReq, user.GetName(), user.GetGroups(), user.GetExtra())
	}

	handler := genericrest.NewUpgradeAwareProxyHandler(location, proxyRoundTripper, true, upgrade, &responder{w: w})
	handler.ServeHTTP(w, newReq)
}

// maybeWrapForConnectionUpgrades wraps the roundtripper for upgrades.  The bool indicates if it was wrapped
func maybeWrapForConnectionUpgrades(restConfig *restclient.Config, rt http.RoundTripper, req *http.Request) (http.RoundTripper, bool, error) {
	if !httpstream.IsUpgradeRequest(req) {
		return rt, false, nil
	}

	tlsConfig, err := restclient.TLSConfigFor(restConfig)
	if err != nil {
		return nil, true, err
	}
	followRedirects := utilfeature.DefaultFeatureGate.Enabled(genericfeatures.StreamingProxyRedirects)
	upgradeRoundTripper := spdy.NewRoundTripper(tlsConfig, followRedirects)
	wrappedRT, err := restclient.HTTPWrappersForConfig(restConfig, upgradeRoundTripper)
	if err != nil {
		return nil, true, err
	}

	return wrappedRT, true, nil
}

// responder implements rest.Responder for assisting a connector in writing objects or errors.
type responder struct {
	w http.ResponseWriter
}

// TODO this should properly handle content type negotiation
// if the caller asked for protobuf and you write JSON bad things happen.
func (r *responder) Object(statusCode int, obj runtime.Object) {
	responsewriters.WriteRawJSON(statusCode, obj, r.w)
}

func (r *responder) Error(err error) {
	http.Error(r.w, err.Error(), http.StatusInternalServerError)
}

// these methods provide locked access to fields

func (r *proxyHandler) updateAPIService(apiService *apiregistrationapi.APIService) error {
	if apiService.Spec.Service == nil {
		r.handlingInfo.Store(proxyHandlingInfo{local: true})
		return nil
	}

	newInfo := proxyHandlingInfo{
		restConfig: &restclient.Config{
			TLSClientConfig: restclient.TLSClientConfig{
				Insecure:   apiService.Spec.InsecureSkipTLSVerify,
				ServerName: apiService.Spec.Service.Name + "." + apiService.Spec.Service.Namespace + ".svc",
				CertData:   r.proxyClientCert,
				KeyData:    r.proxyClientKey,
				CAData:     apiService.Spec.CABundle,
			},
		},
		serviceName:      apiService.Spec.Service.Name,
		serviceNamespace: apiService.Spec.Service.Namespace,
		servicePriority:  max(apiService.Spec.VersionPriority, apiService.Spec.GroupPriorityMinimum),
	}
	newInfo.proxyRoundTripper, newInfo.transportBuildingError = restclient.TransportFor(newInfo.restConfig)
	if newInfo.transportBuildingError == nil && r.proxyTransport.Dial != nil {
		switch transport := newInfo.proxyRoundTripper.(type) {
		case *http.Transport:
			transport.Dial = r.proxyTransport.Dial
		default:
			newInfo.transportBuildingError = fmt.Errorf("unable to set dialer for %s/%s as rest transport is of type %T", apiService.Spec.Service.Namespace, apiService.Spec.Service.Name, newInfo.proxyRoundTripper)
			glog.Warning(newInfo.transportBuildingError.Error())
		}
	}
	if !r.disableOpenAPI {
		if err := r.updateOpenAPISpec(&newInfo); err != nil {
			return err
		}
	}
	r.handlingInfo.Store(newInfo)
	return nil
}

func (r *proxyHandler) defaultUserInfo() user.Info {
	return &user.DefaultInfo{Name: "system:aggregator"}
}

func (r *proxyHandler) updateOpenAPISpec(handlingInfo *proxyHandlingInfo) error {
	if handlingInfo.local {
		return nil
	}
	if handlingInfo.transportBuildingError != nil {
		return handlingInfo.transportBuildingError
	}
	if handlingInfo.proxyRoundTripper == nil {
		return fmt.Errorf("handlingInfo.proxyRoundTripper is nil.")
	}

	req, err := http.NewRequest("GET", "/swagger.json", nil)
	if err != nil {
		return err
	}
	req, err = r.rewriteRequest(req, handlingInfo.serviceNamespace, handlingInfo.serviceName)
	if err != nil {
		return err
	}

	user := r.defaultUserInfo()
	roundTripper := transport.NewAuthProxyRoundTripper(user.GetName(), user.GetGroups(), user.GetExtra(), handlingInfo.proxyRoundTripper)
	resp, err := roundTripper.RoundTrip(req)
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to retrive openAPI spec, http error code %d", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	bytes := buf.Bytes()

	openAPISpec := &spec.Swagger{}
	if err := json.Unmarshal(bytes, &openAPISpec); err != nil {
		return err
	}
	aggregator.FilterSpecByPaths(openAPISpec, []string{r.groupPath + "/"})
	handlingInfo.openAPISpec = openAPISpec
	return nil
}

func (r *proxyHandler) OpenAPISpec() (*spec.Swagger, int32) {
	value := r.handlingInfo.Load()
	if value == nil {
		return nil, 0
	}
	handlingInfo := value.(proxyHandlingInfo)
	return handlingInfo.openAPISpec, handlingInfo.servicePriority
}
