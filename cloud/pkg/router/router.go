package router

import (
	"github.com/kubeedge/beehive/pkg/core"
	routerconfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/provider/eventbus"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/provider/rest"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/rule"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"k8s.io/klog/v2"
)

type router struct {
	enable bool
}

func newRouter(enable bool) *router {
	return &router{
		enable: enable,
	}
}

func Register(router *v1alpha1.Router) {
	routerconfig.InitConfigure(router)
	core.Register(newRouter(router.Enable))
}

func (r *router) Name() string {
	return "router"
}

func (r *router) Group() string {
	return "router"
}

func (r *router) Enable() bool {
	return r.enable
}

func (r *router) Start() {
	klog.Info("In router module, start...")
	listener.Process(r.Name())
}
