package controller

import (
	"os"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/controller"
)

// Controller use beehive context message layer
type Controller struct{}

func init() {
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
	upstream, err := controller.NewUpstreamController()
	if err != nil {
		log.LOGGER.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	for {
		stopChannel := make(<-chan struct{})
		downstream, err := controller.NewDownstreamController()
		if err != nil {
			log.LOGGER.Warnf("new downstream controller failed with error: %s", err)
			continue
		}
		downstream.Start()

		<-stopChannel
		log.LOGGER.Warnf("election as slave, start to stop downstream controller")
		downstream.Stop()
		log.LOGGER.Warnf("downstream controller stopped")
	}
}

// Cleanup controller
func (ctl *Controller) Cleanup() {
	// TODO
}
