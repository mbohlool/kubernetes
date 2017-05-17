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

package openapi


import (
	"encoding/json"
	"gopkg.in/yaml.v2"

	"github.com/go-openapi/spec"
	"github.com/googleapis/gnostic/OpenAPIv2"
	"github.com/googleapis/gnostic/compiler"
	"strconv"
	"reflect"
)

// What is spec.Swagger.ID?

func fromProto(doc openapi_v2.Document) spec.Swagger {
	pc := protoConverter{}
	return pc.Convert(doc)
}

func toProto(openapi spec.Swagger) (*openapi_v2.Document, error) {
	bytes, err := json.Marshal(openapi)
	if err != nil {
		return nil, err
	}
	return toProtoFromBytes(bytes)
}

func toProtoFromBytes(bytes []byte) (*openapi_v2.Document, error) {
	var info yaml.MapSlice
	if err := yaml.Unmarshal(bytes, &info); err != nil {
		return nil, err
	}
	document, err := openapi_v2.NewDocument(info, compiler.NewContext("$root", nil))
	if err != nil {
		return nil, err
	}
	return document, nil
}

type protoConverter struct {}

func (c *protoConverter) Convert(in openapi_v2.Document) spec.Swagger {
	return spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: in.Swagger,
			Info: c.info(in.Info),
			Host: in.Host,
			BasePath: in.BasePath,
			Schemes: in.Schemes,
			Consumes: in.Consumes,
			Produces: in.Produces,
			Paths: c.paths(in.Paths),
			Definitions: c.definitions(in.Definitions),
			Parameters: c.rootParameters(in.Parameters),
			Responses: c.rootResponses(in.Responses),
			Security: c.security(in.Security),
			SecurityDefinitions: c.securityDefinitions(in.SecurityDefinitions),
			Tags: c.tags(in.Tags),
			ExternalDocs: c.externalDocs(in.ExternalDocs),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) definitions(in *openapi_v2.Definitions) spec.Definitions {
	if in == nil {
		return nil
	}
	ret := spec.Definitions{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = *c.schema(i.Value)
	}
	return ret
}

func (c *protoConverter) securityDefinitions(in *openapi_v2.SecurityDefinitions) spec.SecurityDefinitions {
	if in == nil {
		return nil
	}
	ret := spec.SecurityDefinitions{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = c.securityDefinition(i.Value)
	}
	return ret
}

func (c *protoConverter) securityDefinition(in *openapi_v2.SecurityDefinitionsItem) *spec.SecurityScheme {
	if i := in.GetApiKeySecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				In: i.In,
				Name: i.Name,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	if i := in.GetBasicAuthenticationSecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	if i := in.GetOauth2AccessCodeSecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				Flow: i.Flow,
				Scopes: c.scopes(i.Scopes),
				AuthorizationURL: i.AuthorizationUrl,
				TokenURL: i.TokenUrl,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	if i := in.GetOauth2ImplicitSecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				Flow: i.Flow,
				Scopes: c.scopes(i.Scopes),
				AuthorizationURL: i.AuthorizationUrl,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	if i := in.GetOauth2ApplicationSecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				Flow: i.Flow,
				Scopes: c.scopes(i.Scopes),
				TokenURL: i.TokenUrl,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	// Todo(mehdy): this look exactly the same as one of others
	if i := in.GetOauth2PasswordSecurity(); i != nil {
		return &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type: i.Type,
				Flow: i.Flow,
				Scopes: c.scopes(i.Scopes),
				TokenURL: i.TokenUrl,
				Description: i.Description,
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(i.VendorExtension),
			},
		}
	}
	return nil
}

func (c *protoConverter) scopes(in *openapi_v2.Oauth2Scopes) map[string]string {
	if in == nil {
		return nil
	}
	ret := map[string]string{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = i.Value
	}
	return ret
}

func (c *protoConverter) tags(in []*openapi_v2.Tag) []spec.Tag {
	if in  == nil {
		return nil
	}
	ret := []spec.Tag{}
	for _, i := range in {
		ret = append(ret, c.tag(i))
	}
	return ret
}

