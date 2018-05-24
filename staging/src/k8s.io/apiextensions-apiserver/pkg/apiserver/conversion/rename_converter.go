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
)

func rename(obj runtime.Unstructured, jsonPath, from, to string, ignoreMissing bool) error {
	jpath := jsonpath.New("crd")
	jpath = jpath.AllowMissingKeys(true)
	if err := jpath.Parse(jsonPath); err != nil {
		return err
	}
	values, err := jpath.FindResults(obj.UnstructuredContent())
	if err != nil {
		return err
	}
	for _, v := range values {
		for _, v2 := range v {
			t := v2.Type()
			if t.Kind() != reflect.Map || t.Key().Kind() != reflect.String {
				return fmt.Errorf("jsonpath in rename conversion should point to a map of string")
			}

			fmt.Printf("%s, %s, %s : %s\n", t.Kind(), t.Key(), t.Elem(), v2)
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