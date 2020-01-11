package edgecontroller

import (
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

// EdgeController use beehive context message layer
type EdgeController struct {
}

func newEdgeController() *EdgeController {
	return &EdgeController{}
}

func Register(c *v1alpha1.EdgeController, k *v1alpha1.KubeAPIConfig, nodeName string, edgesite bool) {
	config.InitConfigure(c, k, nodeName, edgesite)
	core.Register(newEdgeController())
}

// Name of controller
func (ctl *EdgeController) Name() string {
	return constants.EdgeControllerModuleName
}

// Group of controller
func (ctl *EdgeController) Group() string {
	return constants.EdgeControllerModuleName
}

// Start controller
func (ctl *EdgeController) Start() {
	upstream, err := controller.NewUpstreamController()
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		klog.Warningf("new downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	downstream.Start()
}
