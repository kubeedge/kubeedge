package sqlite

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	patchutil "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/util"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// ErrCountUnsupported is returned by Count to signal that the operation is
// not implemented by this SQLite-backed store. Callers can test for it with
// errors.Is.
var ErrCountUnsupported = errors.New("Count is not supported by the edge SQLite store")

/*
This file is designed to encapsulate the Imitator as Store.Interface,
*/
type store struct {
	client    imitator.Client
	versioner storage.Versioner
	codec     runtime.Codec
	watcher   *watcher
}

func (s *store) Versioner() storage.Versioner {
	return s.versioner
}

func (s *store) Create(context.Context, string, runtime.Object, runtime.Object, uint64) error {
	panic("Do not call this function")
}

func (s *store) Delete(ctx context.Context, key string, out runtime.Object, preconditions *storage.Preconditions,
	validateDeletion storage.ValidateObjectFunc, cachedExistingObject runtime.Object, opts storage.DeleteOptions) error {
	panic("Do not call this function")
}

func (s *store) Watch(ctx context.Context, key string, opts storage.ListOptions) (watch.Interface, error) {
	return s.watch(ctx, key, opts, false)
}

func (s *store) watch(ctx context.Context, key string, opts storage.ListOptions, recursive bool) (watch.Interface, error) {
	rev, err := s.versioner.ParseResourceVersion(opts.ResourceVersion)
	if err != nil {
		return nil, err
	}
	return s.watcher.Watch(ctx, key, int64(rev), recursive, opts.Predicate)
}

func (s *store) Get(ctx context.Context, key string, _ storage.GetOptions, objPtr runtime.Object) error {
	resp, err := s.client.Get(context.TODO(), key)
	if err != nil || len(*resp.Kvs) == 0 {
		klog.Error(err)
		return err
	}
	return patchutil.DecodeStored(s.codec, (*resp.Kvs)[0].Value, objPtr.(*unstructured.Unstructured))
}

func (s *store) GetList(ctx context.Context, key string, opts storage.ListOptions, listObj runtime.Object) error {
	klog.Infof("get a list req, key=%v", key)
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return fmt.Errorf("need ptr to slice: %v", err)
	}

	resp, err := s.client.List(context.TODO(), key)

	if err != nil || len(*resp.Kvs) == 0 {
		klog.Error(err)
		return err
	}
	unstrList := listObj.(*unstructured.UnstructuredList)
	for _, v := range *resp.Kvs {
		var unstrObj unstructured.Unstructured
		if err := patchutil.DecodeStored(s.codec, v.Value, &unstrObj); err != nil {
			return err
		}

		labelSet := labels.Set(unstrObj.GetLabels())
		if !opts.Predicate.Label.Matches(labelSet) {
			continue
		}

		// only support metadata.name & metadata.namespace
		fieldSet := fields.Set{
			"metadata.name":      unstrObj.GetName(),
			"metadata.namespace": unstrObj.GetNamespace(),
		}
		if !opts.Predicate.Field.Matches(fieldSet) {
			continue
		}

		unstrList.Items = append(unstrList.Items, unstrObj)
	}
	rv := strconv.FormatUint(resp.Revision, 10)
	unstrList.SetResourceVersion(rv)
	unstrList.SetSelfLink(key)
	gvr, _, _ := metaserver.ParseKey(key)
	unstrList.SetGroupVersionKind(gvr.GroupVersion().WithKind(util.UnsafeResourceToKind(gvr.Resource) + "List"))
	return nil
}

func (s *store) GuaranteedUpdate(context.Context, string, runtime.Object, bool, *storage.Preconditions, storage.UpdateFunc, runtime.Object) error {
	panic("Do not call this function")
}

func (s *store) Count(_ string) (int64, error) {
	return 0, ErrCountUnsupported
}

func (s *store) RequestWatchProgress(context.Context) error {
	panic("Do not call this function")
}

func (s *store) ReadinessCheck() error {
	return nil
}

func New() storage.Interface {
	return newStore()
}
func newStore() *store {
	codec := unstructured.UnstructuredJSONScheme
	client := imitator.DefaultV2Client
	s := store{
		client:    client,
		versioner: imitator.Versioner,
		watcher:   newWatcher(client, codec),
		codec:     codec,
	}
	return &s
}
