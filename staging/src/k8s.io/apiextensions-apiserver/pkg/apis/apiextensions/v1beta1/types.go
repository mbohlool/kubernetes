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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type ConversionStrategyType string

const (
	// NopConverter is a converter that only sets apiversion of the CR and leave everything else unchanged.
	NopConverter     ConversionStrategyType = "None"
	WebhookConverter ConversionStrategyType = "Webhook"
)

// CustomResourceDefinitionSpec describes how a user wants their resource to appear
type CustomResourceDefinitionSpec struct {
	// Group is the group this resource belongs in
	Group string `json:"group" protobuf:"bytes,1,opt,name=group"`
	// Version is the version this resource belongs in
	// Should be always first item in Versions field if provided.
	// Optional, but at least one of Version or Versions must be set.
	// Deprecated: Please use `Versions`.
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	// Names are the names used to describe this custom resource
	Names CustomResourceDefinitionNames `json:"names" protobuf:"bytes,3,opt,name=names"`
	// Scope indicates whether this resource is cluster or namespace scoped.  Default is namespaced
	Scope ResourceScope `json:"scope" protobuf:"bytes,4,opt,name=scope,casttype=ResourceScope"`
	// Validation describes the validation methods for CustomResources
	// +optional
	Validation *CustomResourceValidation `json:"validation,omitempty" protobuf:"bytes,5,opt,name=validation"`
	// Subresources describes the subresources for CustomResources
	// +optional
	Subresources *CustomResourceSubresources `json:"subresources,omitempty" protobuf:"bytes,6,opt,name=subresources"`
	// Versions is the list of all supported versions for this resource.
	// If Version field is provided, this field is optional.
	// Validation: All versions must use the same validation schema for now. i.e., top
	// level Validation field is applied to all of these versions.
	// Order: The version name will be used to compute the order.
	// If the version string is "kube-like", it will sort above non "kube-like" version strings, which are ordered
	// lexicographically. "Kube-like" versions start with a "v", then are followed by a number (the major version),
	// then optionally the string "alpha" or "beta" and another number (the minor version). These are sorted first
	// by GA > beta > alpha (where GA is a version with no suffix such as beta or alpha), and then by comparing
	// major version, then minor version. An example sorted list of versions:
	// v10, v2, v1, v11beta2, v10beta3, v3beta1, v12alpha1, v11alpha2, foo1, foo10.
	Versions []CustomResourceDefinitionVersion `json:"versions,omitempty" protobuf:"bytes,7,rep,name=versions"`
	// AdditionalPrinterColumns are additional columns shown e.g. in kubectl next to the name. Defaults to a created-at column.
	AdditionalPrinterColumns []CustomResourceColumnDefinition `json:"additionalPrinterColumns,omitempty" protobuf:"bytes,8,rep,name=additionalPrinterColumns"`

	// conversion defines conversion settings for the CRD.
	// +optional
	Conversion *CustomResourceConversion `json:"conversion,omitempty" protobuf:"bytes,9,opt,name=conversion"`
}

type CustomResourceConversion struct {
	// Strategy specifies the conversion strategy. Allowed values are:
	// - `None`: The converter only change the apiVersion and would not touch any other field in the CR.
	// - `Webhook`: API Server will call to an external webhook to do the conversion. Additional information is needed for this option.
	Strategy ConversionStrategyType `json:"strategy" protobuf:"bytes,1,name=strategy"`

	// Additional information for external conversion if strategy is set to external
	// +optional
	Webhook *CustomResourceConversionWebhook `json:"webhook,omitempty" protobuf:"bytes,2,name=webhook"`
}

type CustomResourceConversionWebhook struct {
	// ClientConfig defines how to communicate with the webhook. This is the same config used for validating/mutating webhooks.
	ClientConfig WebhookClientConfig `json:"clientConfig" protobuf:"bytes,1,name=clientConfig"`
}

