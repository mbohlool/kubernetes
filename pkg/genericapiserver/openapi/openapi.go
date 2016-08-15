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

// Note: Any reference to swagger in this document is to swagger 1.2 spec.

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"github.com/go-openapi/spec"
	"github.com/golang/glog"
)

// Config is set of configuration for openAPI spec generation.
type Config struct {
	// SwaggerConfig is set of configuration for go-restful swagger spec generation. Currently
	// openAPI implementation depends on go-restful to generate models.
	SwaggerConfig *swagger.Config
	// Info is general information about the API.
	Info *spec.Info
	// FixOperation is a customization function that will be called on each operation.
	// This function is optional.
	FixOperation func(path string, op *spec.Operation, repeated bool)
	// FixPath is a customization function that will be called on all generated path strings.
	// This function is optional.
	FixPath func(path string) string
	// FixResponses is a customization function that is called on all responses.
	// This function is optional.
	FixResponses func(responses *spec.Responses)
	// List of webservice's path prefixes to ignore
	IgnorePrefixes []string
}

type openAPI struct {
	config  *Config
	swagger *spec.Swagger
}

// RegisterOpenAPIService registers a handler for /swagger.json and provides standard
// swagger 2.0 (aka openAPI) specification.
func RegisterOpenAPIService(config *Config, containers *restful.Container) (err error) {
	o := openAPI{
		config: config,
	}
	err = o.buildSwaggerSpec()
	if err != nil {
		return err
	}
	containers.ServeMux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		resp := restful.NewResponse(w)
		if r.URL.Path != "/swagger.json" {
			resp.WriteErrorString(http.StatusNotFound, "Path not found!")
		}
		resp.WriteAsJson(o.swagger)
	})
	return nil
}

func (o *openAPI) buildSwaggerSpec() (err error) {
	if o.swagger != nil {
		return fmt.Errorf("Swagger spec is already built. Duplicate call to buildSwaggerSpec is not allowed.")
	}
	definitions, err := o.buildModels()
	if err != nil {
		return err
	}
	paths, err := o.buildPaths()
	if err != nil {
		return err
	}
	o.swagger = &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:     "2.0",
			Definitions: definitions,
			Paths:       &paths,
			Info:        o.config.Info,
		},
	}
	return nil
}

func (o *openAPI) buildModels() (definitions spec.Definitions, err error) {
	definitions = spec.Definitions{}
	for _, decl := range swagger.NewSwaggerBuilder(*o.config.SwaggerConfig).ProduceAllDeclarations() {
		for _, swaggerModel := range decl.Models.List {
			_, ok := definitions[swaggerModel.Name]
			if ok {
				// TODO(mbohlool): decide what to do with repeated models
				// The best way is to make sure they have the same content and
				// fail otherwise.
				continue
			}
			definitions[swaggerModel.Name], err = buildModel(swaggerModel.Model)
			if err != nil {
				return spec.Definitions{}, err
			}
		}
	}
	return definitions, nil
}

