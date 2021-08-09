package devicecontroller

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// DeviceController use beehive context message layer
type DeviceController struct {
	downstream *controller.DownstreamController
	upstream   *controller.UpstreamController
	enable     bool
	size       int
}

func newDeviceController(cfg *v1alpha1.DeviceController) *DeviceController {
	if !cfg.Enable {
		return &DeviceController{enable: cfg.Enable}
	}
	downstream, err := controller.NewDownstreamController(informers.GetInformersManager().GetCRDInformerFactory())
	if err != nil {
		klog.Fatalf("New downstream controller failed with error: %s", err)
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		klog.Fatalf("new upstream controller failed with error: %s", err)
	}
	return &DeviceController{
		downstream: downstream,
		upstream:   upstream,
		enable:     cfg.Enable,
		size:       cfg.Size,
	}
}

func Register(dc *v1alpha1.DeviceController) {
	config.InitConfigure(dc)
	core.Register(newDeviceController(dc))
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

func (dc *DeviceController) Size() int {
	return dc.size
}

// Start controller
func (dc *DeviceController) Start() {
	if err := dc.downstream.Start(); err != nil {
		klog.Fatalf("start downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load deviceModels and devices
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := dc.upstream.Start(); err != nil {
		klog.Fatalf("start upstream failed with error: %s", err)
	}
}
