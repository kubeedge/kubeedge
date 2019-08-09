package edgecontroller

import (
	"os"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
)

// Controller use beehive context message layer
type Controller struct {
	stopChan chan bool
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
func (ctl *Controller) Start(c *bcontext.Context) {
	config.Context = c
	ctl.stopChan = make(chan bool)

	initConfig()

	upstream, err := controller.NewUpstreamController()
	if err != nil {
		log.LOGGER.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		log.LOGGER.Warnf("new downstream controller failed with error: %s", err)
		os.Exit(1)
	}
	downstream.Start()

	<-ctl.stopChan
	upstream.Stop()
	downstream.Stop()
}

// Cleanup controller
func (ctl *Controller) Cleanup() {
	ctl.stopChan <- true
	config.Context.Cleanup(ctl.Name())
}

func initConfig() {
	config.InitBufferConfig()
	config.InitContextConfig()
	config.InitKubeConfig()
	config.InitLoadConfig()
	config.InitMessageLayerConfig()
}
