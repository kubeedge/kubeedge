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
	ctx    context.Context
}

func newDeviceController() *DeviceController {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeviceController{
		cancel: cancel,
		ctx:    ctx,
	}
}

func Register() {
	core.Register(newDeviceController())
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
	initConfig()

	downstream, err := controller.NewDownstreamController(dctl.ctx)
	if err != nil {
		klog.Errorf("New downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream, err := controller.NewUpstreamController(downstream, dctl.ctx)
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}

	downstream.Start()
	// wait for downstream controller to start and load deviceModels and devices
	// TODO think about sync
	time.Sleep(1 * time.Second)
	upstream.Start()
}

// Cancel controller
func (dctl *DeviceController) Cancel() {
	dctl.cancel()
}

func initConfig() {
	config.InitBufferConfig()
	config.InitContextConfig()
	config.InitKubeConfig()
	config.InitLoadConfig()
	config.InitMessageLayerConfig()
}
