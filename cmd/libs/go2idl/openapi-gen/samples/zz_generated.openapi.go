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

package samples

import (
	spec "github.com/go-openapi/spec"
	openapi "k8s.io/kubernetes/pkg/genericapiserver/openapi"
)

func (_ MapSample) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "Map sample tests openAPIGen.generateMapProperty method."
	schema.Properties = map[string]spec.Schema{
		"StringToString": {
			SchemaProps: spec.SchemaProps{
				Description: "A sample String to String map",
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
		"StringToStruct": {
			SchemaProps: spec.SchemaProps{
				Description: "A sample String to struct map",
				Type:        []string{"object"},
				AdditionalProperties: &spec.SchemaOrBool{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/samples.MapSample"),
						},
					},
				},
			},
		},
		"name": {
			SchemaProps: spec.SchemaProps{
				Description: "",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	schema.Required = []string{"StringToString", "StringToStruct", "name"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"samples.MapSample": MapSample{},
		},
	}
}

func (_ PointerSample) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "PointerSample demonstrate pointer's properties"
	schema.Properties = map[string]spec.Schema{
		"StringPointer": {
			SchemaProps: spec.SchemaProps{
				Description: "A string pointer",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"StructPointer": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/samples.SimpleSample"),
			},
		},
		"SlicePointer": {
			SchemaProps: spec.SchemaProps{
				Description: "A slice pointer",
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
		"MapPointer": {
			SchemaProps: spec.SchemaProps{
				Description: "A map pointer",
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
	}
	schema.Required = []string{"StringPointer", "StructPointer", "SlicePointer", "MapPointer"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"samples.SimpleSample": SimpleSample{},
		},
	}
}

func (_ SampleSliceProperty) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "SampleSliceProperty demonstrates properties with slice type"
	schema.Properties = map[string]spec.Schema{
		"StringSlice": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple string slice",
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
		"StructSlice": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple struct slice",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/samples.SimpleSample"),
						},
					},
				},
			},
		},
	}
	schema.Required = []string{"StringSlice", "StructSlice"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"samples.SimpleSample": SimpleSample{},
		},
	}
}

func (_ SampleStructProperty) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "SampleStructProperty demonstrates properties with struct type"
	schema.Properties = map[string]spec.Schema{
		"Struct": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/samples.SimpleSample"),
			},
		},
	}
	schema.Required = []string{"Struct"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"samples.SimpleSample": SimpleSample{},
		},
	}
}

func (_ SimpleSample) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "Sample with simple data types."
	schema.Properties = map[string]spec.Schema{
		"String": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple string",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"Int": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"Int64": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int64",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"Int32": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int32",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"Int16": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int16",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"Int8": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int8",
				Type:        []string{"integer"},
				Format:      "byte",
			},
		},
		"Uint": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"Uint64": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int64",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"Uint32": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int32",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"Uint16": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int16",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"Uint8": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple int8",
				Type:        []string{"integer"},
				Format:      "byte",
			},
		},
		"Byte": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple byte",
				Type:        []string{"integer"},
				Format:      "byte",
			},
		},
		"Bool": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple boolean",
				Type:        []string{"boolean"},
				Format:      "",
			},
		},
		"Float64": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple float64",
				Type:        []string{"number"},
				Format:      "double",
			},
		},
		"Float32": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple float32",
				Type:        []string{"number"},
				Format:      "float",
			},
		},
		"Time": {
			SchemaProps: spec.SchemaProps{
				Description: "A simple time",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"ByteArray": {
			SchemaProps: spec.SchemaProps{
				Description: "a base64 encoded characters",
				Type:        []string{"string"},
				Format:      "byte",
			},
		},
		"RObject": {
			SchemaProps: spec.SchemaProps{
				Description: "a runtime object",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"IntOrString": {
			SchemaProps: spec.SchemaProps{
				Description: "an int or string type",
				Type:        []string{"string"},
				Format:      "int-or-string",
			},
		},
	}
	schema.Required = []string{"String", "Int", "Int64", "Int32", "Int16", "Int8", "Uint", "Uint64", "Uint32", "Uint16", "Uint8", "Byte", "Bool", "Float64", "Float32", "Time", "ByteArray", "RObject", "IntOrString"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}
