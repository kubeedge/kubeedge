package storage

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/cacher"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// REST implements a RESTStorage for all resource against imitator.
type REST struct {
	*genericregistry.Store
	*application.Agent
}

// NewREST returns a RESTStorage object that will work against all resources
func NewREST() (*REST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &unstructured.Unstructured{} },
		NewListFunc:              func() runtime.Object { return &unstructured.UnstructuredList{} },
		DefaultQualifiedResource: schema.GroupResource{},

		KeyFunc:     metaserver.KeyFuncReq,
		KeyRootFunc: metaserver.KeyRootFunc,

		CreateStrategy: nil,
		UpdateStrategy: nil,
		DeleteStrategy: nil,

		TableConvertor:   nil,
		StorageVersioner: nil,
		Storage:          genericregistry.DryRunnableStorage{},
	}
	store.PredicateFunc = func(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
		return storage.SelectionPredicate{
			Label:    label,
			Field:    field,
			GetAttrs: util.UnstructuredAttr,
		}
	}
	Storage, err := cacher.NewCacher()
	utilruntime.Must(err)
	store.Storage.Storage = Storage
	store.Storage.Codec = unstructured.UnstructuredJSONScheme

	return &REST{store, application.NewApplicationAgent(metaserverconfig.Config.NodeName)}, nil
}

// Deprecated: use REST.List to set list's gvk.
func decorator(obj runtime.Object) error {
	unstrList, ok := obj.(*unstructured.UnstructuredList)
	if ok {
		var gvk schema.GroupVersionKind
		if len(unstrList.Items) != 0 {
			gvk = unstrList.Items[0].GroupVersionKind()
		}
		gvk.Kind = gvk.Kind + "List"
		unstrList.GetObjectKind().SetGroupVersionKind(gvk)
	}
	return nil
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	path := info.Path
	// try remote
	obj, err := func() (runtime.Object, error) {
		app := r.Agent.Generate(ctx, application.Get, *options, application.LabelFieldSelector{}, nil)
		err := r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to get obj from cloud, %v", err)
			return nil, errors.NewInternalError(err)
		}
		var obj = new(unstructured.Unstructured)
		err = json.Unmarshal(app.RespBody, obj)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		// save to local, ignore error
		imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), obj)
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) through cloud", path)
		return obj, nil
	}()
	// try local
	if err != nil {
		obj, err = r.Store.Get(ctx, "", options) // name is needless, we get all key information from ctx
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) at local", path)
	}
	return obj, err
}
func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	path := info.Path
	// try remote
	list, err := func() (runtime.Object, error) {
		selector := application.LabelFieldSelector{Label: options.LabelSelector, Field: options.FieldSelector}
		app := r.Agent.Generate(ctx, application.List, *options, selector, nil)
		err := r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to list obj from cloud, %v", err)
			return nil, errors.NewInternalError(err)
		}
		var list = new(unstructured.UnstructuredList)
		err = json.Unmarshal(app.RespBody, list)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		// ignore error
		// imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), list)
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) through cloud", path)
		return list, nil
	}()

	// try local
	if err != nil {
		list, err = r.Store.List(ctx, options)
		if err != nil {
			return nil, errors.NewInternalError(err)
		}
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) at local", path)
	}

	// decorate before return
	info, ok := apirequest.RequestInfoFrom(ctx)
	if ok && list.GetObjectKind().GroupVersionKind().Empty() {
		gvk := schema.GroupVersionKind{
			Group:   info.APIGroup,
			Version: info.APIVersion,
			Kind:    util.UnsafeResourceToKind(info.Resource) + "List",
		}
		list.GetObjectKind().SetGroupVersionKind(gvk)
	}
	return list, err
}

func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	path := info.Path
	// try remote
	_, err := func() (runtime.Object, error) {
		selector := application.LabelFieldSelector{Label: options.LabelSelector, Field: options.FieldSelector}
		app := r.Agent.Generate(ctx, application.Watch, *options, selector, nil)
		err := r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to apply for a watch listener from cloud, %v", err)
			return nil, errors.NewInternalError(err)
		}
		klog.Infof("[metaserver/reststorage] successfully apply for a watch listener (%v) through cloud", path)
		return nil, nil
	}()
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to get a approved application for watch(%v) from cloud application center, %v", path, err)
		// do not return here, although err occurs, we can still get watch event if a watch application is approved before,
	}

	return r.Store.Watch(ctx, options)
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	app := r.Agent.Generate(ctx, "delete", *options, application.LabelFieldSelector{}, nil)
	if err := r.Agent.Apply(app); err != nil {
		return nil, err
	}

	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, err
	}
	return retObj, nil
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	app := r.Agent.Generate(ctx, "delete", *options, application.LabelFieldSelector{}, nil)
	if err := r.Agent.Apply(app); err != nil {
		return nil, false, err
	}

	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, false, err
	}
	return retObj, true, nil
}

func (r *REST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, err := objInfo.UpdatedObject(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	app := r.Agent.Generate(ctx, "get", *options, application.LabelFieldSelector{}, obj)
	if err := r.Agent.Apply(app); err != nil {
		return nil, false, err
	}
	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, false, err
	}
	return retObj, false, nil
}
