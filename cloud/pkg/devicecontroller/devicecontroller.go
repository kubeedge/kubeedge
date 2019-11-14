package devicecontroller

import (
	"context"
	"os"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
)

// DeviceController use beehive context message layer
type DeviceController struct {
	cancel context.CancelFunc
}

func Register() {
	deviceController := DeviceController{}
	core.Register(&deviceController)
}

// Name of controller
func (dctl *DeviceController) Name() string {
	return constants.DeviceControllerModuleName
}

// Group of controller
func (dctl *DeviceController) Group() string {
	return constants.DeviceControllerModuleGroup
}

// Start controller
func (dctl *DeviceController) Start() {
	var ctx context.Context

	ctx, dctl.cancel = context.WithCancel(context.Background())

	initConfig()

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		klog.Errorf("New downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}

	downstream.Start(ctx)
	// wait for downstream controller to start and load deviceModels and devices
	// TODO think about sync
	time.Sleep(1 * time.Second)
	upstream.Start(ctx)
}

// Cleanup controller
func (dctl *DeviceController) Cleanup() {
	dctl.cancel()
}

func initConfig() {
	config.InitBufferConfig()
	config.InitContextConfig()
	config.InitKubeConfig()
	config.InitLoadConfig()
	config.InitMessageLayerConfig()
}
