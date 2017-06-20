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

package diff

import (
	"github.com/go-openapi/spec"
	"fmt"
)

const (
	VERBS = []string{}
)
type pathItemPair struct {
	old, new *spec.PathItem
}

type Differ struct {
	Old, New *spec.Swagger
	OnlyCompatibility bool

	commonPaths map[string]pathItemPair
	errors []error
}

func (d *Differ) Diff() []error {
	ret := []error{}
	ret = append(ret, d.diffPaths())
	ret = append(ret, d.diffDefinitions())
	return ret
}

func (d *Differ) diffPaths() []error {
	ret := []error{}
	for name, oldPath := range d.Old.Paths.Paths {
		newPath, exists := d.New.Paths.Paths[name]
		if !exists {
			d.errorf("Removed path %s", name)
			continue
		}
		d.commonPaths[name] = pathItemPair{oldPath, newPath}
	}
	if !d.OnlyCompatibility {
		for name := range d.Old.Paths.Paths {
			if _, exists := d.commonPaths[name]; !exists {
				d.errorf("Added path %s", name)
				continue
			}
		}
	}
	for name, pair := range d.commonPaths {
		d.diffParameters("common parameters of " + name, pair.old.Parameters, pair.new.Parameters)
		d.diffOperation("get " + name, pair.old.Get, pair.new.Get)
		d.diffOperation("put " + name, pair.old.Put, pair.new.Put)
		d.diffOperation("delete " + name, pair.old.Delete, pair.new.Delete)
		d.diffOperation("options " + name, pair.old.Options, pair.new.Options)
		d.diffOperation("head " + name, pair.old.Head, pair.new.Head)
		d.diffOperation("path " + name, pair.old.Patch, pair.new.Patch)
	}
	return ret

}

func (d *Differ) diffParameters(name string, params []spec.Parameter) []error {
	for _, v := range params {
		v.
	}
}

func (d *Differ) errorf(msg string, a ...interface{}) {
	d.errors = append(d.errors, fmt.Errorf(msg, a))
}

