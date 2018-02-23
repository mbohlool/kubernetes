package apiserver

import (
	"bytes"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/conversion"
	"fmt"
)

// GroupName is a generic name for CustomResource wrapper type.
const GroupName = "customresource.extensionapiserver.k8s.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
)

type CustomResource struct {
	obj *unstructured.Unstructured
}

var _ v1.Object = &CustomResource{}
var _ runtime.Object = &CustomResource{}

func (c *CustomResource) DeepCopyObject() runtime.Object {
	ret := CustomResource{}
	newObj := c.DeepCopyObject()
	ret.obj = newObj.(*unstructured.Unstructured)
	return &ret
}
func (c *CustomResource) GetObjectKind() schema.ObjectKind { return c.obj }

// MarshalJSON ensures that the unstructured object produces proper
// JSON when passed to Go's standard JSON library.
func (c *CustomResource) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	err := unstructured.UnstructuredJSONScheme.Encode(c.obj, &buf)
	return buf.Bytes(), err
}

// UnmarshalJSON ensures that the unstructured object properly decodes
// JSON when passed to Go's standard JSON library.
func (c *CustomResource) UnmarshalJSON(b []byte) error {
	_, _, err := unstructured.UnstructuredJSONScheme.Decode(b, nil, c.obj)
	return err
}

func (c *CustomResource) GetNamespace() string              { return c.obj.GetNamespace() }
func (c *CustomResource) SetNamespace(namespace string)     { c.obj.SetNamespace(namespace) }
func (c *CustomResource) GetName() string                   { return c.obj.GetName() }
func (c *CustomResource) SetName(name string)               { c.obj.SetName(name) }
func (c *CustomResource) GetGenerateName() string           { return c.obj.GetGenerateName() }
func (c *CustomResource) SetGenerateName(name string)       { c.obj.SetGenerateName(name) }
func (c *CustomResource) GetUID() types.UID                 { return c.obj.GetUID() }
func (c *CustomResource) SetUID(uid types.UID)              { c.obj.SetUID(uid) }
func (c *CustomResource) GetResourceVersion() string        { return c.obj.GetResourceVersion() }
func (c *CustomResource) SetResourceVersion(version string) { c.obj.SetResourceVersion(version) }
func (c *CustomResource) GetGeneration() int64              { return c.obj.GetGeneration() }
func (c *CustomResource) SetGeneration(generation int64)    { c.obj.SetGeneration(generation) }
func (c *CustomResource) GetSelfLink() string               { return c.obj.GetSelfLink() }
func (c *CustomResource) SetSelfLink(selfLink string)       { c.obj.SetSelfLink(selfLink) }
func (c *CustomResource) GetCreationTimestamp() v1.Time { return c.obj.GetCreationTimestamp() }
func (c *CustomResource) SetCreationTimestamp(timestamp v1.Time) {
	c.obj.SetCreationTimestamp(timestamp)
}
func (c *CustomResource) GetDeletionTimestamp() *v1.Time { return c.obj.GetDeletionTimestamp() }
func (c *CustomResource) SetDeletionTimestamp(timestamp *v1.Time) {
	c.obj.SetDeletionTimestamp(timestamp)
}
func (c *CustomResource) GetDeletionGracePeriodSeconds() *int64 {
	return c.obj.GetDeletionGracePeriodSeconds()
}
func (c *CustomResource) SetDeletionGracePeriodSeconds(t *int64) {
	c.obj.SetDeletionGracePeriodSeconds(t)
}
func (c *CustomResource) GetLabels() map[string]string       { return c.obj.GetLabels() }
func (c *CustomResource) SetLabels(labels map[string]string) { c.obj.SetLabels(labels) }
func (c *CustomResource) GetAnnotations() map[string]string  { return c.obj.GetAnnotations() }
func (c *CustomResource) SetAnnotations(annotations map[string]string) {
	c.obj.SetAnnotations(annotations)
}
func (c *CustomResource) GetInitializers() *v1.Initializers { return c.obj.GetInitializers() }
func (c *CustomResource) SetInitializers(initializers *v1.Initializers) {
	c.obj.SetInitializers(initializers)
}
func (c *CustomResource) GetFinalizers() []string           { return c.obj.GetFinalizers() }
func (c *CustomResource) SetFinalizers(finalizers []string) { c.obj.SetFinalizers(finalizers) }
func (c *CustomResource) GetOwnerReferences() []v1.OwnerReference {
	return c.obj.GetOwnerReferences()
}
func (c *CustomResource) SetOwnerReferences(p []v1.OwnerReference) { c.obj.SetOwnerReferences(p) }
func (c *CustomResource) GetClusterName() string                       { return c.obj.GetClusterName() }
func (c *CustomResource) SetClusterName(clusterName string)            { c.obj.SetClusterName(clusterName) }

var crdSchemeMap = map[schema.GroupVersion]*runtime.Scheme{}

// SchemeForCRD return a Scheme object for a CRD
// TODO: add mutex lock
func AddCRDScheme(crd apiextensions.CustomResourceDefinition, version string) *runtime.Scheme {
	crdGroupVersion := schema.GroupVersion{
		Group: crd.Spec.Group,
		Version: version,
	}
	ret, ok := crdSchemeMap[crdGroupVersion]
	if !ok {
		ret = runtime.NewScheme()
		crdSchemeMap[crdGroupVersion] = ret
		ret.AddKnownTypes(crdGroupVersion, &CustomResource{})
		v1.AddToGroupVersion(ret, crdGroupVersion)
		ret.AddConversionFuncs(
			func(in *CustomResource, out *CustomResource, scope conversion.Scope) error {
				outGVK := out.obj.GroupVersionKind()
				// TODO: Should we validate that group and kind are the same?
				out.obj.SetUnstructuredContent(in.obj.UnstructuredContent())
				out.obj.SetGroupVersionKind(outGVK)
				return nil
			})
		//ret.AddTypeDefaultingFunc()
		clusterScoped := crd.Spec.Scope == apiextensions.ClusterScoped
		ret.AddFieldLabelConversionFunc(version, "*", func (label, value string) (string, string, error) {
			// We currently only support metadata.namespace and metadata.name.
			switch {
			case label == "metadata.name":
			return label, value, nil
			case !clusterScoped && label == "metadata.namespace":
			return label, value, nil
			default:
			return "", "", fmt.Errorf("field label not supported: %s", label)
		}
		})
	}
	return ret
}

// RemoveCRDScheme removes Scheme for given CRD
func RemoveCRDScheme(crdGroupVersion schema.GroupVersion) {
	delete(crdSchemeMap, crdGroupVersion)
}

func