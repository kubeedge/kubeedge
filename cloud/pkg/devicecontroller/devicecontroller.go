package devicecontroller

import (
	"os"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
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

func Register(dc *v1alpha1.DeviceController, kubeAPIConfig *v1alpha1.KubeAPIConfig) {
	config.InitConfigure(dc, kubeAPIConfig)
	core.Register(newDeviceController(dc.Enable))
}

// Name of controller
func (dc *DeviceController) Name() string {
	return modules.DeviceControllerModuleName
}

// Group of controller
func (dc *DeviceController) Group() string {
	return modules.DeviceControllerModuleGroup
}

// Enable indicates whether enable this module
func (dc *DeviceController) Enable() bool {
	return dc.enable
}

// Start controller
func (dc *DeviceController) Start() {
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
