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

package apiserver

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	crconversion "k8s.io/apiextensions-apiserver/pkg/apiserver/conversion"
	apiservervalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	informers "k8s.io/apiextensions-apiserver/pkg/client/informers/internalversion/apiextensions/internalversion"
	listers "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/internalversion"
	"k8s.io/apiextensions-apiserver/pkg/controller/finalizer"
	"k8s.io/apiextensions-apiserver/pkg/registry/customresource"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/cr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// crdHandler serves the `/apis` endpoint.
// This is registered as a filter so that it never collides with any explicitly registered endpoints
type crdHandler struct {
	versionDiscoveryHandler *versionDiscoveryHandler
	groupDiscoveryHandler   *groupDiscoveryHandler

	customStorageLock sync.Mutex
	// customStorage contains a crdStorageMap
	// atomic.Value has a very good read performance compared to sync.RWMutex
	// see https://gist.github.com/dim/152e6bf80e1384ea72e17ac717a5000a
	// which is suited for most read and rarely write cases
	customStorage atomic.Value

	requestContextMapper apirequest.RequestContextMapper

	crdLister listers.CustomResourceDefinitionLister

	delegate          http.Handler
	restOptionsGetter generic.RESTOptionsGetter
	admission         admission.Interface
}

// crdInfo stores enough information to serve the storage for the custom resource
type crdInfo struct {
	// spec and acceptedNames are used to compare against if a change is made on a CRD. We only update
	// the storage if one of these changes.
	spec          *apiextensions.CustomResourceDefinitionSpec
	acceptedNames *apiextensions.CustomResourceDefinitionNames

	// Storage per version
	storages map[string]*customresource.REST
	// Request scope per version
	requestScopes  map[string]handlers.RequestScope
	storageVersion string
}

// crdStorageMap goes from customresourcedefinition to its storage
type crdStorageMap map[types.UID]*crdInfo

func NewCustomResourceDefinitionHandler(
	versionDiscoveryHandler *versionDiscoveryHandler,
	groupDiscoveryHandler *groupDiscoveryHandler,
	requestContextMapper apirequest.RequestContextMapper,
	crdInformer informers.CustomResourceDefinitionInformer,
	delegate http.Handler,
	restOptionsGetter generic.RESTOptionsGetter,
	admission admission.Interface) *crdHandler {
	ret := &crdHandler{
		versionDiscoveryHandler: versionDiscoveryHandler,
		groupDiscoveryHandler:   groupDiscoveryHandler,
		customStorage:           atomic.Value{},
		requestContextMapper:    requestContextMapper,
		crdLister:               crdInformer.Lister(),
		delegate:                delegate,
		restOptionsGetter:       restOptionsGetter,
		admission:               admission,
	}

	crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: ret.updateCustomResourceDefinition,
		DeleteFunc: func(obj interface{}) {
			ret.removeDeadStorage()
		},
	})

	ret.customStorage.Store(crdStorageMap{})

	return ret
}

