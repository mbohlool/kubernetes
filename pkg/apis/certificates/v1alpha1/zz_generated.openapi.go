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
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	openapi "k8s.io/kubernetes/pkg/genericapiserver/openapi"
)

func (_ CertificateSigningRequest) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "Describes a certificate signing request"
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1alpha1.CertificateSigningRequestSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1alpha1.CertificateSigningRequestStatus"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":                            v1.ObjectMeta{},
			"v1alpha1.CertificateSigningRequestSpec":   CertificateSigningRequestSpec{},
			"v1alpha1.CertificateSigningRequestStatus": CertificateSigningRequestStatus{},
		},
	}
}

func (_ CertificateSigningRequestCondition) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = ""
	schema.Properties = map[string]spec.Schema{
		"type": {
			SchemaProps: spec.SchemaProps{
				Description: "request approval state, currently Approved or Denied.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"reason": {
			SchemaProps: spec.SchemaProps{
				Description: "brief reason for the request state",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"message": {
			SchemaProps: spec.SchemaProps{
				Description: "human readable message with details about the request state",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"lastUpdateTime": {
			SchemaProps: spec.SchemaProps{
				Description: "timestamp for the last update to this condition",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
	}
	schema.Required = []string{"type"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ CertificateSigningRequestList) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = ""
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/unversioned.ListMeta"),
			},
		},
		"items": {
			SchemaProps: spec.SchemaProps{
				Description: "",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v1alpha1.CertificateSigningRequest"),
						},
					},
				},
			},
		},
	}
	schema.Required = []string{"items"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"unversioned.ListMeta":               unversioned.ListMeta{},
			"v1alpha1.CertificateSigningRequest": CertificateSigningRequest{},
		},
	}
}

func (_ CertificateSigningRequestSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "This information is immutable after the request is created. Only the Request and ExtraInfo fields can be set on creation, other fields are derived by Kubernetes and cannot be modified by users."
	schema.Properties = map[string]spec.Schema{
		"request": {
			SchemaProps: spec.SchemaProps{
				Description: "Base64-encoded PKCS#10 CSR data",
				Type:        []string{"string"},
				Format:      "byte",
			},
		},
		"username": {
			SchemaProps: spec.SchemaProps{
				Description: "Information about the requesting user (if relevant) See user.Info interface for details",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"uid": {
			SchemaProps: spec.SchemaProps{
				Description: "",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"groups": {
			SchemaProps: spec.SchemaProps{
				Description: "",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
	}
	schema.Required = []string{"request"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ CertificateSigningRequestStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = ""
	schema.Properties = map[string]spec.Schema{
		"conditions": {
			SchemaProps: spec.SchemaProps{
				Description: "Conditions applied to the request, such as approval or denial.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v1alpha1.CertificateSigningRequestCondition"),
						},
					},
				},
			},
		},
		"certificate": {
			SchemaProps: spec.SchemaProps{
				Description: "If request was approved, the controller will place the issued certificate here.",
				Type:        []string{"string"},
				Format:      "byte",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1alpha1.CertificateSigningRequestCondition": CertificateSigningRequestCondition{},
		},
	}
}
