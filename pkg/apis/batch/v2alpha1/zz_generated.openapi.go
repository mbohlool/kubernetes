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

package v2alpha1

import (
	spec "github.com/go-openapi/spec"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1 "k8s.io/kubernetes/pkg/api/v1"
	openapi "k8s.io/kubernetes/pkg/genericapiserver/openapi"
)

func (_ Job) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "Job represents the configuration of a single job."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobStatus"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":      v1.ObjectMeta{},
			"v2alpha1.JobSpec":   JobSpec{},
			"v2alpha1.JobStatus": JobStatus{},
		},
	}
}

func (_ JobCondition) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobCondition describes current state of a job."
	schema.Properties = map[string]spec.Schema{
		"type": {
			SchemaProps: spec.SchemaProps{
				Description: "Type of job condition, Complete or Failed.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Description: "Status of the condition, one of True, False, Unknown.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"lastProbeTime": {
			SchemaProps: spec.SchemaProps{
				Description: "Last time the condition was checked.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"lastTransitionTime": {
			SchemaProps: spec.SchemaProps{
				Description: "Last time the condition transit from one status to another.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"reason": {
			SchemaProps: spec.SchemaProps{
				Description: "(brief) reason for the condition's last transition.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"message": {
			SchemaProps: spec.SchemaProps{
				Description: "Human readable message indicating details about last transition.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
	}
	schema.Required = []string{"type", "status"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ JobList) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobList is a collection of jobs."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/unversioned.ListMeta"),
			},
		},
		"items": {
			SchemaProps: spec.SchemaProps{
				Description: "Items is the list of Job.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v2alpha1.Job"),
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
			"unversioned.ListMeta": unversioned.ListMeta{},
			"v2alpha1.Job":         Job{},
		},
	}
}

func (_ JobSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobSpec describes how the job execution will look like."
	schema.Properties = map[string]spec.Schema{
		"parallelism": {
			SchemaProps: spec.SchemaProps{
				Description: "Parallelism specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: http://releases.k8s.io/HEAD/docs/user-guide/jobs.md",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"completions": {
			SchemaProps: spec.SchemaProps{
				Description: "Completions specifies the desired number of successfully finished pods the job should be run with.  Setting to nil means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value.  Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: http://releases.k8s.io/HEAD/docs/user-guide/jobs.md",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"activeDeadlineSeconds": {
			SchemaProps: spec.SchemaProps{
				Description: "Optional duration in seconds relative to the startTime that the job may be active before the system tries to terminate it; value must be positive integer",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"selector": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.LabelSelector"),
			},
		},
		"manualSelector": {
			SchemaProps: spec.SchemaProps{
				Description: "ManualSelector controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template.  When true, the user is responsible for picking unique labels and specifying the selector.  Failure to pick a unique label may cause this and other jobs to not function correctly.  However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. More info: http://releases.k8s.io/HEAD/docs/design/selector-generation.md",
				Type:        []string{"boolean"},
				Format:      "",
			},
		},
		"template": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.PodTemplateSpec"),
			},
		},
	}
	schema.Required = []string{"template"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.PodTemplateSpec":     v1.PodTemplateSpec{},
			"v2alpha1.LabelSelector": LabelSelector{},
		},
	}
}

func (_ JobStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobStatus represents the current state of a Job."
	schema.Properties = map[string]spec.Schema{
		"conditions": {
			SchemaProps: spec.SchemaProps{
				Description: "Conditions represent the latest available observations of an object's current state. More info: http://releases.k8s.io/HEAD/docs/user-guide/jobs.md",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobCondition"),
						},
					},
				},
			},
		},
		"startTime": {
			SchemaProps: spec.SchemaProps{
				Description: "StartTime represents time when the job was acknowledged by the Job Manager. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"completionTime": {
			SchemaProps: spec.SchemaProps{
				Description: "CompletionTime represents time when the job was completed. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
		"active": {
			SchemaProps: spec.SchemaProps{
				Description: "Active is the number of actively running pods.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"succeeded": {
			SchemaProps: spec.SchemaProps{
				Description: "Succeeded is the number of pods which reached Phase Succeeded.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
		"failed": {
			SchemaProps: spec.SchemaProps{
				Description: "Failed is the number of pods which reached Phase Failed.",
				Type:        []string{"integer"},
				Format:      "int32",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v2alpha1.JobCondition": JobCondition{},
		},
	}
}

func (_ JobTemplate) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobTemplate describes a template for creating copies of a predefined pod."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"template": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobTemplateSpec"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":            v1.ObjectMeta{},
			"v2alpha1.JobTemplateSpec": JobTemplateSpec{},
		},
	}
}

func (_ JobTemplateSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "JobTemplateSpec describes the data a Job should have when created from a template"
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobSpec"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":    v1.ObjectMeta{},
			"v2alpha1.JobSpec": JobSpec{},
		},
	}
}

func (_ LabelSelector) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "A label selector is a label query over a set of resources. The result of matchLabels and matchExpressions are ANDed. An empty label selector matches all objects. A null label selector matches no objects."
	schema.Properties = map[string]spec.Schema{
		"matchLabels": {
			SchemaProps: spec.SchemaProps{
				Description: "matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
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
		"matchExpressions": {
			SchemaProps: spec.SchemaProps{
				Description: "matchExpressions is a list of label selector requirements. The requirements are ANDed.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v2alpha1.LabelSelectorRequirement"),
						},
					},
				},
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v2alpha1.LabelSelectorRequirement": LabelSelectorRequirement{},
		},
	}
}