func (r *crdHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, ok := r.requestContextMapper.Get(req)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("no context found for request"))
		return
	}
	requestInfo, ok := apirequest.RequestInfoFrom(ctx)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("no RequestInfo found in the context"))
		return
	}
	if !requestInfo.IsResourceRequest {
		pathParts := splitPath(requestInfo.Path)
		// only match /apis/<group>/<version>
		// only registered under /apis
		if len(pathParts) == 3 {
			r.versionDiscoveryHandler.ServeHTTP(w, req)
			return
		}
		// only match /apis/<group>
		if len(pathParts) == 2 {
			r.groupDiscoveryHandler.ServeHTTP(w, req)
			return
		}

		r.delegate.ServeHTTP(w, req)
		return
	}

	crdName := requestInfo.Resource + "." + requestInfo.APIGroup
	crd, err := r.crdLister.Get(crdName)
	if apierrors.IsNotFound(err) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !apiextensions.HasCRDVersion(crd, requestInfo.APIVersion) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if !apiextensions.IsCRDConditionTrue(crd, apiextensions.Established) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if len(requestInfo.Subresource) > 0 {
		http.NotFound(w, req)
		return
	}

	terminating := apiextensions.IsCRDConditionTrue(crd, apiextensions.Terminating)

	crdInfo, err := r.getOrCreateServingInfoFor(crd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	storage := crdInfo.storages[requestInfo.APIVersion]
	requestScope := crdInfo.requestScopes[requestInfo.APIVersion]
	minRequestTimeout := 1 * time.Minute

	verb := strings.ToUpper(requestInfo.Verb)
	resource := requestInfo.Resource
	subresource := requestInfo.Subresource
	scope := metrics.CleanScope(requestInfo)

	var handler http.HandlerFunc

	switch requestInfo.Verb {
	case "get":
		handler = handlers.GetResource(storage, storage, requestScope)
	case "list":
		forceWatch := false
		handler = handlers.ListResource(storage, storage, requestScope, forceWatch, minRequestTimeout)
	case "watch":
		forceWatch := true
		handler = handlers.ListResource(storage, storage, requestScope, forceWatch, minRequestTimeout)
	case "create":
		if terminating {
			http.Error(w, fmt.Sprintf("%v not allowed while CustomResourceDefinition is terminating", requestInfo.Verb), http.StatusMethodNotAllowed)
			return
		}
		handler = handlers.CreateResource(storage, requestScope, newCRDObjectTyper(nil), r.admission)
	case "update":
		if terminating {
			http.Error(w, fmt.Sprintf("%v not allowed while CustomResourceDefinition is terminating", requestInfo.Verb), http.StatusMethodNotAllowed)
			return
		}
		handler = handlers.UpdateResource(storage, requestScope, newCRDObjectTyper(nil), r.admission)
	case "patch":
		if terminating {
			http.Error(w, fmt.Sprintf("%v not allowed while CustomResourceDefinition is terminating", requestInfo.Verb), http.StatusMethodNotAllowed)
			return
		}
		supportedTypes := []string{
			string(types.JSONPatchType),
			string(types.MergePatchType),
		}
		handler = handlers.PatchResource(storage, requestScope, r.admission,  crconversion.NewNoConversionConverter(crd.Spec.Scope == apiextensions.ClusterScoped), supportedTypes)
	case "delete":
		allowsOptions := true
		handler = handlers.DeleteResource(storage, allowsOptions, requestScope, r.admission)
	case "deletecollection":
		checkBody := true
		handler = handlers.DeleteCollection(storage, checkBody, requestScope, r.admission)
	default:
		http.Error(w, fmt.Sprintf("unhandled verb %q", requestInfo.Verb), http.StatusMethodNotAllowed)
		return
	}
	handler = metrics.InstrumentHandlerFunc(verb, resource, subresource, scope, handler)
	handler(w, req)
	return
}

func (r *crdHandler) updateCustomResourceDefinition(oldObj, newObj interface{}) {
	oldCRD := oldObj.(*apiextensions.CustomResourceDefinition)
	newCRD := newObj.(*apiextensions.CustomResourceDefinition)

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(crdStorageMap)
	oldInfo, found := storageMap[newCRD.UID]
	if !found {
		return
	}
	if apiequality.Semantic.DeepEqual(&newCRD.Spec, oldInfo.spec) && apiequality.Semantic.DeepEqual(&newCRD.Status.AcceptedNames, oldInfo.acceptedNames) {
		glog.V(6).Infof("Ignoring customresourcedefinition %s update because neither spec, nor accepted names changed", oldCRD.Name)
		return
	}

	glog.V(4).Infof("Updating customresourcedefinition %s", oldCRD.Name)

	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere.
	storageMap2 := storageMap.clone()
	if oldInfo, ok := storageMap2[types.UID(oldCRD.UID)]; ok {
		for _, storage := range oldInfo.storages {
			storage.DestroyFunc()
		}
		delete(storageMap2, types.UID(oldCRD.UID))
	}

	r.customStorage.Store(storageMap2)
}

// removeDeadStorage removes REST storage that isn't being used
func (r *crdHandler) removeDeadStorage() {
	allCustomResourceDefinitions, err := r.crdLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(crdStorageMap)
	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere
	storageMap2 := storageMap.clone()
	for uid, s := range storageMap2 {
		found := false
		for _, crd := range allCustomResourceDefinitions {
			if crd.UID == uid {
				found = true
				break
			}
		}
		if !found {
			glog.V(4).Infof("Removing dead CRD storage")
			for _, storage := range s.storages {
				storage.DestroyFunc()
			}
			delete(storageMap2, uid)
		}
	}
	r.customStorage.Store(storageMap2)
}

