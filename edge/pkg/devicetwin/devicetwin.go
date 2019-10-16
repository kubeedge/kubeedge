package devicetwin

import (
	"context"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	bcontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	devicetwinconfig "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/config"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

//DeviceTwin the module
type DeviceTwin struct {
	context      *bcontext.Context
	dtcontroller *DTController
	cancel       context.CancelFunc
}

// Register register devicetwin
func Register(e *edgecoreconfig.EdgedConfig) {
	devicetwinconfig.InitDeviceTwinConfig(e)
	dtclient.InitDBTable()
	dt := DeviceTwin{}
	core.Register(&dt)
}

//Name get name of the module
func (dt *DeviceTwin) Name() string {
	return "twin"
}

//Group get group of the module
func (dt *DeviceTwin) Group() string {
	return modules.TwinGroup
}

//Start run the module
func (dt *DeviceTwin) Start(c *bcontext.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	dt.cancel = cancel

	controller, err := InitDTController(c)
	if err != nil {
		klog.Errorf("Start device twin failed, due to %v", err)
	}
	dt.dtcontroller = controller
	dt.context = c
	err = controller.Start(ctx)
	if err != nil {
		klog.Errorf("Start device twin failed, due to %v", err)
	}
}

//Cleanup clean resource after quit
func (dt *DeviceTwin) Cleanup() {
	dt.cancel()
	dt.context.Cleanup(dt.Name())
}