func (c *protoConverter) tag(in *openapi_v2.Tag) spec.Tag {
	return spec.Tag{
		TagProps: spec.TagProps{
			Name: in.Name,
			Description: in.Description,
			ExternalDocs: c.externalDocs(in.ExternalDocs),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) paths(in *openapi_v2.Paths) *spec.Paths {
	ret := spec.Paths {
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
	for _, p := range in.Path {
		if ret.Paths == nil {
			ret.Paths = map[string]spec.PathItem{}
		}
		ret.Paths[p.Name] = c.pathItem(p.Value)
	}
	return &ret
}

func (c *protoConverter) pathItem(in *openapi_v2.PathItem) spec.PathItem {
	return spec.PathItem{
		Refable: c.refable(in.XRef),
		PathItemProps: spec.PathItemProps{
			Get: c.operation(in.Get),
			Put: c.operation(in.Put),
			Post: c.operation(in.Post),
			Delete: c.operation(in.Delete),
			Options: c.operation(in.Options),
			Head: c.operation(in.Head),
			Patch: c.operation(in.Patch),
			Parameters: c.parameters(in.Parameters),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) operation(in *openapi_v2.Operation) *spec.Operation {
	if in == nil {
		return nil
	}
	return &spec.Operation{
		OperationProps: spec.OperationProps{
			Description: in.Description,
			Consumes: in.Consumes,
			Produces: in.Produces,
			Schemes: in.Schemes,
			Tags: in.Tags,
			Summary: in.Summary,
			ExternalDocs: c.externalDocs(in.ExternalDocs),
			ID: in.OperationId,
			Deprecated: in.Deprecated,
			Security: c.security(in.Security),
			Parameters: c.parameters(in.Parameters),
			Responses: c.responses(in.Responses),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) responses(in *openapi_v2.Responses) *spec.Responses {
	if in == nil {
		return nil
	}
	ret := spec.Responses{
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
	for _, i := range in.ResponseCode {
		if i.Name == "default" {
			ret.Default = c.responseValue(i.Value)
		} else {
			if ret.StatusCodeResponses == nil {
				ret.StatusCodeResponses = map[int]spec.Response{}
			}
			n, _ := strconv.Atoi(i.Name)
			ret.StatusCodeResponses[n] = *c.responseValue(i.Value)
		}
	}
	return &ret
}

func (c *protoConverter) rootResponses(in *openapi_v2.ResponseDefinitions)  map[string]spec.Response {
	if in == nil {
		return nil
	}
	ret := map[string]spec.Response{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = *c.response(i.Value)
	}
	return ret
}

func (c *protoConverter) response(in *openapi_v2.Response) *spec.Response {
	// Todo(mehdy): OpenAPI spec does not let response extend? but proto does?
	return &spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: in.Description,
			Schema: c.schemaItem(in.Schema), // TODO(mehdy): why SchemaItem is not the same as Schema
			Headers: c.headers(in.Headers),
			Examples: c.examples(in.Examples),
		},
	}
}

func (c *protoConverter) responseValue(in *openapi_v2.ResponseValue) *spec.Response {
	if in == nil {
		return nil
	}
	if ref := in.GetJsonReference(); ref != nil {
		return &spec.Response{
			Refable: c.refable(ref.XRef),
			ResponseProps: spec.ResponseProps{
				Description: ref.Description,
			},
		}
	}
	if resp := in.GetResponse(); resp != nil {
		return c.response(resp)
	}
	return nil
}

func (c *protoConverter) examples(in *openapi_v2.Examples) map[string]interface{} {
	if in == nil {
		return nil
	}
	return c.anyMap(in.AdditionalProperties)
}

func (c *protoConverter) headers(in *openapi_v2.Headers) map[string]spec.Header {
	if in == nil {
		return nil
	}
	ret := map[string]spec.Header{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = c.header(i.Value)
	}
	return ret
}

func (c *protoConverter) header(in *openapi_v2.Header) spec.Header {
	// Todo(mehdy): OpenAPI header does not seem to be extensible by proto has Extensible variable
	return spec.Header{
		CommonValidations: c.commonValidationFix(spec.CommonValidations{
			Maximum: &in.Maximum,
			ExclusiveMaximum: in.ExclusiveMaximum,
			Minimum: &in.Minimum,
			ExclusiveMinimum: in.ExclusiveMinimum,
			MaxLength: &in.MaxLength,
			MinLength: &in.MinLength,
			Pattern: in.Pattern,
			MaxItems: &in.MaxItems,
			MinItems: &in.MinItems,
			UniqueItems: in.UniqueItems,
			MultipleOf: &in.MultipleOf,
			Enum: c.enum(in.Enum),
		}),
		SimpleSchema: spec.SimpleSchema {
			Type: in.Type,
			Format: in.Format,
			Items: c.item(in.Items),
			CollectionFormat: in.CollectionFormat,
			Default: c.any(in.Default),
		},
		HeaderProps: spec.HeaderProps{
			Description: in.Description,
		},
	}

}

func (c *protoConverter) commonValidationFix(v spec.CommonValidations) spec.CommonValidations {
	if *v.Maximum == 0.0 {
		v.Maximum = nil
	}
	if *v.Minimum == 0.0 {
		v.Minimum = nil
	}
	if *v.MaxLength == 0 {
		v.MaxLength = nil
	}
	if *v.MinLength == 0 {
		v.MinLength = nil
	}
	if *v.MaxItems == 0 {
		v.MaxItems = nil
	}
	if *v.MinItems == 0 {
		v.MinItems = nil
	}
	if *v.MultipleOf == 0 {
		v.MultipleOf = nil
	}
	return v
}

func (c *protoConverter) schemaPropFix(v spec.SchemaProps) spec.SchemaProps {
	if *v.Maximum == 0.0 {
		v.Maximum = nil
	}
	if *v.Minimum == 0.0 {
		v.Minimum = nil
	}
	if *v.MaxLength == 0 {
		v.MaxLength = nil
	}
	if *v.MinLength == 0 {
		v.MinLength = nil
	}
	if *v.MaxItems == 0 {
		v.MaxItems = nil
	}
	if *v.MinItems == 0 {
		v.MinItems = nil
	}
	if *v.MultipleOf == 0 {
		v.MultipleOf = nil
	}
	if *v.MaxProperties == 0 {
		v.MaxProperties = nil
	}
	if *v.MinProperties == 0 {
		v.MinProperties = nil
	}
	return v
}

func (c *protoConverter) schemaItem(in *openapi_v2.SchemaItem) *spec.Schema {
	if in == nil {
		return nil
	}
	if s := in.GetSchema(); s != nil {
		return c.schema(s)
	}
	if f := in.GetFileSchema(); f != nil {
		return &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Format: f.Format,
				Title: f.Title,
				Description: f.Description,
				Default: c.any(f.Default),
				Required: f.Required,
				Type: []string{f.Type},
			},
			SwaggerSchemaProps: spec.SwaggerSchemaProps{
				ReadOnly: f.ReadOnly,
				ExternalDocs: c.externalDocs(f.ExternalDocs),
				Example: c.any(f.Example),
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(f.VendorExtension),
			},
		}
	}
	return nil
}

func (c *protoConverter) security(in []*openapi_v2.SecurityRequirement) []map[string][]string {
	if in == nil {
		return nil
	}
	ret := []map[string][]string{}
	for _, i := range in {
		m := map[string][]string{}
		for _, j := range i.AdditionalProperties {
			m[j.Name] = j.Value.Value
		}
		ret = append(ret, m)
	}
	return ret
}


func (c *protoConverter) parameter(in *openapi_v2.Parameter) spec.Parameter {
	if paramKind := in.GetNonBodyParameter(); paramKind != nil {
		// TODO(mehdy): why we have different param kind for non-body parameters?
		// The difference is only one AllowEmptyValue and we should let the validity
		// check of the spec take care of that?
		if param := paramKind.GetFormDataParameterSubSchema(); param != nil {
			return spec.Parameter{
				ParamProps: spec.ParamProps{
					Description: param.Description,
					Name: param.Name,
					In: param.In,
					Required: param.Required,
					AllowEmptyValue: param.AllowEmptyValue,
				},
				CommonValidations: c.commonValidationFix(spec.CommonValidations {
					Maximum: &param.Maximum,
					ExclusiveMaximum: param.ExclusiveMaximum,
					Minimum: &param.Minimum,
					ExclusiveMinimum: param.ExclusiveMinimum,
					MaxLength: &param.MaxLength,
					MinLength: &param.MinLength,
					Pattern: param.Pattern,
					MaxItems: &param.MaxItems,
					MinItems: &param.MinItems,
					UniqueItems: param.UniqueItems,
					MultipleOf: &param.MultipleOf,
					Enum: c.enum(param.Enum),
				}),
				SimpleSchema: spec.SimpleSchema {
					Type: param.Type,
					Format: param.Format,
					Items: c.item(param.Items),
					CollectionFormat: param.CollectionFormat,
					Default: c.any(param.Default),
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: c.anyMap(param.VendorExtension),
				},
			}
		}
		if param := paramKind.GetHeaderParameterSubSchema(); param != nil {
			return spec.Parameter{
				ParamProps: spec.ParamProps{
					Description: param.Description,
					Name: param.Name,
					In: param.In,
					Required: param.Required,
				},
				CommonValidations: c.commonValidationFix(spec.CommonValidations {
					Maximum: &param.Maximum,
					ExclusiveMaximum: param.ExclusiveMaximum,
					Minimum: &param.Minimum,
					ExclusiveMinimum: param.ExclusiveMinimum,
					MaxLength: &param.MaxLength,
					MinLength: &param.MinLength,
					Pattern: param.Pattern,
					MaxItems: &param.MaxItems,
					MinItems: &param.MinItems,
					UniqueItems: param.UniqueItems,
					MultipleOf: &param.MultipleOf,
					Enum: c.enum(param.Enum),
				}),
				SimpleSchema: spec.SimpleSchema {
					Type: param.Type,
					Format: param.Format,
					Items: c.item(param.Items),
					CollectionFormat: param.CollectionFormat,
					Default: c.any(param.Default),
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: c.anyMap(param.VendorExtension),
				},
			}
		}
		if param := paramKind.GetPathParameterSubSchema(); param != nil {
			return spec.Parameter{
				ParamProps: spec.ParamProps{
					Description: param.Description,
					Name: param.Name,
					In: param.In,
					Required: param.Required,
				},
				CommonValidations: c.commonValidationFix(spec.CommonValidations {
					Maximum: &param.Maximum,
					ExclusiveMaximum: param.ExclusiveMaximum,
					Minimum: &param.Minimum,
					ExclusiveMinimum: param.ExclusiveMinimum,
					MaxLength: &param.MaxLength,
					MinLength: &param.MinLength,
					Pattern: param.Pattern,
					MaxItems: &param.MaxItems,
					MinItems: &param.MinItems,
					UniqueItems: param.UniqueItems,
					MultipleOf: &param.MultipleOf,
					Enum: c.enum(param.Enum),
				}),
				SimpleSchema: spec.SimpleSchema {
					Type: param.Type,
					Format: param.Format,
					Items: c.item(param.Items),
					CollectionFormat: param.CollectionFormat,
					Default: c.any(param.Default),
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: c.anyMap(param.VendorExtension),
				},
			}
		}
		if param := paramKind.GetQueryParameterSubSchema(); param != nil {
			return spec.Parameter{
				ParamProps: spec.ParamProps{
					Description: param.Description,
					Name: param.Name,
					In: param.In,
					Required: param.Required,
					AllowEmptyValue: param.AllowEmptyValue,
				},
				CommonValidations: c.commonValidationFix(spec.CommonValidations {
					Maximum: &param.Maximum,
					ExclusiveMaximum: param.ExclusiveMaximum,
					Minimum: &param.Minimum,
					ExclusiveMinimum: param.ExclusiveMinimum,
					MaxLength: &param.MaxLength,
					MinLength: &param.MinLength,
					Pattern: param.Pattern,
					MaxItems: &param.MaxItems,
					MinItems: &param.MinItems,
					UniqueItems: param.UniqueItems,
					MultipleOf: &param.MultipleOf,
					Enum: c.enum(param.Enum),
				}),
				SimpleSchema: spec.SimpleSchema {
					Type: param.Type,
					Format: param.Format,
					Items: c.item(param.Items),
					CollectionFormat: param.CollectionFormat,
					Default: c.any(param.Default),
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: c.anyMap(param.VendorExtension),
				},
			}
		}

	}
	if body := in.GetBodyParameter(); body != nil {
		// TODO(mehdy): common validation and AllowEmptyValue are missing in bodyParam.
		return spec.Parameter{
			ParamProps: spec.ParamProps{
				Description: body.Description,
				Name: body.Name,
				In: body.In,
				Required: body.Required,
				Schema: c.schema(body.Schema),
			},
			VendorExtensible: spec.VendorExtensible{
				Extensions: c.anyMap(body.VendorExtension),
			},
		}
	}
	return spec.Parameter{}
}

func (c *protoConverter) schema(in *openapi_v2.Schema) *spec.Schema {
	if in == nil {
		return nil
	}
	return &spec.Schema {
		SchemaProps: c.schemaPropFix(spec.SchemaProps{
			Maximum: &in.Maximum,
			ExclusiveMaximum: in.ExclusiveMaximum,
			Minimum: &in.Minimum,
			ExclusiveMinimum: in.ExclusiveMinimum,
			MaxLength: &in.MaxLength,
			MinLength: &in.MinLength,
			Pattern: in.Pattern,
			MaxItems: &in.MaxItems,
			MinItems: &in.MinItems,
			UniqueItems: in.UniqueItems,
			MultipleOf: &in.MultipleOf,
			Enum: c.enum(in.Enum),
			Type: c.typeItem(in.Type),
			Format: in.Format,
			Items: c.itemItem(in.Items),
			Default: c.any(in.Default),
			Ref: c.ref(in.XRef),
			Title: in.Title,
			Description: in.Description,
			MaxProperties: &in.MaxProperties,
			MinProperties: &in.MinProperties,
			Required: in.Required,
			AdditionalProperties: c.additionalProperty(in.AdditionalProperties),
			AllOf: c.schemaArray(in.AllOf),
			Properties: c.properties(in.Properties),
		}),
		SwaggerSchemaProps: spec.SwaggerSchemaProps {
			Discriminator: in.Discriminator,
			ReadOnly: in.ReadOnly,
			XML: c.xml(in.Xml),
			ExternalDocs: c.externalDocs(in.ExternalDocs),
			Example: c.any(in.Example),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) typeItem(in *openapi_v2.TypeItem) []string {
	if in == nil {
		return nil
	}
	return in.Value
}

func (c *protoConverter) externalDocs(in *openapi_v2.ExternalDocs) *spec.ExternalDocumentation {
	if in == nil {
		return nil
	}
	return &spec.ExternalDocumentation{
		Description: in.Description,
		URL: in.Url,
	}
}

func (c *protoConverter) xml(in *openapi_v2.Xml) *spec.XMLObject {
	if in == nil {
		return nil
	}
	return &spec.XMLObject{
		Name: in.Name,
		Namespace: in.Namespace,
		Prefix: in.Prefix,
		Attribute: in.Attribute,
		Wrapped: in.Wrapped,
	}
}

func (c *protoConverter) properties(in *openapi_v2.Properties) map[string]spec.Schema {
	if in == nil {
		return nil
	}
	ret := map[string]spec.Schema{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = *c.schema(i.Value)
	}
	return ret

}

func (c *protoConverter) schemaArray(in []*openapi_v2.Schema) []spec.Schema {

	ret := []spec.Schema{}
	for _, i := range in {
		ret = append(ret, *c.schema(i))
	}
	return ret
}

func (c *protoConverter) additionalProperty(in *openapi_v2.AdditionalPropertiesItem) *spec.SchemaOrBool {
	if in == nil {
		return nil
	}
	ret := spec.SchemaOrBool{Allows: in.GetBoolean()}
	if s := in.GetSchema(); s != nil {
		ret.Schema = c.schema(s)
	}
	return &ret
}

func (c *protoConverter) itemItem(in *openapi_v2.ItemsItem) *spec.SchemaOrArray {
	if in == nil {
		return nil
	}
	ret := spec.SchemaOrArray{}
	if len(in.Schema) == 1 {
		ret.Schema = c.schema(in.Schema[0])
	} else {
		ret.Schemas = []spec.Schema{}
		for _, i := range in.Schema {
			ret.Schemas = append(ret.Schemas, *c.schema(i))
		}
	}
	return &ret
}

func (c *protoConverter) item(in *openapi_v2.PrimitivesItems) *spec.Items {
	if in == nil {
		return nil
	}
	// TODO: Primitive items cannot have reference?
	// TODO: Items in spec is not extensible while Primitive proto item is extensible!
	return &spec.Items {
		CommonValidations: c.commonValidationFix(spec.CommonValidations{
			Maximum: &in.Maximum,
			ExclusiveMaximum: in.ExclusiveMaximum,
			Minimum: &in.Minimum,
			ExclusiveMinimum: in.ExclusiveMinimum,
			MaxLength: &in.MaxLength,
			MinLength: &in.MinLength,
			Pattern: in.Pattern,
			MaxItems: &in.MaxItems,
			MinItems: &in.MinItems,
			UniqueItems: in.UniqueItems,
			MultipleOf: &in.MultipleOf,
			Enum: c.enum(in.Enum),
		}),
		SimpleSchema: spec.SimpleSchema{
			Type: in.Type,
			Format: in.Format,
			Items: c.item(in.Items),
			CollectionFormat: in.CollectionFormat,
			Default: c.any(in.Default),
		},
	}
}

func (c *protoConverter) enum(in []*openapi_v2.Any) []interface{} {
	ret := []interface{}{}
	for _, i := range in {
		ret = append(ret, c.any(i))
	}
	return ret
}

func (c *protoConverter) rootParameters(in *openapi_v2.ParameterDefinitions) map[string]spec.Parameter {
	if in == nil {
		return nil
	}
	ret := map[string]spec.Parameter{}
	for _, i := range in.AdditionalProperties {
		ret[i.Name] = c.parameter(i.Value)
	}
	return ret
}

func (c *protoConverter) parameters(in []*openapi_v2.ParametersItem) []spec.Parameter {
	ret := []spec.Parameter{}
	for _, i := range in {
		if ref := i.GetJsonReference(); ref != nil {
			ret = append(ret, spec.Parameter{
				Refable: c.refable(ref.XRef),
				ParamProps: spec.ParamProps{
					Description: ref.Description,
				},
			})
		}
		if param := i.GetParameter(); param != nil {
			ret = append(ret, c.parameter(param))
		}
	}
	return ret
}

func (c *protoConverter) info(in *openapi_v2.Info) *spec.Info {
	if in == nil {
		return nil
	}
	return &spec.Info {
		InfoProps: spec.InfoProps {
			Description: in.Description,
			Title: in.Title,
			TermsOfService: in.TermsOfService,
			Contact: c.contact(in.Contact),
			License: c.license(in.License),
			Version: in.Version,
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: c.anyMap(in.VendorExtension),
		},
	}
}

func (c *protoConverter) contact(in *openapi_v2.Contact) *spec.ContactInfo {
	if in == nil {
		return nil
	}
	return &spec.ContactInfo{
		Name: in.Name,
		URL: in.Url,
		Email: in.Email,
	}
}

func (c *protoConverter) license(in *openapi_v2.License) *spec.License {
	if in == nil {
		return nil
	}
	return &spec.License{
		Name: in.Name,
		URL: in.Url,
	}
}

func (c *protoConverter) anyMap(in []*openapi_v2.NamedAny) map[string]interface{} {
	if in == nil {
		return nil
	}
	ret := map[string]interface{}{}
	for _, i := range in {
		ret[i.Name] = c.any(i.Value)
	}
	return ret
}

func (c *protoConverter) refable(in string) spec.Refable{
	return spec.Refable{Ref: c.ref(in)}
}

func (c *protoConverter) ref(in string) spec.Ref {
	if in == "" {
		return spec.Ref{}
	}
	r, _ := spec.NewRef(in) // TODO(mehdy): ?
	return r
}

type mystring struct {
	value string
}

func (m *mystring) SetString(str string) {
	m.value = str
}

func (c *protoConverter) any(in *openapi_v2.Any) interface{} {
	if in == nil {
		return nil
	}

	// Try string
	ptr := reflect.New(reflect.TypeOf(interface{}("")))
	if yaml.Unmarshal([]byte(in.Yaml), ptr.Interface()) == nil {
		return *ptr.Interface().(*string)
	}

	return in.Yaml  // TODO(mehdy): find a way to convert this.
}