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

package converter

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func convertExampleCRD(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	glog.V(2).Info("converting crd")

	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, statusErrorWithMessage("conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	switch Object.GetAPIVersion() {
	case "stable.example.com/v1":
		switch toVersion {
		case "stable.example.com/v2":
			hostPort, ok := convertedObject.Object["hostPort"]
			if !ok {
				return nil, statusErrorWithMessage("expected hostPort in stable.example.com/v1")
			}
			delete(convertedObject.Object, "hostPort")
			parts := strings.Split(hostPort.(string), ":")
			convertedObject.Object["host"] = parts[0]
			convertedObject.Object["port"] = parts[1]
		default:
			return nil, statusErrorWithMessage("unexpected conversion version %s", toVersion)
		}
	case "stable.example.com/v2":
		switch toVersion {
		case "stable.example.com/v1":
			host, ok := convertedObject.Object["host"]
			if !ok {
				return nil, statusErrorWithMessage("expected host in stable.example.com/v2")
			}
			port, ok := convertedObject.Object["port"]
			if !ok {
				return nil, statusErrorWithMessage("expected port in stable.example.com/v2")
			}
			delete(convertedObject.Object, "host")
			delete(convertedObject.Object, "port")
			convertedObject.Object["hostPort"] = fmt.Sprintf("%s:%s", host, port)
		default:
			return nil, statusErrorWithMessage("unexpected conversion version %s", toVersion)
		}

	default:
		return nil, statusErrorWithMessage("unexpected conversion version %s", fromVersion)
	}

	return convertedObject, statusSucceed()
}
