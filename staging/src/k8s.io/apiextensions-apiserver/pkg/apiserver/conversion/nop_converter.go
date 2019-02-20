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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// nopConverter is a converter that only sets the apiVersion fields, but does not real conversion.
type nopConverter struct {
}

var _ crConverterInterface = &nopConverter{}

// ConvertToVersion converts in object to the given gvk in place and returns the same `in` object.
func (c *nopConverter) ConvertToVersion(in runtime.Object, target runtime.GroupVersioner) (runtime.Object, error) {
	// Run the converter on the list items instead of list itself
	if list, ok := in.(*unstructured.UnstructuredList); ok {
		for i := range list.Items {
			obj, err := c.convertToVersion(&list.Items[i], target)
			if err != nil {
				return nil, err
			}

			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return nil, fmt.Errorf("output type %T in not valid for unstructured conversion", obj)
			}
			list.Items[i] = *u
		}
		return list, nil
	}

	return c.convertToVersion(in, target)
}

func (c *nopConverter) convertToVersion(in runtime.Object, target runtime.GroupVersioner) (runtime.Object, error) {
	kind := in.GetObjectKind().GroupVersionKind()
	gvk, ok := target.KindForGroupVersionKinds([]schema.GroupVersionKind{kind})
	if !ok {
		// TODO: should this be a typed error?
		return nil, fmt.Errorf("%v is unstructured and is not suitable for converting to %q", kind, target)
	}
	in.GetObjectKind().SetGroupVersionKind(gvk)
	return in, nil
}
