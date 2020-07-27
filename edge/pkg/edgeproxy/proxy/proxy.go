package proxy

import (
	"net/http"
	"net/url"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/proxy/local"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/proxy/remote"
)

func NewEdgeProxyHandler(cacheMgr *cache.Mgr, c checker.Checker) (*EdgeProxyHandler, error) {
	remoteURL, err := url.Parse(config.Config.RemoteAddr)
	if err != nil {
		return nil, err
	}
	r := remote.NewRemoteProxy(remoteURL, cacheMgr)
	l := local.NewLocalProxy(cacheMgr, c)
	eph := &EdgeProxyHandler{remote: r, local: l, checker: c}
	return eph, nil
}

type EdgeProxyHandler struct {
	remote  *remote.Proxy
	local   *local.Proxy
	checker checker.Checker
}

func (ep *EdgeProxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if ep.checker.Check() {
		klog.V(1).Info("proxy client request handle by remote server!")
		ep.remote.ServeHTTP(writer, request)
	} else {
		klog.V(1).Info("proxy client request handle by local server!")
		ep.local.ServeHTTP(writer, request)
	}
}