// WebhookClientConfig contains the information to make a TLS
// connection with the webhook. It has the same field as admissionregistration.v1beta1.WebhookClientConfig.
type WebhookClientConfig struct {
	// `url` gives the location of the webhook, in standard URL form
	// (`[scheme://]host:port/path`). Exactly one of `url` or `service`
	// must be specified.
	//
	// The `host` should not refer to a service running in the cluster; use
	// the `service` field instead. The host might be resolved via external
	// DNS in some apiservers (e.g., `kube-apiserver` cannot resolve
	// in-cluster DNS as that would be a layering violation). `host` may
	// also be an IP address.
	//
	// Please note that using `localhost` or `127.0.0.1` as a `host` is
	// risky unless you take great care to run this webhook on all hosts
	// which run an apiserver which might need to make calls to this
	// webhook. Such installs are likely to be non-portable, i.e., not easy
	// to turn up in a new cluster.
	//
	// The scheme must be "https"; the URL must begin with "https://".
	//
	// A path is optional, and if present may be any string permissible in
	// a URL. You may use the path to pass an arbitrary string to the
	// webhook, for example, a cluster identifier.
	//
	// Attempting to use a user or basic auth e.g. "user:password@" is not
	// allowed. Fragments ("#...") and query parameters ("?...") are not
	// allowed, either.
	//
	// +optional
	URL *string `json:"url,omitempty" protobuf:"bytes,3,opt,name=url"`

	// `service` is a reference to the service for this webhook. Either
	// `service` or `url` must be specified.
	//
	// If the webhook is running within the cluster, then you should use `service`.
	//
	// Port 443 will be used if it is open, otherwise it is an error.
	//
	// +optional
	Service *ServiceReference `json:"service" protobuf:"bytes,1,opt,name=service"`

	// `caBundle` is a PEM encoded CA bundle which will be used to validate
	// the webhook's server certificate.
	// Required.
	CABundle []byte `json:"caBundle" protobuf:"bytes,2,opt,name=caBundle"`
}

// ServiceReference holds a reference to Service.legacy.k8s.io
type ServiceReference struct {
	// `namespace` is the namespace of the service.
	// Required
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	// `name` is the name of the service.
	// Required
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

	// `path` is an optional URL path which will be sent in any request to
	// this service.
	// +optional
	Path *string `json:"path,omitempty" protobuf:"bytes,3,opt,name=path"`
}

type CustomResourceDefinitionVersion struct {
	// Name is the version name, e.g. “v1”, “v2beta1”, etc.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Served is a flag enabling/disabling this version from being served via REST APIs
	Served bool `json:"served" protobuf:"varint,2,opt,name=served"`
	// Storage flags the version as storage version. There must be exactly one
	// flagged as storage version.
	Storage bool `json:"storage" protobuf:"varint,3,opt,name=storage"`
}

// CustomResourceColumnDefinition specifies a column for server side printing.
type CustomResourceColumnDefinition struct {
	// name is a human readable name for the column.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// type is an OpenAPI type definition for this column.
	// See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.
	Type string `json:"type" protobuf:"bytes,2,opt,name=type"`
	// format is an optional OpenAPI type definition for this column. The 'name' format is applied
	// to the primary identifier column to assist in clients identifying column is the resource name.
	// See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.
	Format string `json:"format,omitempty" protobuf:"bytes,3,opt,name=format"`
	// description is a human readable description of this column.
	Description string `json:"description,omitempty" protobuf:"bytes,4,opt,name=description"`
	// priority is an integer defining the relative importance of this column compared to others. Lower
	// numbers are considered higher priority. Columns that may be omitted in limited space scenarios
	// should be given a higher priority.
	Priority int32 `json:"priority,omitempty" protobuf:"bytes,5,opt,name=priority"`

	// JSONPath is a simple JSON path, i.e. with array notation.
	JSONPath string `json:"JSONPath" protobuf:"bytes,6,opt,name=JSONPath"`
}