// GetCustomResourceListerCollectionDeleter returns the ListerCollectionDeleter of
// the given crd.
func (r *crdHandler) GetCustomResourceListerCollectionDeleter(crd *apiextensions.CustomResourceDefinition) (finalizer.ListerCollectionDeleter, error) {
	info, err := r.getOrCreateServingInfoFor(crd)
	if err != nil {
		return nil, err
	}
	return info.storages[info.storageVersion], nil
}

func (r *crdHandler) getOrCreateServingInfoFor(crd *apiextensions.CustomResourceDefinition) (*crdInfo, error) {
	storageMap := r.customStorage.Load().(crdStorageMap)
	if ret, ok := storageMap[crd.UID]; ok {
		return ret, nil
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap = r.customStorage.Load().(crdStorageMap)
	if ret, ok := storageMap[crd.UID]; ok {
		return ret, nil
	}

	storageVersion := apiextensions.GetCRDStorageVersion(crd)

	requestScopes := map[string]handlers.RequestScope{}
	storages := map[string]*customresource.REST{}
	for _, v := range crd.Spec.Versions {
		converter := crconversion.NewNoConversionConverter(crd.Spec.Scope == apiextensions.ClusterScoped)
		// In addition to Unstructured objects (Custom Resources), we also may sometimes need to
		// decode unversioned Options objects, so we delegate to parameterScheme for such types.
		parameterScheme := runtime.NewScheme()
		parameterScheme.AddUnversionedTypes(schema.GroupVersion{Group: crd.Spec.Group, Version: storageVersion},
			&metav1.ListOptions{},
			&metav1.ExportOptions{},
			&metav1.GetOptions{},
			&metav1.DeleteOptions{},
		)
		parameterCodec := runtime.NewParameterCodec(parameterScheme)

		typer := newCRDObjectTyper(parameterScheme)
		creator := crdCreator{}

		validator, err := apiservervalidation.NewSchemaValidator(crd.Spec.Validation)
		if err != nil {
			return nil, err
		}

		storages[v.Name] = customresource.NewREST(
			schema.GroupResource{Group: crd.Spec.Group, Resource: crd.Status.AcceptedNames.Plural},
			schema.GroupVersionKind{Group: crd.Spec.Group, Version: v.Name, Kind: crd.Status.AcceptedNames.ListKind},
			customresource.NewStrategy(
				typer,
				crd.Spec.Scope == apiextensions.NamespaceScoped,
				schema.GroupVersionKind{Group: crd.Spec.Group, Version: storageVersion, Kind: crd.Status.AcceptedNames.Kind},
				validator,
				converter,
				storageVersion,
			),
			r.restOptionsGetter,
		)

		selfLinkPrefix := ""
		switch crd.Spec.Scope {
		case apiextensions.ClusterScoped:
			selfLinkPrefix = "/" + path.Join("apis", crd.Spec.Group, v.Name) + "/" + crd.Status.AcceptedNames.Plural + "/"
		case apiextensions.NamespaceScoped:
			selfLinkPrefix = "/" + path.Join("apis", crd.Spec.Group, v.Name, "namespaces") + "/"
		}

		clusterScoped := crd.Spec.Scope == apiextensions.ClusterScoped
		requestScopes[v.Name] = handlers.RequestScope{
			Namer: handlers.ContextBasedNaming{
				GetContext: func(req *http.Request) apirequest.Context {
					ret, _ := r.requestContextMapper.Get(req)
					return ret
				},
				SelfLinker:         meta.NewAccessor(),
				ClusterScoped:      clusterScoped,
				SelfLinkPathPrefix: selfLinkPrefix,
			},
			ContextFunc: func(req *http.Request) apirequest.Context {
				ret, _ := r.requestContextMapper.Get(req)
				return ret
			},

			Serializer:     crdNegotiatedSerializer{typer: typer, creator: creator, converter: converter},
			ParameterCodec: parameterCodec,

			Creater:         creator,
			Convertor:       converter,
			Defaulter:       crdDefaulter{parameterScheme},
			Typer:           typer,
			UnsafeConvertor: converter,

			Resource:    schema.GroupVersionResource{Group: crd.Spec.Group, Version: v.Name, Resource: crd.Status.AcceptedNames.Plural},
			Kind:        schema.GroupVersionKind{Group: crd.Spec.Group, Version: v.Name, Kind: crd.Status.AcceptedNames.Kind},
			Subresource: "",

			MetaGroupVersion: metav1.SchemeGroupVersion,
		}
	}

	ret := &crdInfo{
		spec:          &crd.Spec,
		acceptedNames: &crd.Status.AcceptedNames,

		storages:       storages,
		requestScopes:  requestScopes,
		storageVersion: storageVersion,
	}

	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere.
	storageMap2 := storageMap.clone()

	storageMap2[crd.UID] = ret
	r.customStorage.Store(storageMap2)

	return ret, nil
}

type crdNegotiatedSerializer struct {
	typer     runtime.ObjectTyper
	creator   runtime.ObjectCreater
	converter runtime.ObjectConvertor
}

func (s crdNegotiatedSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			EncodesAsText:    true,
			Serializer:       json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, false),
			PrettySerializer: json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, true),
			StreamSerializer: &runtime.StreamSerializerInfo{
				EncodesAsText: true,
				Serializer:    json.NewSerializer(json.DefaultMetaFactory, s.creator, s.typer, false),
				Framer:        json.Framer,
			},
		},
		{
			MediaType:     "application/yaml",
			EncodesAsText: true,
			Serializer:    json.NewYAMLSerializer(json.DefaultMetaFactory, s.creator, s.typer),
		},
	}
}

