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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/jsonpath"
	"fmt"
	"reflect"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

// nopConverter is a converter that only sets the apiVersion fields, but does not real conversion. It supports fields selectors.
type declarativeConverter struct {
	declarativeConversions []apiextensions.CustomResourceDeclarativeConversion
}

var _ crdConverterInterface = &declarativeConverter{}

func (c *declarativeConverter) ConvertCustomResource(obj runtime.Unstructured, target runtime.GroupVersioner) (bool, error) {
	for _, d := range c.declarativeConversions {
		switch {
		case len(d.Rename) > 0:
			for _, r := range d.Rename {
				err := rename(obj, string(r.Object), r.FieldName, r.NewFieldName, r.FailOnMissing)
				if err != nil {
					return false, err
				}
			}
		default:
			return false, fmt.Errorf("declaraitve conversion is invalid")
		}
	}
	return false, nil
}

func rename(obj runtime.Unstructured, jsonPath, from, to string, ignoreMissing bool) error {
	path := jsonpath.New("crd").AllowMissingKeys(true)
	if err := path.Parse(jsonPath); err != nil {
		return err
	}
	values, err := path.FindResults(obj.UnstructuredContent())
	if err != nil {
		return err
	}
	for _, v1 := range values {
		for _, v2 := range v1 {
			t := v2.Type()
			if t.Kind() != reflect.Map || t.Key().Kind() != reflect.String || t.Elem().Kind() != reflect.Interface {
				return fmt.Errorf("jsonpath in rename conversion should point to a map of string")
			}
			baseMap := v2.Interface().(map[string]interface{})
			if value, ok := baseMap[from]; ok {
				baseMap[to] = value
				delete(baseMap, from)
			} else if !ignoreMissing {
				return fmt.Errorf("%s path refer to a map that does not have key %s", jsonPath, from)
			}
		}
	}
	return nil
}