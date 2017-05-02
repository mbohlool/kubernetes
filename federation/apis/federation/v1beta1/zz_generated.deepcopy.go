// +build !ignore_autogenerated

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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1beta1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	reflect "reflect"
)

func init() {
	SchemeBuilder.Register(RegisterDeepCopies)
}

// RegisterDeepCopies adds deep-copy functions to the given scheme. Public
// to allow building arbitrary schemes.
func RegisterDeepCopies(scheme *runtime.Scheme) error {
	return scheme.AddGeneratedDeepCopyFuncs(
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_Cluster, InType: reflect.TypeOf(&Cluster{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_ClusterCondition, InType: reflect.TypeOf(&ClusterCondition{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_ClusterList, InType: reflect.TypeOf(&ClusterList{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_ClusterSpec, InType: reflect.TypeOf(&ClusterSpec{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_ClusterStatus, InType: reflect.TypeOf(&ClusterStatus{})},
		conversion.GeneratedDeepCopyFunc{Fn: DeepCopy_v1beta1_ServerAddressByClientCIDR, InType: reflect.TypeOf(&ServerAddressByClientCIDR{})},
	)
}

func DeepCopy_v1beta1_Cluster(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*Cluster)
		out := out.(*Cluster)
		*out = *in
		if newVal, err := c.DeepCopy(&in.ObjectMeta); err != nil {
			return err
		} else {
			out.ObjectMeta = *newVal.(*v1.ObjectMeta)
		}
		if err := DeepCopy_v1beta1_ClusterSpec(&in.Spec, &out.Spec, c); err != nil {
			return err
		}
		if err := DeepCopy_v1beta1_ClusterStatus(&in.Status, &out.Status, c); err != nil {
			return err
		}
		return nil
	}
}

func DeepCopy_v1beta1_ClusterCondition(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ClusterCondition)
		out := out.(*ClusterCondition)
		*out = *in
		// in.Status is kind 'Unsupported'
		if in.Status != nil {
			if newVal, err := c.DeepCopy(&in.Status); err != nil {
				return err
			} else {
				out.Status = *newVal.(*unnameable_Unsupported)
			}
		}
		out.LastProbeTime = in.LastProbeTime.DeepCopy()
		out.LastTransitionTime = in.LastTransitionTime.DeepCopy()
		return nil
	}
}

func DeepCopy_v1beta1_ClusterList(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ClusterList)
		out := out.(*ClusterList)
		*out = *in
		if in.Items != nil {
			in, out := &in.Items, &out.Items
			*out = make([]Cluster, len(*in))
			for i := range *in {
				if err := DeepCopy_v1beta1_Cluster(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func DeepCopy_v1beta1_ClusterSpec(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ClusterSpec)
		out := out.(*ClusterSpec)
		*out = *in
		if in.ServerAddressByClientCIDRs != nil {
			in, out := &in.ServerAddressByClientCIDRs, &out.ServerAddressByClientCIDRs
			*out = make([]ServerAddressByClientCIDR, len(*in))
			copy(*out, *in)
		}
		if in.SecretRef != nil {
			in, out := &in.SecretRef, &out.SecretRef
			if newVal, err := c.DeepCopy(*in); err != nil {
				return err
			} else {
				*out = newVal.(*unnameable_Unsupported)
			}
		}
		return nil
	}
}

func DeepCopy_v1beta1_ClusterStatus(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ClusterStatus)
		out := out.(*ClusterStatus)
		*out = *in
		if in.Conditions != nil {
			in, out := &in.Conditions, &out.Conditions
			*out = make([]ClusterCondition, len(*in))
			for i := range *in {
				if err := DeepCopy_v1beta1_ClusterCondition(&(*in)[i], &(*out)[i], c); err != nil {
					return err
				}
			}
		}
		if in.Zones != nil {
			in, out := &in.Zones, &out.Zones
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
		return nil
	}
}

func DeepCopy_v1beta1_ServerAddressByClientCIDR(in interface{}, out interface{}, c *conversion.Cloner) error {
	{
		in := in.(*ServerAddressByClientCIDR)
		out := out.(*ServerAddressByClientCIDR)
		*out = *in
		return nil
	}
}
