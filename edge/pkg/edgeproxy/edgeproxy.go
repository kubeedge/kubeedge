package edgeproxy

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/decoder"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/proxy"
	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/server"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const EdgeProxyModuleName = "edgeproxy"

func Register(ep *v1alpha1.EdgePorxy) {
	config.InitConfigure(ep)
	epModule := newEdgeProxy(ep.Enable)
	core.Register(epModule)
	cache.InitDBTable(epModule)
}

func newEdgeProxy(enable bool) *edgeProxy {
	return &edgeProxy{
		enable: enable,
	}
}

type edgeProxy struct {
	enable bool
}

func (e *edgeProxy) Name() string {
	return EdgeProxyModuleName
}

func (e *edgeProxy) Group() string {
	return modules.ProxyGroup
}

func (e *edgeProxy) Start() {
	decoderMgr := decoder.DefaultDecoderMgr
	cacheMgr := cache.NewCacheMgr(decoderMgr)
	c := checker.NewHealthzChecker(config.Config.RemoteAddr)
	//c := checker.AlwaysFlase{}
	eph, err := proxy.NewEdgeProxyHandler(cacheMgr, c)
	if err != nil {
		panic(err)
	}
	svr, err := server.NewProxyServer(eph)
	if err != nil {
		panic(err)
	}
	svr.Run()
}

func (e *edgeProxy) Enable() bool {
	return e.enable
}