func (_ LabelSelectorRequirement) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values."
	schema.Properties = map[string]spec.Schema{
		"key": {
			SchemaProps: spec.SchemaProps{
				Description: "key is the label key that the selector applies to.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"operator": {
			SchemaProps: spec.SchemaProps{
				Description: "operator represents a key's relationship to a set of values. Valid operators ard In, NotIn, Exists and DoesNotExist.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"values": {
			SchemaProps: spec.SchemaProps{
				Description: "values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.",
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
	schema.Required = []string{"key", "operator"}
	return openapi.OpenAPIType{
		Schema:       &schema,
		Dependencies: map[string]interface{}{},
	}
}

func (_ ScheduledJob) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScheduledJob represents the configuration of a single scheduled job."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v1.ObjectMeta"),
			},
		},
		"spec": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.ScheduledJobSpec"),
			},
		},
		"status": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.ScheduledJobStatus"),
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectMeta":               v1.ObjectMeta{},
			"v2alpha1.ScheduledJobSpec":   ScheduledJobSpec{},
			"v2alpha1.ScheduledJobStatus": ScheduledJobStatus{},
		},
	}
}

func (_ ScheduledJobList) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScheduledJobList is a collection of scheduled jobs."
	schema.Properties = map[string]spec.Schema{
		"metadata": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/unversioned.ListMeta"),
			},
		},
		"items": {
			SchemaProps: spec.SchemaProps{
				Description: "Items is the list of ScheduledJob.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v2alpha1.ScheduledJob"),
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
			"unversioned.ListMeta":  unversioned.ListMeta{},
			"v2alpha1.ScheduledJob": ScheduledJob{},
		},
	}
}

func (_ ScheduledJobSpec) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScheduledJobSpec describes how the job execution will look like and when it will actually run."
	schema.Properties = map[string]spec.Schema{
		"schedule": {
			SchemaProps: spec.SchemaProps{
				Description: "Schedule contains the schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"startingDeadlineSeconds": {
			SchemaProps: spec.SchemaProps{
				Description: "Optional deadline in seconds for starting the job if it misses scheduled time for any reason.  Missed jobs executions will be counted as failed ones.",
				Type:        []string{"integer"},
				Format:      "int64",
			},
		},
		"concurrencyPolicy": {
			SchemaProps: spec.SchemaProps{
				Description: "ConcurrencyPolicy specifies how to treat concurrent executions of a Job.",
				Type:        []string{"string"},
				Format:      "",
			},
		},
		"suspend": {
			SchemaProps: spec.SchemaProps{
				Description: "Suspend flag tells the controller to suspend subsequent executions, it does not apply to already started executions.  Defaults to false.",
				Type:        []string{"boolean"},
				Format:      "",
			},
		},
		"jobTemplate": {
			SchemaProps: spec.SchemaProps{
				Ref: spec.MustCreateRef("#/definitions/v2alpha1.JobTemplateSpec"),
			},
		},
	}
	schema.Required = []string{"schedule", "jobTemplate"}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v2alpha1.JobTemplateSpec": JobTemplateSpec{},
		},
	}
}

func (_ ScheduledJobStatus) OpenAPI() openapi.OpenAPIType {
	schema := spec.Schema{}
	schema.Description = "ScheduledJobStatus represents the current state of a Job."
	schema.Properties = map[string]spec.Schema{
		"active": {
			SchemaProps: spec.SchemaProps{
				Description: "Active holds pointers to currently running jobs.",
				Type:        []string{"array"},
				Items: &spec.SchemaOrArray{
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: spec.MustCreateRef("#/definitions/v1.ObjectReference"),
						},
					},
				},
			},
		},
		"lastScheduleTime": {
			SchemaProps: spec.SchemaProps{
				Description: "LastScheduleTime keeps information of when was the last time the job was successfully scheduled.",
				Type:        []string{"string"},
				Format:      "date-time",
			},
		},
	}
	return openapi.OpenAPIType{
		Schema: &schema,
		Dependencies: map[string]interface{}{
			"v1.ObjectReference": v1.ObjectReference{},
		},
	}
}
