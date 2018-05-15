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
	"fmt"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nopConverter is a converter that only sets the apiVersion fields, but does not real conversion. It supports fields selectors.
type nopConverter struct {
	clusterScoped bool
	crd           *apiextensions.CustomResourceDefinition
}

var _ runtime.ObjectConvertor = &nopConverter{}

func (c *nopConverter) ConvertFieldLabel(version, kind, label, value string) (string, string, error) {
	// We currently only support metadata.namespace and metadata.name.
	switch {
	case label == "metadata.name":
		return label, value, nil
	case !c.clusterScoped && label == "metadata.namespace":
		return label, value, nil
	default:
		return "", "", fmt.Errorf("field label not supported: %s", label)
	}
}

func (c *nopConverter) Convert(in, out, context interface{}) error {
	unstructIn, ok := in.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("input type %T in not valid for unstructured conversion", in)
	}

	unstructOut, ok := out.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("output type %T in not valid for unstructured conversion", out)
	}

	unstructOut.SetUnstructuredContent(unstructIn.UnstructuredContent())

	_, err := c.ConvertToVersion(unstructOut, unstructOut.GroupVersionKind().GroupVersion())
	if err != nil {
		return err
	}
	return nil
}

func (c *nopConverter) convertToVersion(in runtime.Object, target runtime.GroupVersioner) error {
	kind := in.GetObjectKind().GroupVersionKind()
	gvk, ok := target.KindForGroupVersionKinds([]schema.GroupVersionKind{kind})
	if !ok {
		// TODO: should this be a typed error?
		return fmt.Errorf("%v is unstructured and is not suitable for converting to %q", kind, target)
	}
	if !apiextensions.HasCRDVersion(c.crd, gvk.Version) {
		return fmt.Errorf("request to convert CRD to an invalid version: %s", gvk.String())
	}
	in.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// ConvertToVersion converts in object to the given gvk in place and returns the same `in` object.
func (c *nopConverter) ConvertToVersion(in runtime.Object, target runtime.GroupVersioner) (runtime.Object, error) {
	var err error
	// Run the converter on the list items instead of list itself
	if meta.IsListType(in) {
		err = meta.EachListItem(in, func(item runtime.Object) error {
			return c.convertToVersion(item, target)
		})
	} else {
		err = c.convertToVersion(in, target)
	}
	return in, err
}
