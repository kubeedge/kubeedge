package local

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"net/http"
	"path"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

func NewLocalProxy(cacheMgr cache.Manager, checker checker.Checker) *Proxy {
	return &Proxy{cacheMgr: cacheMgr, checker: checker}
}

type Proxy struct {
	cacheMgr cache.Manager
	checker  checker.Checker
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	if !util.CanRespResource(reqInfo.Resource) {
		p.forbidden(w, req)
		return
	}
	klog.V(4).Infof("serve request %v from local server!", req)
	switch reqInfo.Verb {
	case "watch":
		p.watch(w, req)
	case "list":
		p.list(w, req)
	case "get":
		p.get(w, req)
	case "delete", "create", "deletecollection":
		p.forbidden(w, req)
	default:
		p.get(w, req)
	}
}

//Respond to operations that the local server cannot handle
func (p *Proxy) forbidden(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	info, _ := apirequest.RequestInfoFrom(ctx)
	klog.V(4).Infof("reqest verb %s doesn't support by local server", info.Verb)
	qualitiedResource := schema.GroupResource{
		Group:    info.APIGroup,
		Resource: info.Resource,
	}
	s := errors.NewForbidden(qualitiedResource, info.Name, fmt.Errorf("don't support delete opetion in local mode"))
	p.Err(s, w, req)
}

//Responding to client's watch operation。But No watch events wil be generated。
//when k8s apiserver is accessible, the method will be interrupted.
func (p *Proxy) watch(w http.ResponseWriter, req *http.Request) {
	opts := metainternalversion.ListOptions{}
	err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion, &opts)
	if err != nil {
		p.Err(err, w, req)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		err := fmt.Errorf("unable to start watch - can't get http.Flusher: %#v", w)
		utilruntime.HandleError(err)
		p.Err(err, w, req)
		return
	}
	ctx := req.Context()
	contentType, _ := util.GetReqContentType(ctx)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	defaultTimeoutSeconds := 0
	timeout := time.Duration(defaultTimeoutSeconds) * time.Second
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	watchTimeout := time.NewTimer(timeout)
	checkInterval := time.NewTicker(time.Duration(config.Config.HealthzCheckInterval) * time.Second)
	for {
		select {
		case <-watchTimeout.C:
			return
		case <-checkInterval.C:
			if p.checker.Check() {
				return
			}
		}
	}
}

//Responding to client's list operation
func (p *Proxy) list(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqinfo, _ := apirequest.RequestInfoFrom(ctx)
	listKind := util.GetReourceList(reqinfo.Resource)
	gkv := schema.GroupVersionKind{
		Group:   reqinfo.APIGroup,
		Version: reqinfo.APIVersion,
		Kind:    listKind,
	}
	ua, _ := util.GetAppUserAgent(ctx)
	objs, err := p.cacheMgr.QueryList(ctx, ua, reqinfo.Resource, reqinfo.Namespace)
	if err != nil {
		p.Err(err, w, req)
		return
	}
	listobj, err := scheme.Scheme.New(gkv)
	if err != nil {
		listobj = &unstructured.UnstructuredList{}
		listobj.GetObjectKind().SetGroupVersionKind(gkv)
	}
	// iterate objs to get the latest resourceversion
	listRv := 0
	accessor := meta.NewAccessor()
	for i := range objs {
		rvStr, _ := accessor.ResourceVersion(objs[i])
		rvInt, _ := strconv.Atoi(rvStr)
		if rvInt > listRv {
			listRv = rvInt
		}
	}
	accessor.SetResourceVersion(listobj, strconv.Itoa(listRv))
	// compute and set selflink of listobjs
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
		p.Err(err, w, req)
		return
	}
	if err := namer.SetSelfLink(listobj, uri); err != nil {
		p.Err(err, w, req)
		return
	}
	meta.SetList(listobj, objs)
	p.WriteObject(http.StatusOK, listobj, w, req)
}

//Responding to client's get operation
func (p *Proxy) get(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqinfo, _ := apirequest.RequestInfoFrom(ctx)
	ua, _ := util.GetAppUserAgent(ctx)
	//TODO  cannot support create events craete operation。
	if reqinfo.Resource == "events" {
		p.forbidden(w, req)
		return
	}
	obj, err := p.cacheMgr.QueryObj(ctx, ua, reqinfo.Resource, reqinfo.Namespace, reqinfo.Name)
	if err != nil {
		p.Err(err, w, req)
		return
	}
	p.WriteObject(http.StatusOK, obj, w, req)
}