func (s crdNegotiatedSerializer) EncoderForVersion(encoder runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return versioning.NewCodec(encoder, nil, s.converter, crdCreator{}, newCRDObjectTyper(Scheme), Scheme, gv, nil)
}

func (s crdNegotiatedSerializer) DecoderToVersion(decoder runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return versioning.NewCodec(nil,
		decoder, s.converter, crdCreator{}, newCRDObjectTyper(Scheme), Scheme, nil, gv)
}

type crdObjectTyper struct {
	delegate          runtime.ObjectTyper
	unstructuredTyper runtime.ObjectTyper
}

func newCRDObjectTyper(delegate runtime.ObjectTyper) *crdObjectTyper {
	return &crdObjectTyper{
		delegate: delegate,
		unstructuredTyper: discovery.NewUnstructuredObjectTyper(nil),
	}
}

func (t crdObjectTyper) ObjectKinds(obj runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	// Delegate for things other than Unstructured.
	crd, ok := obj.(*cr.CustomResource)
	if !ok {
		if t.delegate != nil {
			return t.delegate.ObjectKinds(obj)
		} else {
			return t.unstructuredTyper.ObjectKinds(obj)
		}
	}
	return t.unstructuredTyper.ObjectKinds(crd.Obj)
}

func (t crdObjectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	if t.delegate != nil {
		return t.delegate.Recognizes(gvk) || t.unstructuredTyper.Recognizes(gvk)
	} else {
		return t.unstructuredTyper.Recognizes(gvk)
	}
}

type crdCreator struct{}

func (c crdCreator) New(kind schema.GroupVersionKind) (runtime.Object, error) {
	ret := &cr.CustomResource{Obj:&unstructured.Unstructured{}}
	ret.Obj.SetGroupVersionKind(kind)
	return ret, nil
}

type crdDefaulter struct {
	delegate runtime.ObjectDefaulter
}

func (d crdDefaulter) Default(in runtime.Object) {
	// Delegate for things other than Unstructured.
	if crd, ok := in.(*cr.CustomResource); !ok {
		d.delegate.Default(crd.Obj)
	}
}

type CRDRESTOptionsGetter struct {
	StorageConfig           storagebackend.Config
	StoragePrefix           string
	EnableWatchCache        bool
	DefaultWatchCacheSize   int
	EnableGarbageCollection bool
	DeleteCollectionWorkers int
}

func (t CRDRESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	ret := generic.RESTOptions{
		StorageConfig:           &t.StorageConfig,
		Decorator:               generic.UndecoratedStorage,
		EnableGarbageCollection: t.EnableGarbageCollection,
		DeleteCollectionWorkers: t.DeleteCollectionWorkers,
		ResourcePrefix:          resource.Group + "/" + resource.Resource,
	}
	if t.EnableWatchCache {
		ret.Decorator = genericregistry.StorageWithCacher(t.DefaultWatchCacheSize)
	}
	return ret, nil
}

// clone returns a clone of the provided crdStorageMap.
// The clone is a shallow copy of the map.
func (in crdStorageMap) clone() crdStorageMap {
	if in == nil {
		return nil
	}
	out := make(crdStorageMap, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
