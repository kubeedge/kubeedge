package edgecontroller

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// EdgeController use beehive context message layer
type EdgeController struct {
	enable bool
}

func newEdgeController(enable bool) *EdgeController {
	return &EdgeController{
		enable: enable,
	}
}

func Register(ec *v1alpha1.EdgeController, kubeAPIConfig *v1alpha1.KubeAPIConfig, nodeName string, edgesite bool) {
	// TODO move module config into EdgeController struct @kadisi
	config.InitConfigure(ec, kubeAPIConfig, nodeName, edgesite)
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
	upstream, err := controller.NewUpstreamController()
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}

	if err := upstream.Start(); err != nil {
		klog.Fatalf("start upstream failed with error: %s", err)
	}

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		klog.Fatalf("new downstream controller failed with error: %s", err)
	}

	if err := downstream.Start(); err != nil {
		klog.Fatalf("start downstream failed with error: %s", err)
	}
}
