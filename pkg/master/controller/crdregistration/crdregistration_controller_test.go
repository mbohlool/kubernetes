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

package crdregistration

import (
	"reflect"
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	crdlisters "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
)

func TestHandleVersionUpdate(t *testing.T) {
	tests := []struct {
		name         string
		startingCRDs []*apiextensions.CustomResourceDefinition
		version      schema.GroupVersion

		expectedAdded   []*apiregistration.APIService
		expectedRemoved []string
	}{
		{
			name: "simple add crd",
			startingCRDs: []*apiextensions.CustomResourceDefinition{
				{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group: "group.com",
						// Version field is deprecated and crd registration won't rely on it at all.
						// defaulting route will fill up Versions field if user only provided version field.
						Versions: []apiextensions.CustomResourceDefinitionVersion{
							{
								Name:    "v1",
								Served:  true,
								Storage: true,
							},
						},
					},
				},
			},
			version: schema.GroupVersion{Group: "group.com", Version: "v1"},

			expectedAdded: []*apiregistration.APIService{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "v1.group.com"},
					Spec: apiregistration.APIServiceSpec{
						Group:                "group.com",
						Version:              "v1",
						GroupPriorityMinimum: 1000,
						VersionPriority:      100,
					},
				},
			},
		},
		{
			name: "simple remove crd",
			startingCRDs: []*apiextensions.CustomResourceDefinition{
				{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group: "group.com",
						Versions: []apiextensions.CustomResourceDefinitionVersion{
							{
								Name:    "v1",
								Served:  true,
								Storage: true,
							},
						},
					},
				},
			},
			version: schema.GroupVersion{Group: "group.com", Version: "v2"},

			expectedRemoved: []string{"v2.group.com"},
		},
	}

	for _, test := range tests {
		registration := &fakeAPIServiceRegistration{}
		crdCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		crdLister := crdlisters.NewCustomResourceDefinitionLister(crdCache)
		c := crdRegistrationController{
			crdLister:              crdLister,
			apiServiceRegistration: registration,
		}
		for i := range test.startingCRDs {
			crdCache.Add(test.startingCRDs[i])
		}

		c.handleVersionUpdate(test.version)

		if !reflect.DeepEqual(test.expectedAdded, registration.added) {
			t.Errorf("%s expected %v, got %v", test.name, test.expectedAdded, registration.added)
		}
		if !reflect.DeepEqual(test.expectedRemoved, registration.removed) {
			t.Errorf("%s expected %v, got %v", test.name, test.expectedRemoved, registration.removed)
		}
	}
}

type fakeAPIServiceRegistration struct {
	added   []*apiregistration.APIService
	removed []string
}

func (a *fakeAPIServiceRegistration) AddAPIServiceToSync(in *apiregistration.APIService) {
	a.added = append(a.added, in)
}
func (a *fakeAPIServiceRegistration) RemoveAPIServiceToSync(name string) {
	a.removed = append(a.removed, name)
}


func testCrdFor(name string, versions ...string) *apiextensions.CustomResourceDefinition {
	ret := apiextensions.CustomResourceDefinition{
		Spec: apiextensions.CustomResourceDefinitionSpec{
			Group: "test",
		},
	}
	ret.Name = name
	for _, v := range versions {
		ret.Spec.Versions = append(ret.Spec.Versions, apiextensions.CustomResourceDefinitionVersion{Name: v})
	}
	return &ret
}

func TestComputeDeterministicVersionPriorities(t *testing.T) {
	tests := []struct {
		// List of CRD cases, first item is the name of the CRD, and the rest is versions
		crdNameAndVersions [][]string
		finalList []string
	}{
		{
			crdNameAndVersions: [][]string{
				{"A", "v1", "v2"},
				{"B", "v3", "v1"},
				{"C", "v1", "v4", "v1"},
			},
			finalList: []string{
				// v4 can be technically at the beginning or end as C cannot be satisfied.
				"v3", "v1", "v2", "v4",
			},
		},
		{
			crdNameAndVersions: [][]string{
				{"A", "v1", "v2"},
				{"B", "v2", "v1"},
			},
			finalList: []string{
				// any order is acceptable as long as it is consistent
				"v1", "v2",
			},
		},
		{
			crdNameAndVersions: [][]string{
				{"D", "v4", "v5"},
				{"A", "v1", "v5"},
				{"C", "v3", "v5"},
				{"B", "v2", "v5"},
			},
			finalList: []string{
				// Any order of v4 to v1 is acceptable
				"v4", "v3", "v2", "v1", "v5",
			},
		},
		{
			crdNameAndVersions: [][]string{
				{"A", "v2"},
				{"B", "v1", "v2"},
			},
			finalList: []string{
				"v1", "v2",
			},
		},
		{
			crdNameAndVersions: [][]string{
				{"A", "v1", "v2"},
				{"B", "v2", "v3"},
				{"C", "v3", "v4"},
			},
			finalList: []string{
				"v1", "v2", "v3", "v4",
			},
		},
	}

	for _, test := range tests {
		var crd []*apiextensions.CustomResourceDefinition
		for _, crdCase := range test.crdNameAndVersions {
			crd = append(crd, testCrdFor(crdCase[0], crdCase[1:]...))
		}
		if e, a := test.finalList, computeDeterministicVersionPriorities(crd, "test"); !reflect.DeepEqual(e, a) {
			t.Errorf("expected %v, got %v", e, a)
		}
	}
}