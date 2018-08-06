package conversion

import (
	"errors"
	"fmt"

	"context"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/config"
	"k8s.io/client-go/rest"
)

type webhookConverterFactory struct {
	clientManager config.ClientManager
}

func newWebhookConverterFactory() (*webhookConverterFactory, error) {
	clientManager, err := config.NewClientManager()
	if err != nil {
		return nil, err
	}
	return &webhookConverterFactory{clientManager}, nil
}

// nopConverter is a converter that only sets the apiVersion fields, but does not real conversion.
type webhookConverter struct {
	validVersions map[schema.GroupVersion]bool
	clientManager config.ClientManager
	restClient    *rest.RESTClient
	name          string
}

var _ runtime.ObjectConvertor = &webhookConverter{}

func (f *webhookConverterFactory) NewWebhookConverter(validVersions map[schema.GroupVersion]bool, crd *apiextensions.CustomResourceDefinition) (*webhookConverter, error) {
	v1beta1Webhook := &v1beta1.CustomResourceConversionWebhook{}
	err := v1beta1.Convert_apiextensions_CustomResourceConversionWebhook_To_v1beta1_CustomResourceConversionWebhook(crd.Spec.Conversion.Webhook, v1beta1Webhook, nil)
	if err != nil {
		return nil, err
	}
	restClient, err := f.clientManager.HookClient("conversion_webhook_for_"+crd.Name, &v1beta1Webhook.ClientConfig)
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
		return fmt.Errorf("request to convert CRD from an invalid group/version: %s", outGVK.String())
	}
	inGVK := unstructIn.GroupVersionKind()
	if !c.validVersions[inGVK.GroupVersion()] {
		return fmt.Errorf("request to convert CRD to an invalid group/version: %s", inGVK.String())
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
			Object: runtime.RawExtension{
				Object: obj,
			},
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
		return nil, fmt.Errorf("%v is unstructured and is not suitable for converting to %q", fromGVK, target)
	}
	if !c.validVersions[toGVK.GroupVersion()] {
		return nil, fmt.Errorf("request to convert CRD to an invalid group/version: %s", toGVK.String())
	}
	if !c.validVersions[fromGVK.GroupVersion()] {
		return nil, fmt.Errorf("request to convert CRD from an invalid group/version: %s", fromGVK.String())
	}
	if fromGVK == toGVK {
		// No conversion is required
		return in, nil
	}

	request := createConversionReview(in, toGVK.GroupVersion().String())
	response := &v1beta1.ConversionReview{}
	ctx := context.TODO()
	if err := c.restClient.Post().Context(ctx).Body(&request).Do().Into(response); err != nil {
		// TODO: Return a webhook specific error to be able to convert it to meta.Status
		return nil, fmt.Errorf("calling to conversion webhook failed for %s, with error=%s", c.name, err)
	}

	if response.Response == nil {
		// TODO: Return a webhook specific error to be able to convert it to meta.Status
		return nil, fmt.Errorf("conversion webhook response was absent for %s", c.name)
	}

	ret := response.Response.ConvertedObject.Object
	convertedGVK := ret.GetObjectKind().GroupVersionKind()
	if convertedGVK != toGVK {
		return nil, fmt.Errorf("invalid GVK returned by conversion webhook. expected=%s, actual=%s", toGVK, convertedGVK)
	}

	return response.Response.ConvertedObject.Object, nil
}
