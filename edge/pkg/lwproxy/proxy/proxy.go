package proxy

import (
	"net/http"
	"net/url"

	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/proxy/local"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/proxy/remote"
)

func NewLWProxyHandler(cacheMgr cache.Manager, c checker.Checker) (*LWHandler, error) {
	remoteURL, err := url.Parse(config.Config.RemoteAddr)
	if err != nil {
		return nil, err
	}
	r := remote.NewRemoteProxy(remoteURL, cacheMgr)
	l := local.NewLocalProxy(cacheMgr, c)
	eph := &LWHandler{remote: r, local: l, checker: c}
	return eph, nil
}

type LWHandler struct {
	remote  *remote.Proxy
	local   *local.Proxy
	checker checker.Checker
}

func (ep *LWHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if ep.checker.Check() {
		klog.V(1).Info("proxy client request handle by remote server!")
		ep.remote.ServeHTTP(writer, request)
	} else {
		klog.V(1).Info("proxy client request handle by local server!")
		ep.local.ServeHTTP(writer, request)
	}
}
