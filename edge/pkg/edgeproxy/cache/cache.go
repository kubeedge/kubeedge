package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/astaxie/beego/orm"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/decoder"

	"github.com/kubeedge/beehive/pkg/core"
	cachedao "github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

func InitDBTable(module core.Module) {
	if !module.Enable() {
		klog.Infof("Module %s is disabled, DB cache for it will not be registered", module.Name())
		return
	}
	orm.RegisterModel(new(cachedao.Cache))
}

func NewCacheMgr(decoderMgr decoder.Mgr) *Mgr {
	return &Mgr{
		decoderMgr: decoderMgr,
	}
}

type Mgr struct {
	decoderMgr decoder.Mgr
}

func (cm *Mgr) CacheListObj(ctx context.Context, rc io.ReadCloser) error {
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	cachedao.DeleteCache(ua, reqInfo.Resource, reqInfo.Namespace, "")
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
	meta.EachListItem(listObj, func(object runtime.Object) error {
		accessor.SetKind(object, util.GetResourceKind(reqInfo.Resource))
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
	accessor.SetKind(obj, util.GetResourceKind(reqInfo.Resource))
	accessor.SetAPIVersion(obj, apiVersion)
	cm.cacheSingleObj(ctx, obj)
	return nil
}

func (cm *Mgr) cacheSingleObj(ctx context.Context, obj runtime.Object) error {
	objbyte, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	ua, _ := util.GetAppUserAgent(ctx)
	accessor := meta.NewAccessor()
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	name, _ := accessor.Name(obj)
	namesapce, _ := accessor.Namespace(obj)
	cache := cachedao.Cache{
		UA:        ua,
		Resource:  reqInfo.Resource,
		Namespace: namesapce,
		Name:      name,
		Value:     string(objbyte),
	}
	err = cachedao.InsertOrUpdate(&cache)
	return err
}

func (cm *Mgr) CacheWatchObj(ctx context.Context, rc io.ReadCloser) error {
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	contentType, _ := util.GetRespContentType(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}
	watchDecoder, err := cm.decoderMgr.GetStreamDecocer(contentType, gv, rc)
	if err != nil {
		return err
	}
	for {
		watchType, obj, err := watchDecoder.Decode()
		if err != nil {
			klog.Errorf("cache watch obj error! %v", err)
			return err
		}
		accessor := meta.NewAccessor()
		name, _ := accessor.Name(obj)
		namespace, _ := accessor.Namespace(obj)
		switch watchType {
		case watch.Modified, watch.Added:
			objbyte, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			cache := cachedao.Cache{
				UA:        ua,
				Resource:  reqInfo.Resource,
				Namespace: namespace,
				Name:      name,
				Value:     string(objbyte),
			}
			cachedao.InsertOrUpdate(&cache)
		case watch.Deleted:
			cachedao.DeleteCache(ua, reqInfo.Resource, namespace, name)
		case watch.Error:
			klog.Warningf("watch event type is watch.Error! %v", obj)
		}
	}
}

func (cm *Mgr) QueryList(ctx context.Context, ua, resource, namespace string) ([]runtime.Object, error) {
	liststr, err := cachedao.QueryCacheList(ua, resource, namespace)
	if err != nil {
		return nil, err
	}
	objs := make([]runtime.Object, 0)
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	contentType, _ := util.GetReqContentType(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}
	objDecoder, err := cm.decoderMgr.GetDecoder(contentType, gv)
	if err != nil {
		return nil, err
	}
	for i := range liststr {
		obj, _, err := objDecoder.Decode([]byte(liststr[i]), nil, nil)
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func (cm *Mgr) QueryObj(ctx context.Context, ua, resource, namespace, name string) (runtime.Object, error) {
	objstr, err := cachedao.Query(ua, resource, namespace, name)
	if err != nil {
		return nil, err
	}
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	contentType, _ := util.GetReqContentType(ctx)
	gv := schema.GroupVersion{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
	}
	objDecoder, err := cm.decoderMgr.GetDecoder(contentType, gv)
	if err != nil {
		return nil, err
	}
	obj, _, err := objDecoder.Decode([]byte(objstr), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
