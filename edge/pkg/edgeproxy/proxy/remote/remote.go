package remote

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"k8s.io/klog"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

func NewRemoteProxy(remote *url.URL, cacheMgr *cache.Mgr) *Proxy {
	rp := &Proxy{
		proxy:    httputil.NewSingleHostReverseProxy(remote),
		cacheMgr: cacheMgr,
	}
	rp.proxy.ModifyResponse = rp.modifyResponse
	// flush response immediately
	rp.proxy.FlushInterval = -1
	rp.proxy.Transport = util.GetTransport()
	return rp
}

type Proxy struct {
	proxy    *httputil.ReverseProxy
	cacheMgr *cache.Mgr
}

func (r *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	r.proxy.ServeHTTP(writer, request)
}

func (r *Proxy) modifyResponse(resp *http.Response) error {
	req := resp.Request
	ctx := req.Context()
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	respContentType := resp.Header.Get("Content-Type")
	ctx = util.WithRespContentType(ctx, respContentType)
	req = req.WithContext(ctx)
	// get http code range from https://github.com/kubernetes/kubernetes/blob/release-1.19/staging/src/k8s.io/client-go/rest/request.go#L1044
	klog.V(4).Infof("cache request %v", req)
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusPartialContent {
		source := resp.Body
		wrapped := util.NewDuplicateReadCloser(source)
		go func() {
			var err error
			switch reqInfo.Verb {
			case "list":
				err = r.cacheMgr.CacheListObj(ctx, wrapped.DupData())
			case "get":
				err = r.cacheMgr.CacheObj(ctx, wrapped.DupData())
			case "watch":
				err = r.cacheMgr.CacheWatchObj(ctx, wrapped.DupData())
			}
			if err != nil {
				klog.Errorf("req %v cache resp error: %v", req, err)
			}
		}()
		resp.Body = wrapped
	}
	return nil
}