// CustomResourceDefinitionNames indicates the names to serve this CustomResourceDefinition
type CustomResourceDefinitionNames struct {
	// Plural is the plural name of the resource to serve.  It must match the name of the CustomResourceDefinition-registration
	// too: plural.group and it must be all lowercase.
	Plural string `json:"plural" protobuf:"bytes,1,opt,name=plural"`
	// Singular is the singular name of the resource.  It must be all lowercase  Defaults to lowercased <kind>
	Singular string `json:"singular,omitempty" protobuf:"bytes,2,opt,name=singular"`
	// ShortNames are short names for the resource.  It must be all lowercase.
	ShortNames []string `json:"shortNames,omitempty" protobuf:"bytes,3,opt,name=shortNames"`
	// Kind is the serialized kind of the resource.  It is normally CamelCase and singular.
	Kind string `json:"kind" protobuf:"bytes,4,opt,name=kind"`
	// ListKind is the serialized kind of the list for this resource.  Defaults to <kind>List.
	ListKind string `json:"listKind,omitempty" protobuf:"bytes,5,opt,name=listKind"`
	// Categories is a list of grouped resources custom resources belong to (e.g. 'all')
	// +optional
	Categories []string `json:"categories,omitempty" protobuf:"bytes,6,rep,name=categories"`
}

// ResourceScope is an enum defining the different scopes available to a custom resource
type ResourceScope string

