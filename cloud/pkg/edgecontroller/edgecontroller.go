package edgecontroller

import (
	"context"
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
)

// Controller use beehive context message layer
type Controller struct {
	cancel context.CancelFunc
}

func Register() {
	edgeController := Controller{}
	core.Register(&edgeController)
}

// Name of controller
func (ctl *Controller) Name() string {
	return constants.EdgeControllerModuleName
}

// Group of controller
func (ctl *Controller) Group() string {
	return constants.EdgeControllerModuleName
}

// Start controller
func (ctl *Controller) Start(c *beehiveContext.Context) {
	var ctx context.Context

	config.Context = c
	ctx, ctl.cancel = context.WithCancel(context.Background())

	initConfig()

	upstream, err := controller.NewUpstreamController()
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start(ctx)

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		klog.Warningf("new downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	downstream.Start(ctx)

}

// Cleanup controller
func (ctl *Controller) Cleanup() {
	ctl.cancel()
	config.Context.Cleanup(ctl.Name())
}

func initConfig() {
	config.InitBufferConfig()
	config.InitContextConfig()
	config.InitKubeConfig()
	config.InitLoadConfig()
	config.InitMessageLayerConfig()
}
