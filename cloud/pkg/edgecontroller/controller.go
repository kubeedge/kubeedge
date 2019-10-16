package edgecontroller

import (
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	controllerconfig "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/controller"
	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

// Controller use beehive context message layer
type Controller struct {
	stopChan chan bool
}

func Register(ecc *cloudcoreconfig.EdgeControllerConfig,
	k *cloudcoreconfig.KubeConfig,
	cc *cloudcoreconfig.ControllerContext,
	ec *edgecoreconfig.EdgedConfig,
	m *edgecoreconfig.Metamanager) {
	controllerconfig.InitEdgeControllerConfig(ecc, k, cc, ec, m)
	core.Register(&Controller{})
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
	controllerconfig.Context = c

	ctl.stopChan = make(chan bool)

	upstream, err := controller.NewUpstreamController()
	if err != nil {
		klog.Errorf("new upstream controller failed with error: %s", err)
		os.Exit(1)
	}
	upstream.Start()

	downstream, err := controller.NewDownstreamController()
	if err != nil {
		klog.Warningf("new downstream controller failed with error: %s", err)
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
	controllerconfig.Context.Cleanup(ctl.Name())
}