func buildModel(swaggerModel swagger.Model) (ret spec.Schema, err error) {
	ret = spec.Schema{}
	// model.SubTypes is not used in go-restful, ignoring.
	ret.Description = swaggerModel.Description
	ret.Required = swaggerModel.Required
	ret.Discriminator = swaggerModel.Discriminator
	ret.Properties = make(map[string]spec.Schema)
	for _, swaggerProp := range swaggerModel.Properties.List {
		if _, ok := ret.Properties[swaggerProp.Name]; ok {
			glog.Warning("Duplicate property in swagger 1.2 spec.")
		}
		ret.Properties[swaggerProp.Name], err = buildProperty(swaggerProp)
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func buildProperty(swaggerProperty swagger.NamedModelProperty) (openAPIProperty spec.Schema, err error) {
	openAPIProperty = spec.Schema{}
	if swaggerProperty.Property.Ref != nil {
		openAPIProperty.Ref = spec.MustCreateRef("#/definitions/" + *swaggerProperty.Property.Ref)
		return openAPIProperty, nil
	}
	openAPIProperty.Description = swaggerProperty.Property.Description
	openAPIProperty.Default = getDefaultValue(swaggerProperty.Property.DefaultValue)
	openAPIProperty.Enum = make([]interface{}, len(swaggerProperty.Property.Enum))
	for i, e := range swaggerProperty.Property.Enum {
		openAPIProperty.Enum[i] = e
	}
	openAPIProperty.Minimum, err = getFloat64OrNil(swaggerProperty.Property.Minimum)
	if err != nil {
		return spec.Schema{}, err
	}
	openAPIProperty.Maximum, err = getFloat64OrNil(swaggerProperty.Property.Maximum)
	if err != nil {
		return spec.Schema{}, err
	}
	if swaggerProperty.Property.UniqueItems != nil {
		openAPIProperty.UniqueItems = *swaggerProperty.Property.UniqueItems
	}

	if swaggerProperty.Property.Items != nil {
		if swaggerProperty.Property.Items.Ref != nil {
			openAPIProperty.Items = new(spec.SchemaOrArray)
			openAPIProperty.Items.Schema = new(spec.Schema)
			openAPIProperty.Items.Schema.Ref = spec.MustCreateRef("#/definitions/" + *swaggerProperty.Property.Items.Ref)
		} else {
			openAPIProperty.Items = new(spec.SchemaOrArray)
			openAPIProperty.Items.Schema = new(spec.Schema)
			openAPIProperty.Items.Schema.Type, openAPIProperty.Items.Schema.Format, err =
				buildType(swaggerProperty.Property.Items.Type, swaggerProperty.Property.Items.Format)
			if err != nil {
				return spec.Schema{}, err
			}
		}
	}
	openAPIProperty.Type, openAPIProperty.Format, err =
		buildType(swaggerProperty.Property.Type, swaggerProperty.Property.Format)
	if err != nil {
		return spec.Schema{}, err
	}
	return openAPIProperty, nil
}

func (o *openAPI) buildPaths() (spec.Paths, error) {
	paths := spec.Paths{
		Paths: make(map[string]spec.PathItem),
	}
	pathsToIgnore := createTrie(o.config.IgnorePrefixes)
	operationIdCount := make(map[string]int)
	// Initialize operation ID count map used in finding duplicate operation IDs.
	for _, service := range o.config.SwaggerConfig.WebServices {
		if pathsToIgnore.HasPrefix(service.RootPath()) {
			continue
		}
		for _, route := range service.Routes() {
			operationIdCount[route.Operation]++
		}
	}
	for _, w := range o.config.SwaggerConfig.WebServices {
		rootPath := w.RootPath()
		protocols, err := buildProtocolList(w)
		if err != nil {
			return paths, err
		}
		if pathsToIgnore.HasPrefix(rootPath) {
			continue
		}
		commonParams, err := buildParameters(w.PathParameters())
		if err != nil {
			return paths, err
		}
		for path, routes := range groupRoutesByPath(w.Routes()) {
			if o.config.FixPath != nil {
				path = o.config.FixPath(path)
			}
			inPathCommonParamsMap, err := findCommonParameters(routes)
			if err != nil {
				return paths, err
			}
			pathItem, exists := paths.Paths[path]
			if exists {
				return paths, fmt.Errorf("Duplicate webservice route has been found for path: %v", path)
			}
			pathItem = spec.PathItem{
				PathItemProps: spec.PathItemProps{
					Parameters: make([]spec.Parameter, 0),
				},
			}
			// add web services's parameters as well as any parameters appears in all ops, as common parameters
			for _, p := range commonParams {
				pathItem.Parameters = append(pathItem.Parameters, p)
			}
			for _, p := range inPathCommonParamsMap {
				pathItem.Parameters = append(pathItem.Parameters, p)
			}
			for _, route := range routes {
				op, err := o.buildOperations(route, inPathCommonParamsMap, protocols)
				if err != nil {
					return paths, err
				}
				switch strings.ToUpper(route.Method) {
				case "GET":
					pathItem.Get = op
				case "POST":
					pathItem.Post = op
				case "HEAD":
					pathItem.Head = op
				case "PUT":
					pathItem.Put = op
				case "DELETE":
					pathItem.Delete = op
				case "OPTIONS":
					pathItem.Options = op
				case "PATCH":
					pathItem.Patch = op
				}
				if o.config.FixOperation != nil {
					o.config.FixOperation(path, op, operationIdCount[op.ID] > 1)
				}
			}
			paths.Paths[path] = pathItem
		}
	}

	return paths, nil
}

func buildProtocolList(w *restful.WebService) ([]string, error) {
	// TODO(mehdy): we should be able to infer this from webService somehow.
	return []string{"http"}, nil
}

func (o *openAPI) buildOperations(route restful.Route, inPathCommonParamsMap map[string]spec.Parameter, protocols []string) (ret *spec.Operation, err error) {
	ret = &spec.Operation{
		OperationProps: spec.OperationProps{
			Description: route.Doc,
			Consumes:    route.Consumes,
			Produces:    route.Produces,
			ID:          route.Operation,
			Schemes:     protocols,
			Responses: &spec.Responses{
				ResponsesProps: spec.ResponsesProps{
					StatusCodeResponses: make(map[int]spec.Response),
				},
			},
		},
	}
	for _, resp := range route.ResponseErrors {
		openAPIResp := spec.Response{
			ResponseProps: spec.ResponseProps{
				Description: resp.Message,
				Schema:      new(spec.Schema),
			},
		}
		name := reflect.TypeOf(resp.Model).String()
		openAPIResp.Schema.Ref = spec.MustCreateRef("#/definitions/" + name)
		ret.Responses.StatusCodeResponses[resp.Code] = openAPIResp
	}
	if o.config.FixResponses != nil {
		o.config.FixResponses(ret.Responses)
	}
	ret.Parameters = make([]spec.Parameter, 0)
	for _, param := range route.ParameterDocs {
		_, isCommon := inPathCommonParamsMap[param.Data().Name]
		if !isCommon {
			openAPIParam, err := buildParameter(param.Data())
			if err != nil {
				return ret, err
			}
			ret.Parameters = append(ret.Parameters, openAPIParam)
		}
	}
	return ret, nil
}

func groupRoutesByPath(routes []restful.Route) (ret map[string][]restful.Route) {
	ret = make(map[string][]restful.Route)
	for _, r := range routes {
		route, exists := ret[r.Path]
		if !exists {
			route = make([]restful.Route, 0, 1)
		}
		ret[r.Path] = append(route, r)
	}
	return ret
}

func findCommonParameters(routes []restful.Route) (commonParamsMap map[string]spec.Parameter, err error) {
	commonParamsMap = make(map[string]spec.Parameter, 0)
	paramOpsCountMap := make(map[string]int, 0)
	paramDataMap := make(map[string]restful.ParameterData, 0)
	for _, route := range routes {
		routeParamMap := make(map[string]bool)
		for _, param := range route.ParameterDocs {
			if routeParamMap[param.Data().Name] {
				return commonParamsMap, fmt.Errorf("Duplicate parameter %v for route %v.", param.Data().Name, route)
			}
			paramOpsCountMap[param.Data().Name]++
			paramDataMap[param.Data().Name] = param.Data()
		}
	}
	for paramName, count := range paramOpsCountMap {
		if count == len(routes) {
			openAPIParam, err := buildParameter(paramDataMap[paramName])
			if err != nil {
				return commonParamsMap, err
			}
			commonParamsMap[paramName] = openAPIParam
		}
	}
	return commonParamsMap, nil
}

func buildParameter(restParam restful.ParameterData) (ret spec.Parameter, err error) {
	ret = spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:        restParam.Name,
			Description: restParam.Description,
			Required:    restParam.Required,
		},
	}
	switch restParam.Kind {
	case restful.BodyParameterKind:
		ret.In = "body"
		ret.Schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/" + restParam.DataType),
			},
		}
		return ret, nil
	case restful.PathParameterKind:
		ret.In = "path"
		if !restParam.Required {
			return ret, fmt.Errorf("Path parameters should be marked at required for parameter %v", restParam)
		}
	case restful.QueryParameterKind:
		ret.In = "query"
	case restful.HeaderParameterKind:
		ret.In = "header"
	case restful.FormParameterKind:
		ret.In = "form"
	default:
		return ret, fmt.Errorf("Unknown restful operation kind : %v", restParam.Kind)
	}
	if !isSimpleDataType(restParam.DataType) {
		return ret, fmt.Errorf("Restful DataType should be a simple type, but got : %v", restParam.DataType)
	}
	ret.Type = restParam.DataType
	ret.Format = restParam.DataFormat
	ret.UniqueItems = !restParam.AllowMultiple
	// TODO(mbohlool): make sure the type of default value matches Type
	if restParam.DefaultValue != "" {
		ret.Default = restParam.DefaultValue
	}

	return ret, nil
}

