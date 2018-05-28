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
	"testing"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"fmt"
	"k8s.io/client-go/util/jsonpath"
)

func TestRename(t *testing.T) {
//	json.Unmarshal("{\"list\": { \"")
	cr := unstructured.Unstructured{}
	cr.Object = map[string]interface{}{}
	cr.Object["list"] = []map[string]interface{} {
		{"name":"name1", "id":[]string{"id1", "id2"}},
		{"name":"name2", "id":"id2"},
	}
	cr.Object["Spec"] = map[string]string {
		"Name": "fullName",
		"User": "user1",
	}
	cr.Object["rootValue"] = "value"

	rename(&cr, "{.list[*]}", "id", "identifier", true)

	fmt.Print(cr)
}

func dumpTree(node jsonpath.Node, indent string) {
	switch (node.Type()) {
	case jsonpath.NodeArray:
		arrayNode := node.(*jsonpath.ArrayNode)
		fmt.Printf("%s%s: start=(known: %v, value: %v), end=(known: %v, value: %v), step=(known: %v, value: %v)\n",
			indent, node.Type(),
			arrayNode.Params[0].Known, arrayNode.Params[0].Value,
			arrayNode.Params[1].Known, arrayNode.Params[1].Value,
			arrayNode.Params[2].Known, arrayNode.Params[2].Value)
	case jsonpath.NodeList:
		fmt.Printf("%s%s ->\n", indent, node.String())
		for _, n := range node.(*jsonpath.ListNode).Nodes {
			dumpTree(n, indent + "  ")
		}
	case jsonpath.NodeUnion:
		fmt.Printf("%s%s ->\n", indent, node.String())
		for _, n := range node.(*jsonpath.UnionNode).Nodes {
			dumpTree(n, indent + "  ")
		}
	default:
		fmt.Printf("%s%s\n", indent, node.String())
	}
}

func TestParser(t *testing.T) {
	parser, err := jsonpath.Parse("test", "{.list[1,2,3]}{.list.*}")
	if err != nil {
		t.Fatal(err)
	}
	dumpTree(parser.Root, "")
}