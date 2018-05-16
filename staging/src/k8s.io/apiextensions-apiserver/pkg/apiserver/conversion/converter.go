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
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewCRDConverter returns a new CRD converter based on the conversion settings in crd object.
func NewCRDConverter(crd *apiextensions.CustomResourceDefinition) runtime.ObjectConvertor {
	validVersions := map[schema.GroupVersion]bool{}
	for _, version := range crd.Spec.Versions {
		validVersions[schema.GroupVersion{Group: crd.Spec.Group, Version: version.Name}] = true
	}

	// The only converter right now is nopConverter. More converters will be returned based on the
	// CRD object when they introduced.
	return &nopConverter{
		clusterScoped: crd.Spec.Scope == apiextensions.ClusterScoped,
		validVersions: validVersions,
	}
}
