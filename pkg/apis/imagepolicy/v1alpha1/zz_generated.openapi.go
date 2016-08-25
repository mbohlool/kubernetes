// +build !ignore_autogenerated

/*
Copyright 2016 The Kubernetes Authors.

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

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	openapi "k8s.io/kubernetes/pkg/genericapiserver/openapi"
)

func (_ ImageReview) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ImageReview checks if the set of images in a pod are allowed."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1alpha1.ImageReviewSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1alpha1.ImageReviewStatus"),
			},
		},
	}
	schema.Required = []string{"spec"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":              v1.ObjectMeta{},
			"v1alpha1.ImageReviewSpec":   ImageReviewSpec{},
			"v1alpha1.ImageReviewStatus": ImageReviewStatus{},
		},
	}
}

func (_ ImageReviewContainerSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ImageReviewContainerSpec is a description of a container within the pod creation request."
	schema.Properties = map[string]spec.Schema{
		"image": {
			SchemaProps: spec.SchemaProps{
				Description: "This can be in the form image:tag or image@SHA:012345679abcdef.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ ImageReviewSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ImageReviewSpec is a description of the pod creation request."
	schema.Properties = map[string]spec.Schema{
		"containers": {
			SchemaProps: spec.SchemaProps{
				Description: "Containers is a list of a subset of the information in each container of the Pod being created.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v1alpha1.ImageReviewContainerSpec"),
						},
					},
				},
			},
		},
		"annotations": {
			SchemaProps: spec.SchemaProps{
				Description: "Annotations is a list of key-value pairs extracted from the Pod's annotations. It only includes keys which match the pattern `*.image-policy.k8s.io/*`. It is up to each webhook backend to determine how to interpret these annotations, if at all.",
				Type:        []string{"object"},
				AdditionalProperties: &spec.SchemaOrBool{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
		"namespace": {
			SchemaProps: spec.SchemaProps{
				Description: "Namespace is the namespace the pod is being created in.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1alpha1.ImageReviewContainerSpec": ImageReviewContainerSpec{},
		},
	}
}

func (_ ImageReviewStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ImageReviewStatus is the result of the token authentication request."
	schema.Properties = map[string]spec.Schema{
		"allowed": {
			SchemaProps: spec.SchemaProps{
				Description: "Allowed indicates that all images were allowed to be run.",
				Type:        []string{"boolean"},
				Format:      "",
			},
		},
		"reason": {
			SchemaProps: spec.SchemaProps{
				Description: "Reason should be empty unless Allowed is false in which case it may contain a short description of what is wrong.  Kubernetes may truncate excessively long errors when displaying to the user.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	schema.Required = []string{"allowed"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}
