package edgecontroller

import (
	"context"
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
)

// EdgeController use beehive context message layer
type EdgeController struct {
	cancel context.CancelFunc
	ctx    context.Context
}

func newEdgeController() *EdgeController {
	ctx, cancel := context.WithCancel(context.Background())
	return &EdgeController{
		cancel: cancel,
		ctx:    ctx,
	}
}

func Register() {
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
	initConfig()

	upstream, err := controller.NewUpstreamController(ctl.ctx)
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	downstream, err := controller.NewDownstreamController(ctl.ctx)
	if err != nil {
		klog.Warningf("new downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	downstream.Start()
}

// Cancel controller
func (ctl *EdgeController) Cancel() {
	ctl.cancel()
}

func initConfig() {
	config.InitBufferConfig()
	config.InitContextConfig()
	config.InitKubeConfig()
	config.InitLoadConfig()
	config.InitMessageLayerConfig()
}
