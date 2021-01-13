package storage

import (
	"context"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/apiserver-lite/kubernetes/storage/cacher"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite"
	"github.com/kubeedge/kubeedge/pkg/apiserverlite/util"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/storage"
)

// REST implements a RESTStorage for all resource against imitator.
type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against all resources
func NewREST() (*REST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &unstructured.Unstructured{} },
		NewListFunc:              func() runtime.Object { return &unstructured.UnstructuredList{} },
		DefaultQualifiedResource: schema.GroupResource{},

		KeyFunc: apiserverlite.KeyFuncReq,
		KeyRootFunc: apiserverlite.KeyRootFunc,

		CreateStrategy: nil,
		UpdateStrategy: nil,
		DeleteStrategy: nil,

		TableConvertor: nil,
		StorageVersioner : nil,
		Storage: genericregistry.DryRunnableStorage{},
	}
	store.PredicateFunc = func(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
		return storage.SelectionPredicate{
			Label:    label,
			Field:    field,
			GetAttrs: util.UnstructuredAttr,
		}
	}
	Storage,err:= cacher.NewCacher()
	utilruntime.Must(err)
	store.Storage.Storage = Storage
	store.Storage.Codec = unstructured.UnstructuredJSONScheme

	return &REST{store}, nil
}

// Deprecated: use REST.List to set list's gvk.
func decorator (obj runtime.Object) error {
	unstrList,ok :=obj.(*unstructured.UnstructuredList)
	if ok {
		var gvk schema.GroupVersionKind
		if len(unstrList.Items) != 0{
			gvk = unstrList.Items[0].GroupVersionKind()
		}
		gvk.Kind = gvk.Kind + "List"
		unstrList.GetObjectKind().SetGroupVersionKind(gvk)
	}
	return nil
}

func(r *REST)List(ctx context.Context, options *metainternalversion.ListOptions)(runtime.Object, error){
	obj,err := r.Store.List(ctx,options)
	if err != nil{
		return obj,err
	}
	info, ok  :=apirequest.RequestInfoFrom(ctx)
	if ok && obj.GetObjectKind().GroupVersionKind().Empty(){
		gvk := schema.GroupVersionKind{
			Group: info.APIGroup,
			Version: info.APIVersion,
			Kind: util.UnsafeResourceToKind(info.Resource) + "List",
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}
	return obj,err
}
