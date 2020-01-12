package devicecontroller

import (
	"os"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

// DeviceController use beehive context message layer
type DeviceController struct {
	enable bool
}

func newDeviceController(enable bool) *DeviceController {
	return &DeviceController{
		enable: enable,
	}
}

func Register(c *v1alpha1.DeviceController, k *v1alpha1.KubeAPIConfig) {
	config.InitConfigure(c, k)
	core.Register(newDeviceController(c.Enable))
}

// Name of controller
func (dctl *DeviceController) Name() string {
	return constants.DeviceControllerModuleName
}

// Group of controller
func (dctl *DeviceController) Group() string {
	return constants.DeviceControllerModuleGroup
}

// Enable indicates whether enable this module
func (dctl *DeviceController) Enable() bool {
	return dctl.enable
}

// Start controller
func (dctl *DeviceController) Start() {
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

	downstream.Start()
	// wait for downstream controller to start and load deviceModels and devices
	// TODO think about sync
	time.Sleep(1 * time.Second)
	upstream.Start()
}
