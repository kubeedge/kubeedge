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
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/agent"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// REST implements a RESTStorage for all resource against imitator.
type REST struct {
	*genericregistry.Store
	*agent.Agent
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

	store.Storage.Storage = sqlite.New()
	store.Storage.Codec = unstructured.UnstructuredJSONScheme

	return &REST{store, agent.DefaultAgent}, nil
}

// decorateList set list's gvk if it's gvk is empty
func decorateList(ctx context.Context, list runtime.Object) {
	info, ok := apirequest.RequestInfoFrom(ctx)
	if ok && list.GetObjectKind().GroupVersionKind().Empty() {
		gvk := schema.GroupVersionKind{
			Group:   info.APIGroup,
			Version: info.APIVersion,
			Kind:    util.UnsafeResourceToKind(info.Resource) + "List",
		}
		list.GetObjectKind().SetGroupVersionKind(gvk)
	}
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	// First try to get the object from remote cloud
	obj, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Get, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to get obj from cloud: %v", err)
			return nil, err
		}
		var obj = new(unstructured.Unstructured)
		err = json.Unmarshal(app.RespBody, obj)
		if err != nil {
			return nil, err
		}
		// save to local, ignore error
		imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), obj)
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) through cloud", info.Path)
		return obj, nil
	}()

	// If we get object from cloud failed and RequireAuthorization FeatureGate
	// is not enabled, try to get the object from the local metaManager
	if err != nil && !kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		obj, err = r.Store.Get(ctx, "", options) // name is needless, we get all key information from ctx
		if err != nil {
			return nil, errors.NewNotFound(schema.GroupResource{Group: info.APIGroup, Resource: info.Resource}, info.Name)
		}
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) at local", info.Path)
	}
	return obj, err
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	// First try to list the object from remote cloud
	list, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.List, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to list obj from cloud: %v", err)
			return nil, err
		}
		var list = new(unstructured.UnstructuredList)
		err = json.Unmarshal(app.RespBody, list)
		if err != nil {
			return nil, err
		}
		// imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), list)
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) through cloud", info.Path)
		return list, nil
	}()

	// If we list object from cloud failed and RequireAuthorization FeatureGate
	// is not enabled, try to list the object from the local metaManager
	if err != nil && !kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		list, err = r.Store.List(ctx, options)
		if err != nil {
			return nil, err
		}
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) at local", info.Path)
	}

	decorateList(ctx, list)
	return list, err
}

func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)

	// First try watch from remote cloud
	_, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Watch, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		// For watch long connection request, we close the application when the watch is closed.
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to apply for a watch listener from cloud: %v", err)
			app.Close()
			return nil, errors.NewInternalError(err)
		}

		ctx = util.WithApplicationID(ctx, app.ID)
		klog.Infof("[metaserver/reststorage] successfully apply for a watch listener (%v) through cloud", info.Path)
		return nil, nil
	}()

	// If we watch object from cloud failed and RequireAuthorization FeatureGate
	// is enabled, just return the err
	if err != nil && kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		klog.Errorf("[metaserver/reststorage] failed to get a approved application for watch(%v) from cloud application center, %v", info.Path, err)
		return nil, err
	}

	return r.Store.Watch(ctx, options)
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	obj, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Create, *options, obj)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to create obj: %v", err)
			return nil, err
		}

		retObj := new(unstructured.Unstructured)
		if err := json.Unmarshal(app.RespBody, retObj); err != nil {
			return nil, err
		}
		return retObj, nil
	}()

	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to create (%v)", metaserver.KeyFunc(obj))
		return nil, err
	}

	klog.Infof("[metaserver/reststorage] successfully create (%v)", metaserver.KeyFunc(obj))
	return obj, nil
}

func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	key, _ := metaserver.KeyFuncReq(ctx, "")
	app, err := r.Agent.Generate(ctx, metaserver.Delete, options, nil)
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, false, err
	}
	err = r.Agent.Apply(app)
	defer app.Close()
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to delete (%v) through cloud", key)
		return nil, false, err
	}
	klog.Infof("[metaserver/reststorage] successfully delete (%v) through cloud", key)
	return nil, true, nil
}

func (r *REST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, err := objInfo.UpdatedObject(ctx, nil)
	if err != nil {
		return nil, false, errors.NewInternalError(err)
	}

	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	var app *metaserver.Application
	if reqInfo.Subresource == "status" {
		app, err = r.Agent.Generate(ctx, metaserver.UpdateStatus, options, obj)
	} else {
		app, err = r.Agent.Generate(ctx, metaserver.Update, options, obj)
	}
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, false, err
	}
	defer app.Close()
	if err := r.Agent.Apply(app); err != nil {
		return nil, false, err
	}
	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, false, errors.NewInternalError(err)
	}
	return retObj, false, nil
}

func (r *REST) Patch(ctx context.Context, pi metaserver.PatchInfo) (runtime.Object, error) {
	app, err := r.Agent.Generate(ctx, metaserver.Patch, pi, nil)
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, err
	}
	defer app.Close()
	if err := r.Agent.Apply(app); err != nil {
		return nil, err
	}
	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, errors.NewInternalError(err)
	}
	return retObj, nil
}
