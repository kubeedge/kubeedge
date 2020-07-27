package local

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

func NewLocalProxy(cacheMgr *cache.CacheMgr, checker checker.Checker) *LocalProxy {
	return &LocalProxy{cacheMgr: cacheMgr, checker: checker}
}

type LocalProxy struct {
	cacheMgr *cache.CacheMgr
	checker  checker.Checker
}

func (l *LocalProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	requinfo, _ := apirequest.RequestInfoFrom(ctx)
	klog.V(4).Infof("serve request %v from local server!", req)
	switch requinfo.Verb {
	case "watch":
		l.watch(w, req)
	case "list":
		l.list(w, req)
	case "get":
		l.get(w, req)
	case "delete", "deletecollection":
		l.forbidden(w, req)
	default:
		l.get(w, req)
	}
}

func (l *LocalProxy) forbidden(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	info, _ := apirequest.RequestInfoFrom(ctx)
	klog.V(4).Infof("reqest verb %s doesn't support by local server", info.Verb)
	qualitiedResource := schema.GroupResource{
		Group:    info.APIGroup,
		Resource: info.Resource,
	}
	s := errors.NewForbidden(qualitiedResource, info.Name, fmt.Errorf("don't support delete opetion in local mode"))
	l.Err(s, w, req)
}

func (l *LocalProxy) watch(w http.ResponseWriter, req *http.Request) {
	opts := metainternalversion.ListOptions{}
	err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metainternalversion.SchemeGroupVersion, &opts)
	if err != nil {
		l.Err(err, w, req)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		err := fmt.Errorf("unable to start watch - can't get http.Flusher: %#v", w)
		utilruntime.HandleError(err)
		l.Err(err, w, req)
		return
	}
	ctx := req.Context()
	contentType, _ := util.GetReqContentType(ctx)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	timeout := time.Duration(10) * time.Minute
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	watchTimeout := time.NewTimer(timeout)
	checkInterval := time.NewTicker(time.Duration(2) * time.Second)
	for {
		select {
		case <-watchTimeout.C:
			return
		case <-checkInterval.C:
			if l.checker.Check() {
				return
			}
		}
	}
}

func (l *LocalProxy) list(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqinfo, _ := apirequest.RequestInfoFrom(ctx)
	listKind := util.GetReourceList(reqinfo.Resource)
	gkv := schema.GroupVersionKind{
		Group:   reqinfo.APIGroup,
		Version: reqinfo.APIVersion,
		Kind:    listKind,
	}
	ua, _ := util.GetAppUserAgent(ctx)
	objs, err := l.cacheMgr.QueryList(ctx, ua, reqinfo.Resource, reqinfo.Namespace)
	if err != nil {
		l.Err(err, w, req)
		return
	}
	listRv := 0
	accessor := meta.NewAccessor()
	for i := range objs {
		rvStr, _ := accessor.ResourceVersion(objs[i])
		rvInt, _ := strconv.Atoi(rvStr)
		if rvInt > listRv {
			listRv = rvInt
		}
	}
	listobj, err := scheme.Scheme.New(gkv)
	if err != nil {
		l.Err(err, w, req)
		return
	}
	accessor.SetResourceVersion(listobj, strconv.Itoa(listRv))
	clusterScoped := true
	if reqinfo.Namespace != "" {
		clusterScoped = false
	}

	prefix := "/" + path.Join(reqinfo.APIPrefix, reqinfo.APIGroup, reqinfo.APIVersion)
	namer := handlers.ContextBasedNaming{
		SelfLinker:         runtime.SelfLinker(meta.NewAccessor()),
		SelfLinkPathPrefix: path.Join(prefix, reqinfo.Resource) + "/",
		SelfLinkPathSuffix: "",
		ClusterScoped:      clusterScoped,
	}

	uri, err := namer.GenerateListLink(req)
	if err != nil {
		l.Err(err, w, req)
		return
	}
	if err := namer.SetSelfLink(listobj, uri); err != nil {
		l.Err(err, w, req)
		return
	}
	meta.SetList(listobj, objs)
	l.WriteObject(http.StatusOK, listobj, w, req)
}

func (l *LocalProxy) get(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqinfo, _ := apirequest.RequestInfoFrom(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	//TODO  cannot support create events craete operation。
	if reqinfo.Resource == "events" {
		l.forbidden(w, req)
		return
	}
	obj, err := l.cacheMgr.QueryObj(ctx, ua, reqinfo.Resource, reqinfo.Namespace, reqinfo.Name)
	if err != nil {
		l.Err(err, w, req)
		return
	}
	l.WriteObject(http.StatusOK, obj, w, req)
}
