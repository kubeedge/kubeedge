package remote

import (
	"compress/gzip"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

func NewRemoteProxy(remote *url.URL, cacheMgr cache.Manager) *Proxy {
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
	cacheMgr cache.Manager
}

func (r *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	r.proxy.ServeHTTP(writer, request)
}

func (r *Proxy) modifyResponse(resp *http.Response) error {
	req := resp.Request
	ctx := req.Context()
	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	if !util.CanCacheResource(reqInfo.Resource) {
		return nil
	}
	// Store Resoponse Content-Type Header information to the context
	// to support mime types with addition detail,such as application/vnd.kubernetes.protobuf;stream=watch
	respContentType := resp.Header.Get("Content-Type")
	parts := strings.Split(respContentType, ",")
	if len(parts) >= 1 {
		respContentType = parts[0]
	}
	parts = strings.Split(respContentType, ";")
	if len(parts) >= 1 {
		respContentType = parts[0]
	}
	ctx = util.WithRespContentType(ctx, respContentType)
	// Store Resoponse Content-Encoding Header information to the context, k8s apiserver automatically enables gzip compression when the response content greater than 128k
	algo := resp.Header.Get("Content-Encoding")
	ctx = util.WithRespContentEncoding(ctx, algo)
	req = req.WithContext(ctx)
	// get http code range from https://github.com/kubernetes/kubernetes/blob/release-1.19/staging/src/k8s.io/client-go/rest/request.go#L1044
	klog.V(4).Infof("cache request %v", req)
	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusPartialContent {
		source := resp.Body
		wrapped := util.NewDuplicateReadCloser(source)
		// cache response content according to the reqestInfo.Verb
		go func() {
			var err error
			drc := wrapped.DupReadCloser()
			//When the response uses gzip compression, use gzip.Reader to read the response body
			algo, _ := util.GetRespContentEncoding(ctx)
			if algo == "gzip" {
				drc, err = gzip.NewReader(drc)
				if err != nil {
					klog.Errorf("compress response body error! %v", err)
					return
				}
			}
			switch reqInfo.Verb {
			case "list":
				err = r.cacheMgr.CacheListObj(ctx, drc)
			case "get":
				err = r.cacheMgr.CacheObj(ctx, drc)
			case "watch":
				err = r.cacheMgr.CacheWatchObj(ctx, drc)
			}
			if err != nil {
				klog.Errorf("req %v cache resp error: %v", req, err)
			}
		}()
		resp.Body = wrapped
	}
	return nil
}
