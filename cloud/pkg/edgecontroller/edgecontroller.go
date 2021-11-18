package edgecontroller

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// EdgeController use beehive context message layer
type EdgeController struct {
	config     v1alpha1.EdgeController
	upstream   *controller.UpstreamController
	downstream *controller.DownstreamController
}

var _ core.Module = (*EdgeController)(nil)

func newEdgeController(config *v1alpha1.EdgeController, tunnelPort int) *EdgeController {
	ec := &EdgeController{config: *config}
	if !ec.Enable() {
		return ec
	}
	var err error
	ec.upstream, err = controller.NewUpstreamController(config, informers.GetInformersManager().GetK8sInformerFactory())
	if err != nil {
		klog.Exitf("new upstream controller failed with error: %s", err)
	}
	ec.upstream.TunnelPort = tunnelPort

	ec.downstream, err = controller.NewDownstreamController(config, informers.GetInformersManager().GetK8sInformerFactory(), informers.GetInformersManager(), informers.GetInformersManager().GetCRDInformerFactory())
	if err != nil {
		klog.Exitf("new downstream controller failed with error: %s", err)
	}
	return ec
}

func Register(ec *v1alpha1.EdgeController, commonConfig *v1alpha1.CommonConfig) {
	core.Register(newEdgeController(ec, commonConfig.TunnelPort))
}

// Name of controller
func (ec *EdgeController) Name() string {
	return modules.EdgeControllerModuleName
}

// Group of controller
func (ec *EdgeController) Group() string {
	return modules.EdgeControllerGroupName
}

// Enable indicates whether enable this module
func (ec *EdgeController) Enable() bool {
	return ec.config.Enable
}

// Start controller
func (ec *EdgeController) Start() {
	if err := ec.upstream.Start(); err != nil {
		klog.Exitf("start upstream failed with error: %s", err)
	}

	if err := ec.downstream.Start(); err != nil {
		klog.Exitf("start downstream failed with error: %s", err)
	}
}
