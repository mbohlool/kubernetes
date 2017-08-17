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

package apiserver

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"

	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/kube-aggregator/pkg/apis/apiregistration"
)

func newApiServiceForTest(name, group string, minGroupPriority, versionPriority int32) apiregistration.APIService {
	r := apiregistration.APIService{}
	r.Spec.Group = group
	r.Spec.GroupPriorityMinimum = minGroupPriority
	r.Spec.VersionPriority = versionPriority
	r.Name = name
	return r
}

func assertSortedServices(t *testing.T, actual []openAPISpecInfo, expectedNames []string) {
	actualNames := []string{}
	for _, a := range actual {
		actualNames = append(actualNames, a.apiService.Name)
	}
	if !reflect.DeepEqual(actualNames, expectedNames) {
		t.Errorf("Expected %s got %s.", expectedNames, actualNames)
	}
}

func TestApiServiceSort(t *testing.T) {
	list := []openAPISpecInfo{
		{
			apiService: newApiServiceForTest("FirstService", "Group1", 10, 5),
			spec:       &spec.Swagger{},
		},
		{
			apiService: newApiServiceForTest("SecondService", "Group2", 15, 3),
			spec:       &spec.Swagger{},
		},
		{
			apiService: newApiServiceForTest("FirstServiceInternal", "Group1", 16, 3),
			spec:       &spec.Swagger{},
		},
		{
			apiService: newApiServiceForTest("ThirdService", "Group3", 15, 3),
			spec:       &spec.Swagger{},
		},
	}
	sortByPriority(list)
	assertSortedServices(t, list, []string{"FirstService", "FirstServiceInternal", "SecondService", "ThirdService"})
}

type handlerTest struct {
	etag string
	data []byte
}

var _ http.Handler = handlerTest{}

func (h handlerTest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.etag) > 0 {
		w.Header().Add("Etag", h.etag)
	}
	ifNoneMatches := r.Header["If-None-Match"]
	for _, match := range ifNoneMatches {
		if match == h.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Write(h.data)
}

func assertDownloadedSpec(t *testing.T, actualSpec *spec.Swagger, actualEtag string, err error,
	expectedSpecId string, expectedEtag string) {
	if err != nil {
		t.Errorf("downloadOpenAPISpec failed : %s", err)
	}
	if expectedSpecId == "" && actualSpec != nil {
		t.Errorf("expected Not Modified, actual ID %s", actualSpec.ID)
	}
	if actualSpec != nil && actualSpec.ID != expectedSpecId {
		t.Errorf("expected ID %s, actual ID %s", expectedSpecId, actualSpec.ID)
	}
	if actualEtag != expectedEtag {
		t.Errorf("expected ETag '%s', actual ETag '%s'", expectedEtag, actualEtag)
	}
}

func TestDownloadOpenAPISpec(t *testing.T) {

	s := openAPIAggregator{contextMapper: request.NewRequestContextMapper()}

	// Test with no eTag
	actualSpec, actualEtag, _, err := s.downloadOpenAPISpec(handlerTest{data: []byte("{\"id\": \"test\"}")}, "")
	assertDownloadedSpec(t, actualSpec, actualEtag, err, "test", "\"356ECAB19D7FBE1336BABB1E70F8F3025050DE218BE78256BE81620681CFC9A268508E542B8B55974E17B2184BBFC8FFFAA577E51BE195D32B3CA2547818ABE4\"")

	// Test with eTag
	actualSpec, actualEtag, _, err = s.downloadOpenAPISpec(
		handlerTest{data: []byte("{\"id\": \"test\"}"), etag: "etag_test"}, "")
	assertDownloadedSpec(t, actualSpec, actualEtag, err, "test", "etag_test")

	// Test not modified
	actualSpec, actualEtag, _, err = s.downloadOpenAPISpec(
		handlerTest{data: []byte("{\"id\": \"test\"}"), etag: "etag_test"}, "etag_test")
	assertDownloadedSpec(t, actualSpec, actualEtag, err, "", "etag_test")

	// Test different eTags
	actualSpec, actualEtag, _, err = s.downloadOpenAPISpec(
		handlerTest{data: []byte("{\"id\": \"test\"}"), etag: "etag_test1"}, "etag_test2")
	assertDownloadedSpec(t, actualSpec, actualEtag, err, "test", "etag_test1")
}
