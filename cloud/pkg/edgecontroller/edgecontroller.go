package edgecontroller

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// EdgeController use beehive context message layer
type EdgeController struct {
	enable     bool
	upstream   *controller.UpstreamController
	downstream *controller.DownstreamController
}

func newEdgeController(enable bool) *EdgeController {
	upstream, err := controller.NewUpstreamController(informers.GetInformersManager().GetK8sInformerFactory())
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	downstream, err := controller.NewDownstreamController(informers.GetInformersManager().GetK8sInformerFactory(), informers.GetInformersManager())
	if err != nil {
		klog.Fatalf("new downstream controller failed with error: %s", err)
	}
	return &EdgeController{
		enable:     enable,
		upstream:   upstream,
		downstream: downstream,
	}
}

func Register(ec *v1alpha1.EdgeController) {
	// TODO move module config into EdgeController struct @kadisi
	config.InitConfigure(ec)
	core.Register(newEdgeController(ec.Enable))
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
	return ec.enable
}

// Start controller
func (ec *EdgeController) Start() {
	if err := ec.upstream.Start(); err != nil {
		klog.Fatalf("start upstream failed with error: %s", err)
	}

	if err := ec.downstream.Start(); err != nil {
		klog.Fatalf("start downstream failed with error: %s", err)
	}
}