func buildParameters(restParam []*restful.Parameter) (ret []spec.Parameter, err error) {
	ret = make([]spec.Parameter, len(restParam))
	for i, v := range restParam {
		ret[i], err = buildParameter(v.Data())
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func isSimpleDataType(typeName string) bool {
	switch typeName {
	// Note that "file" intentionally kept out of this list as it is not being used.
	// "file" type has more requirements.
	case "string", "number", "integer", "boolean", "array":
		return true
	}
	return false
}

func getFloat64OrNil(str string) (*float64, error) {
	if len(str) > 0 {
		num, err := strconv.ParseFloat(str, 64)
		return &num, err
	}
	return nil, nil
}

// TODO(mbohlool): Convert default value type to the type of parameter
func getDefaultValue(str swagger.Special) interface{} {
	if len(str) > 0 {
		return str
	}
	return nil
}

func buildType(swaggerType *string, swaggerFormat string) ([]string, string, error) {
	if swaggerType == nil {
		return []string{}, "", nil
	}
	switch *swaggerType {
	case "integer", "number", "string", "boolean", "array", "object", "file":
		return []string{*swaggerType}, swaggerFormat, nil
	case "int":
		return []string{"integer"}, "int32", nil
	case "long":
		return []string{"integer"}, "int64", nil
	case "float", "double":
		return []string{"number"}, *swaggerType, nil
	case "byte", "date", "datetime", "date-time":
		return []string{"string"}, *swaggerType, nil
	default:
		return []string{}, "", fmt.Errorf("Unrecognized swagger 1.2 type : %v, %v", swaggerType, swaggerFormat)
	}
}

// A simple trie implementation with Add an HasPrefix methods only.
type trie struct {
	children map[byte]*trie
}

func createTrie(list []string) trie {
	ret := trie{
		children: make(map[byte]*trie),
	}
	for _, v := range list {
		ret.Add(v)
	}
	return ret
}

func (t *trie) Add(v string) {
	root := t
	for _, b := range []byte(v) {
		child, exists := root.children[b]
		if !exists {
			child = new(trie)
			child.children = make(map[byte]*trie)
			root.children[b] = child
		}
		root = child
	}
}

func (t *trie) HasPrefix(v string) bool {
	root := t
	for _, b := range []byte(v) {
		child, exists := root.children[b]
		if !exists {
			return false
		}
		root = child
	}
	return true
}
