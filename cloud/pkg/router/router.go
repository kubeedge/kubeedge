package router

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	routerconfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/provider/eventbus"   // init eventbus
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/provider/rest"       // init rest
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/provider/servicebus" // init servicebus
	_ "github.com/kubeedge/kubeedge/cloud/pkg/router/rule"                // init rule
)

type router struct {
	enable bool
}

var _ core.Module = (*router)(nil)

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
	return modules.RouterModuleName
}

func (r *router) Group() string {
	return modules.RouterGroupName
}

func (r *router) Enable() bool {
	return r.enable
}

func (r *router) Start() {
	klog.Info("In router module, start...")
	listener.Process(r.Name())
}
