/*
Copyright 2018 The Kubernetes Authors.

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

package conversion

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/webhook"

	internal "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type webhookConverterFactory struct {
	clientManager webhook.ClientManager
}

func newWebhookConverterFactory(serviceResolver webhook.ServiceResolver, authResolverWrapper webhook.AuthenticationInfoResolverWrapper) (*webhookConverterFactory, error) {
	clientManager, err := webhook.NewClientManager(v1beta1.SchemeGroupVersion, v1beta1.AddToScheme)
	if err != nil {
		return nil, err
	}
	authInfoResolver, err := webhook.NewDefaultAuthenticationInfoResolver("")
	if err != nil {
		return nil, err
	}
	// Set defaults which may be overridden later.
	clientManager.SetAuthenticationInfoResolver(authInfoResolver)
	clientManager.SetAuthenticationInfoResolverWrapper(authResolverWrapper)
	clientManager.SetServiceResolver(serviceResolver)
	return &webhookConverterFactory{clientManager}, nil
}

// webhookConverter is a converter that only sets the apiVersion fields, but does not real conversion.
type webhookConverter struct {
	validVersions map[schema.GroupVersion]bool
	clientManager webhook.ClientManager
	restClient    *rest.RESTClient
	name          string
}

var _ runtime.ObjectConvertor = &webhookConverter{}

func (f *webhookConverterFactory) NewWebhookConverter(validVersions map[schema.GroupVersion]bool, crd *internal.CustomResourceDefinition) (*webhookConverter, error) {
	restClient, err := f.clientManager.HookClient(
		fmt.Sprintf("conversion_webhook_for_%s", crd.Name),
		&crd.Spec.Conversion.Webhook.ClientConfig)
	if err != nil {
		return nil, err
	}
	return &webhookConverter{
		clientManager: f.clientManager,
		validVersions: validVersions,
		restClient:    restClient,
		name:          crd.Name,
	}, nil
}

func (webhookConverter) ConvertFieldLabel(gvk schema.GroupVersionKind, label, value string) (string, string, error) {
	return "", "", errors.New("unstructured cannot convert field labels")
}

func (c *webhookConverter) Convert(in, out, context interface{}) error {
	unstructIn, ok := in.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("input type %T in not valid for unstructured conversion", in)
	}

	unstructOut, ok := out.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("output type %T in not valid for unstructured conversion", out)
	}

	outGVK := unstructOut.GroupVersionKind()
	if !c.validVersions[outGVK.GroupVersion()] {
		return fmt.Errorf("request to convert CR from an invalid group/version: %s", outGVK.String())
	}
	inGVK := unstructIn.GroupVersionKind()
	if !c.validVersions[inGVK.GroupVersion()] {
		return fmt.Errorf("request to convert CR to an invalid group/version: %s", inGVK.String())
	}

	unstructOut.SetUnstructuredContent(unstructIn.UnstructuredContent())
	_, err := c.ConvertToVersion(unstructOut, outGVK.GroupVersion())
	if err != nil {
		return err
	}
	return nil
}

func createConversionReview(obj runtime.Object, apiVersion string) *v1beta1.ConversionReview {
	_, isList := obj.(*unstructured.UnstructuredList)
	return &v1beta1.ConversionReview{
		Request: &v1beta1.ConversionRequest{
			Object:     runtime.RawExtension{Object: obj},
			APIVersion: apiVersion,
			UID:        uuid.NewUUID(),
			IsList:     isList,
		},
		Response: &v1beta1.ConversionResponse{},
	}
}

func (c *webhookConverter) ConvertToVersion(in runtime.Object, target runtime.GroupVersioner) (runtime.Object, error) {
	fromGVK := in.GetObjectKind().GroupVersionKind()
	toGVK, ok := target.KindForGroupVersionKinds([]schema.GroupVersionKind{fromGVK})
	if !ok {
		// TODO: should this be a typed error?
		return nil, fmt.Errorf("%v is unstructured and is not suitable for converting to %q", fromGVK.String(), target)
	}
	if !c.validVersions[toGVK.GroupVersion()] {
		return nil, fmt.Errorf("request to convert CR to an invalid group/version: %s", toGVK.String())
	}
	if !c.validVersions[fromGVK.GroupVersion()] {
		return nil, fmt.Errorf("request to convert CR from an invalid group/version: %s", fromGVK.String())
	}
	if fromGVK == toGVK {
		// No conversion is required
		return in, nil
	}

	request := createConversionReview(in, toGVK.GroupVersion().String())
	response := &v1beta1.ConversionReview{}
	ctx := context.TODO()
	if err := c.restClient.Post().Context(ctx).Body(request).Do().Into(response); err != nil {
		// TODO: Return a webhook specific error to be able to convert it to meta.Status
		return nil, fmt.Errorf("calling to conversion webhook failed for %s: %v", c.name, err)
	}

	if response.Response == nil {
		// TODO: Return a webhook specific error to be able to convert it to meta.Status
		return nil, fmt.Errorf("conversion webhook response was absent for %s", c.name)
	}

	if response.Response.Result.Status != v1.StatusSuccess {
		// TODO return status message as error
		return nil, fmt.Errorf("conversion request failed for %v: %v, %v, %v", in.GetObjectKind(), response, response.Response, response.Response.Result)
	}

	ret := response.Response.ConvertedObject.Object
	if ret == nil {
		unstruct := unstructured.Unstructured{}
		err := unstruct.UnmarshalJSON(response.Response.ConvertedObject.Raw)
		if err != nil {
			return nil, err
		}
		ret = &unstruct
	}

	convertedGVK := ret.GetObjectKind().GroupVersionKind()
	if convertedGVK != toGVK {
		return nil, fmt.Errorf("invalid GVK returned by conversion webhook. expected=%s, actual=%s", toGVK, convertedGVK)
	}
	// TODO: Look like failing on convert is considered uncommon and we even have a test flag for it: KUBE_PANIC_WATCH_DECODE_ERROR
	// Check on that as this conversion may very well return error.
	// in production, a conversion error may result in watcher stop working.
	return ret, nil
}