const (
	ClusterScoped   ResourceScope = "Cluster"
	NamespaceScoped ResourceScope = "Namespaced"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// CustomResourceDefinitionConditionType is a valid value for CustomResourceDefinitionCondition.Type
type CustomResourceDefinitionConditionType string

const (
	// Established means that the resource has become active. A resource is established when all names are
	// accepted without a conflict for the first time. A resource stays established until deleted, even during
	// a later NamesAccepted due to changed names. Note that not all names can be changed.
	Established CustomResourceDefinitionConditionType = "Established"
	// NamesAccepted means the names chosen for this CustomResourceDefinition do not conflict with others in
	// the group and are therefore accepted.
	NamesAccepted CustomResourceDefinitionConditionType = "NamesAccepted"
	// Terminating means that the CustomResourceDefinition has been deleted and is cleaning up.
	Terminating CustomResourceDefinitionConditionType = "Terminating"
)

// CustomResourceDefinitionCondition contains details for the current condition of this pod.
type CustomResourceDefinitionCondition struct {
	// Type is the type of the condition.
	Type CustomResourceDefinitionConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=CustomResourceDefinitionConditionType"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// CustomResourceDefinitionStatus indicates the state of the CustomResourceDefinition
type CustomResourceDefinitionStatus struct {
	// Conditions indicate state for particular aspects of a CustomResourceDefinition
	Conditions []CustomResourceDefinitionCondition `json:"conditions" protobuf:"bytes,1,opt,name=conditions"`

	// AcceptedNames are the names that are actually being used to serve discovery
	// They may be different than the names in spec.
	AcceptedNames CustomResourceDefinitionNames `json:"acceptedNames" protobuf:"bytes,2,opt,name=acceptedNames"`

	// StoredVersions are all versions of CustomResources that were ever persisted. Tracking these
	// versions allows a migration path for stored versions in etcd. The field is mutable
	// so the migration controller can first finish a migration to another version (i.e.
	// that no old objects are left in the storage), and then remove the rest of the
	// versions from this list.
	// None of the versions in this list can be removed from the spec.Versions field.
	StoredVersions []string `json:"storedVersions" protobuf:"bytes,3,rep,name=storedVersions"`
}

// CustomResourceCleanupFinalizer is the name of the finalizer which will delete instances of
// a CustomResourceDefinition
const CustomResourceCleanupFinalizer = "customresourcecleanup.apiextensions.k8s.io"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomResourceDefinition represents a resource that should be exposed on the API server.  Its name MUST be in the format
// <.spec.name>.<.spec.group>.
type CustomResourceDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec describes how the user wants the resources to appear
	Spec CustomResourceDefinitionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status indicates the actual state of the CustomResourceDefinition
	Status CustomResourceDefinitionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomResourceDefinitionList is a list of CustomResourceDefinition objects.
type CustomResourceDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items individual CustomResourceDefinitions
	Items []CustomResourceDefinition `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// CustomResourceValidation is a list of validation methods for CustomResources.
type CustomResourceValidation struct {
	// OpenAPIV3Schema is the OpenAPI v3 schema to be validated against.
	OpenAPIV3Schema *JSONSchemaProps `json:"openAPIV3Schema,omitempty" protobuf:"bytes,1,opt,name=openAPIV3Schema"`
}

// CustomResourceSubresources defines the status and scale subresources for CustomResources.
type CustomResourceSubresources struct {
	// Status denotes the status subresource for CustomResources
	Status *CustomResourceSubresourceStatus `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// Scale denotes the scale subresource for CustomResources
	Scale *CustomResourceSubresourceScale `json:"scale,omitempty" protobuf:"bytes,2,opt,name=scale"`
}

// CustomResourceSubresourceStatus defines how to serve the status subresource for CustomResources.
// Status is represented by the `.status` JSON path inside of a CustomResource. When set,
// * exposes a /status subresource for the custom resource
// * PUT requests to the /status subresource take a custom resource object, and ignore changes to anything except the status stanza
// * PUT/POST/PATCH requests to the custom resource ignore changes to the status stanza
type CustomResourceSubresourceStatus struct{}

// CustomResourceSubresourceScale defines how to serve the scale subresource for CustomResources.
type CustomResourceSubresourceScale struct {
	// SpecReplicasPath defines the JSON path inside of a CustomResource that corresponds to Scale.Spec.Replicas.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .spec.
	// If there is no value under the given path in the CustomResource, the /scale subresource will return an error on GET.
	SpecReplicasPath string `json:"specReplicasPath" protobuf:"bytes,1,name=specReplicasPath"`
	// StatusReplicasPath defines the JSON path inside of a CustomResource that corresponds to Scale.Status.Replicas.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .status.
	// If there is no value under the given path in the CustomResource, the status replica value in the /scale subresource
	// will default to 0.
	StatusReplicasPath string `json:"statusReplicasPath" protobuf:"bytes,2,opt,name=statusReplicasPath"`
	// LabelSelectorPath defines the JSON path inside of a CustomResource that corresponds to Scale.Status.Selector.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .status.
	// Must be set to work with HPA.
	// If there is no value under the given path in the CustomResource, the status label selector value in the /scale
	// subresource will default to the empty string.
	// +optional
	LabelSelectorPath *string `json:"labelSelectorPath,omitempty" protobuf:"bytes,3,opt,name=labelSelectorPath"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConversionReview describes a conversion request/response.
type ConversionReview struct {
	metav1.TypeMeta `json:",inline"`
	// Request describes the attributes for the conversion request.
	// +optional
	Request *ConversionRequest `json:"request,omitempty" protobuf:"bytes,1,opt,name=request"`
	// Response describes the attributes for the conversion response.
	// +optional
	Response *ConversionResponse `json:"response,omitempty" protobuf:"bytes,2,opt,name=response"`
}

// ConversionRequest describes a conversion request parameters.
type ConversionRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the KAS and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	// +optional
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,1,opt,name=uid"`
	// The version to convert given object to. E.g. "stable.example.com/v1"
	APIVersion string `json:"apiVersion" protobuf:"bytes,2,name=apiVersion"`
	// Object is the CRD object to be converted.
	Object runtime.RawExtension `json:"object" protobuf:"bytes,3,name=object"`
	// IsList indicates that this is a List operation and Object contains a list of items.
	IsList bool `json:"isList" protobuf:"varint,4,name=isList"`
}

// ConversionResponse describes a conversion response.
type ConversionResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding AdmissionRequest.
	// +optional
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,1,opt,name=uid"`
	// ConvertedObject is the converted version of request.Object if the Result is nil.
	// +optional
	ConvertedObject *runtime.RawExtension `json:"convertedObject" protobuf:"bytes,2,name=convertedObject"`
	// Result contains extra details into why a conversion request was failed. If it is not nil, it means
	// the conversion is failed.
	// +optional
	Result *metav1.Status `json:"result" protobuf:"bytes,3,name=result"`
}
