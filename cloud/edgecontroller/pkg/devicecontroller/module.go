package devicecontroller

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/controller"
	"github.com/kubeedge/kubeedge/cloud/edgecontroller/pkg/devicecontroller/messagelayer"
)

// DeviceController use beehive context message layer
type DeviceController struct {
	messageLayer messagelayer.MessageLayer
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
	return constants.DeviceControllerModuleName
}

// Start controller
func (dctl *DeviceController) Start(c *bcontext.Context) {
	config.Context = c
	/*upstream, err := controller.NewUpstreamController()
	if err != nil {
		log.LOGGER.Warnf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()*/

	for {
		stopChannel := make(<-chan struct{})
		downstream, err := controller.NewDownstreamController()
		if err != nil {
			log.LOGGER.Warnf("New downstream controller failed with error: %s", err)
			continue
		}
		downstream.Start()

		<-stopChannel
		log.LOGGER.Warnf("stop downstream controller")
		downstream.Stop()
		log.LOGGER.Warnf("downstream controller stopped")
	}
}

// Cleanup controller
func (dctl *DeviceController) Cleanup() {
	// TODO
}
