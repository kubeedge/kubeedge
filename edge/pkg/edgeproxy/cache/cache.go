package cache

import (
	"bytes"
	"context"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/serializer"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

// Manager interface provides methods for EdgeProxy to cache http req/resp.
type Manager interface {
	// intercept the list response result, split result into individual object and store them in cache table
	CacheListObj(ctx context.Context, rc io.ReadCloser) error
	// intercept the list response result, and store result in cache table
	CacheObj(ctx context.Context, rc io.ReadCloser) error
	// intercept the list response result, and store result in cache table
	CacheWatchObj(ctx context.Context, rc io.ReadCloser) error
	//query all cached data that meets the conditions from the cache table and deserialize it into runtime.Object
	QueryList(ctx context.Context, ua, resource, namespace string) ([]runtime.Object, error)
	// query the cached data that meets the conditions from the cache table and deserialize it into runtime.Object
	QueryObj(ctx context.Context, ua, resource, namespace, name string) (runtime.Object, error)
}

type Mgr struct {
	decoderMgr        serializer.Manager
	backendSerializer runtime.Serializer
}

func NewCacheMgr(decoderMgr serializer.Manager) Manager {
	backendSerializer := decoderMgr.GetBackendSerializer()
	return &Mgr{
		decoderMgr:        decoderMgr,
		backendSerializer: backendSerializer,
	}
}

func (cm *Mgr) CacheListObj(ctx context.Context, rc io.ReadCloser) error {
	var err error

	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	deleteCache(ua, reqInfo.Resource, reqInfo.Namespace, "")
	contentType, _ := util.GetRespContentType(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}

	decoder, err := cm.decoderMgr.GetDecoder(contentType, gv)
	if err != nil {
		return err
	}
	apiVersion := gv.String()
	accessor := meta.NewAccessor()
	var buf bytes.Buffer
	n, err := buf.ReadFrom(rc)
	if err != nil {
		return err
	} else if n == 0 {
		klog.Warningf("response length is 0!")
		return nil
	}
	listObj, _, err := decoder.Decode(buf.Bytes(), nil, nil)
	if err != nil {
		return err
	}

	gr := schema.GroupResource{
		Group:    reqInfo.APIGroup,
		Resource: reqInfo.Resource,
	}.String()
	kind := util.GetResourceKind(gr)
	meta.EachListItem(listObj, func(object runtime.Object) error {
		accessor.SetKind(object, kind)
		accessor.SetAPIVersion(object, apiVersion)
		if err := cm.cacheSingleObj(ctx, object); err != nil {
			return err
		}
		return nil
	})
	return nil
}

func (cm *Mgr) CacheObj(ctx context.Context, rc io.ReadCloser) error {
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	contentType, _ := util.GetRespContentType(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}
	decoder, err := cm.decoderMgr.GetDecoder(contentType, gv)
	if err != nil {
		return err
	}
	apiVersion := gv.String()
	accessor := meta.NewAccessor()
	var buf bytes.Buffer
	n, err := buf.ReadFrom(rc)
	if err != nil {
		return err
	} else if n == 0 {
		klog.Warningf("response length is 0!")
		return nil
	}
	obj, _, err := decoder.Decode(buf.Bytes(), nil, nil)
	if err != nil {
		return err
	}
	gr := schema.GroupResource{
		Group:    reqInfo.APIGroup,
		Resource: reqInfo.Resource,
	}.String()

	kind := util.GetResourceKind(gr)
	accessor.SetKind(obj, kind)
	accessor.SetAPIVersion(obj, apiVersion)
	cm.cacheSingleObj(ctx, obj)
	return nil
}

// store the single object into cache table.
func (cm *Mgr) cacheSingleObj(ctx context.Context, obj runtime.Object) error {
	var objbyte bytes.Buffer
	err := cm.backendSerializer.Encode(obj, &objbyte)
	if err != nil {
		return err
	}
	ua, _ := util.GetAppUserAgent(ctx)
	accessor := meta.NewAccessor()
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	name, _ := accessor.Name(obj)
	namesapce, _ := accessor.Namespace(obj)
	gr := schema.GroupResource{
		Group:    reqInfo.APIGroup,
		Resource: reqInfo.Resource,
	}.String()

	return insertOrUpdateCache(&Cache{
		UA:        ua,
		Resource:  gr,
		Namespace: namesapce,
		Name:      name,
		Value:     objbyte.Bytes(),
	})
}

func (cm *Mgr) CacheWatchObj(ctx context.Context, rc io.ReadCloser) error {
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	contentType, _ := util.GetRespContentType(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}
	apiVersion := gv.String()
	watchDecoder, err := cm.decoderMgr.GetStreamDecoder(contentType, gv, rc)
	if err != nil {
		return err
	}
	gr := schema.GroupResource{
		Group:    reqInfo.APIGroup,
		Resource: reqInfo.Resource,
	}.String()
	kind := util.GetResourceKind(gr)
	// The watch request is httpstream, cyclically to obtain the latest data
	for {
		watchType, obj, err := watchDecoder.Decode()
		if err != nil {
			switch err {
			case io.EOF:
				return nil
			case io.ErrUnexpectedEOF:
				klog.V(1).Infof("Unexpected EOF during watch stream event decoding: %v", err)
			default:
				klog.Errorf("cache watch obj error! %v", err)
				return err
			}
		}
		accessor := meta.NewAccessor()
		name, _ := accessor.Name(obj)
		namespace, _ := accessor.Namespace(obj)

		switch watchType {
		case watch.Modified, watch.Added:
			accessor.SetKind(obj, kind)
			accessor.SetAPIVersion(obj, apiVersion)
			var objbyte bytes.Buffer
			if err := cm.backendSerializer.Encode(obj, &objbyte); err != nil {
				return err
			}
			insertOrUpdateCache(&Cache{
				UA:        ua,
				Resource:  reqInfo.Resource,
				Namespace: namespace,
				Name:      name,
				Value:     objbyte.Bytes(),
			})
		case watch.Deleted:
			deleteCache(ua, reqInfo.Resource, namespace, name)
		case watch.Error:
			klog.Warningf("watch event type is watch.Error! %v", obj)
		}
	}
}

func (cm *Mgr) QueryList(ctx context.Context, ua, resource, namespace string) ([]runtime.Object, error) {
	liststr, err := queryCacheList(ua, resource, namespace)
	if err != nil {
		return nil, err
	}
	objs := make([]runtime.Object, 0)
	for i := range liststr {
		obj, _, err := cm.backendSerializer.Decode([]byte(liststr[i]), nil, nil)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func (cm *Mgr) QueryObj(ctx context.Context, ua, resource, namespace, name string) (runtime.Object, error) {
	objstr, err := queryCache(ua, resource, namespace, name)
	if err != nil {
		return nil, err
	}

	obj, _, err := cm.backendSerializer.Decode([]byte(objstr), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
