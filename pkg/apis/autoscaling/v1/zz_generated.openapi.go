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

package v1

import (
	spec "github.com/go-openapi/spec"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	api_v1 "k8s.io/kubernetes/pkg/api/v1"
	openapi "k8s.io/kubernetes/pkg/genericapiserver/openapi"
)

func (_ CrossVersionObjectReference) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "CrossVersionObjectReference contains enough information to let you identify the referred resource."
	schema.Properties = map[string]spec.Schema{
		"kind": {
			SchemaProps: spec.SchemaProps{
				Description: "Kind of the referent; More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds\"",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"name": {
			SchemaProps: spec.SchemaProps{
				Description: "Name of the referent; More info: http://releases.k8s.io/HEAD/docs/user-guide/identifiers.md#names",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"apiVersion": {
			SchemaProps: spec.SchemaProps{
				Description: "API version of the referent",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	schema.Required = []string{"kind", "name"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ HorizontalPodAutoscaler) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "configuration of a horizontal pod autoscaler."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.HorizontalPodAutoscalerSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.HorizontalPodAutoscalerStatus"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.HorizontalPodAutoscalerSpec":   HorizontalPodAutoscalerSpec{},
			"v1.HorizontalPodAutoscalerStatus": HorizontalPodAutoscalerStatus{},
			"v1.ObjectMeta":                    api_v1.ObjectMeta{},
		},
	}
}

func (_ HorizontalPodAutoscalerList) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "list of horizontal pod autoscaler objects."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/unversioned.ListMeta"),
			},
		},
		"items": {
			SchemaProps: spec.SchemaProps{
				Description: "list of horizontal pod autoscaler objects.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v1.HorizontalPodAutoscaler"),
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
			"unversioned.ListMeta":       unversioned.ListMeta{},
			"v1.HorizontalPodAutoscaler": HorizontalPodAutoscaler{},
		},
	}
}

func (_ HorizontalPodAutoscalerSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "specification of a horizontal pod autoscaler."
	schema.Properties = map[string]spec.Schema{
		"scaleTargetRef": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.CrossVersionObjectReference"),
			},
		},
		"minReplicas": {
			SchemaProps: spec.SchemaProps{
				Description: "lower limit for the number of pods that can be set by the autoscaler, default 1.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"maxReplicas": {
			SchemaProps: spec.SchemaProps{
				Description: "upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"targetCPUUtilizationPercentage": {
			SchemaProps: spec.SchemaProps{
				Description: "target average CPU utilization (represented as a percentage of requested CPU) over all the pods; if not specified the default autoscaling policy will be used.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
	}
	schema.Required = []string{"scaleTargetRef", "maxReplicas"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.CrossVersionObjectReference": CrossVersionObjectReference{},
		},
	}
}

func (_ HorizontalPodAutoscalerStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "current status of a horizontal pod autoscaler"
	schema.Properties = map[string]spec.Schema{
		"observedGeneration": {
			SchemaProps: spec.SchemaProps{
				Description: "most recent generation observed by this autoscaler.",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"lastScaleTime": {
			SchemaProps: spec.SchemaProps{
				Description: "last time the HorizontalPodAutoscaler scaled the number of pods; used by the autoscaler to control how often the number of pods is changed.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"currentReplicas": {
			SchemaProps: spec.SchemaProps{
				Description: "current number of replicas of pods managed by this autoscaler.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"desiredReplicas": {
			SchemaProps: spec.SchemaProps{
				Description: "desired number of replicas of pods managed by this autoscaler.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"currentCPUUtilizationPercentage": {
			SchemaProps: spec.SchemaProps{
				Description: "current average CPU utilization over all pods, represented as a percentage of requested CPU, e.g. 70 means that an average pod is using now 70% of its requested CPU.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
	}
	schema.Required = []string{"currentReplicas", "desiredReplicas"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ Scale) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "Scale represents a scaling request for a resource."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ScaleSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ScaleStatus"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":  api_v1.ObjectMeta{},
			"v1.ScaleSpec":   ScaleSpec{},
			"v1.ScaleStatus": ScaleStatus{},
		},
	}
}

func (_ ScaleSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScaleSpec describes the attributes of a scale subresource."
	schema.Properties = map[string]spec.Schema{
		"replicas": {
			SchemaProps: spec.SchemaProps{
				Description: "desired number of instances for the scaled object.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ ScaleStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScaleStatus represents the current status of a scale subresource."
	schema.Properties = map[string]spec.Schema{
		"replicas": {
			SchemaProps: spec.SchemaProps{
				Description: "actual number of observed instances of the scaled object.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"selector": {
			SchemaProps: spec.SchemaProps{
				Description: "label query over pods that should match the replicas count. This is same as the label selector but in the string format to avoid introspection by clients. The string will be in the same format as the query-param syntax. More info about label selectors: http://releases.k8s.io/HEAD/docs/user-guide/labels.md#label-selectors",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	schema.Required = []string{"replicas"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}
