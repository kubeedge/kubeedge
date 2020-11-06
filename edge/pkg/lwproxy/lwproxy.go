package lwproxy

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/cache"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/checker"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/proxy"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/relation"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/serializer"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/server"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const LWProxyModuleName = "lwproxy"

func Register(ep *v1alpha1.LWPorxy) {
	config.InitConfigure(ep)
	epModule := newLWProxy(ep.Enable)
	core.Register(epModule)
	relation.Init(epModule)
	cache.InitCacheDataPathPrefix(ep.CacheDataPath)
}

func newLWProxy(enable bool) *lwProxy {
	return &lwProxy{
		enable: enable,
	}
}

type lwProxy struct {
	enable bool
}

func (e *lwProxy) Name() string {
	return LWProxyModuleName
}

func (e *lwProxy) Group() string {
	return modules.LWProxyGroup
}

func (e *lwProxy) Start() {
	decoderMgr := serializer.DefaultDecoderMgr
	cacheMgr := cache.NewCacheMgr(decoderMgr)
	healthzChecker := checker.NewHealthzChecker(config.Config.RemoteAddr)
	eph, err := proxy.NewLWProxyHandler(cacheMgr, healthzChecker)
	if err != nil {
		panic(err)
	}
	svr, err := server.NewProxyServer(eph)
	if err != nil {
		panic(err)
	}
	relation.InitMgr()
	svr.Run()
}

func (e *lwProxy) Enable() bool {
	return e.enable
}
