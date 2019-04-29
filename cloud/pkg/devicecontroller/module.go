package devicecontroller

import (
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/messagelayer"
)

// DeviceController use beehive context message layer
type DeviceController struct {
	messageLayer messagelayer.MessageLayer
	stopChan     chan bool
}

func init() {
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
func (dctl *DeviceController) Start(c *bcontext.Context) {
	config.Context = c
	dctl.stopChan = make(chan bool)
	downstream, err := controller.NewDownstreamController()
	if err != nil {
		log.LOGGER.Warnf("New downstream controller failed with error: %s", err)
		return
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		log.LOGGER.Warnf("new upstream controller failed with error: %s", err)
		return
	}
	downstream.Start()
	// wait for downstream controller to start and load deviceModels and devices
	time.Sleep(1 * time.Second)
	upstream.Start()

	<-dctl.stopChan
	log.LOGGER.Warnf("stop upstream controller")
	upstream.Stop()
	log.LOGGER.Warnf("upstream controller stopped")
	log.LOGGER.Warnf("stop downstream controller")
	downstream.Stop()
	log.LOGGER.Warnf("downstream controller stopped")
}

// Cleanup controller
func (dctl *DeviceController) Cleanup() {
	dctl.stopChan <- true
	config.Context.Cleanup(dctl.Name())
}
