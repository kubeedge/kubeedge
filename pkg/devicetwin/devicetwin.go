package devicetwin

import (
	"edge-core/beehive/pkg/common/log"
	core "edge-core/beehive/pkg/core"
	"edge-core/beehive/pkg/core/context"
	"edge-core/pkg/devicetwin/dtclient"
)

//DeviceTwin the module
type DeviceTwin struct {
	context      *context.Context
	dtcontroller *DTController
}

func init() {
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
	return "twin"
}

//Start run the module
func (dt *DeviceTwin) Start(c *context.Context) {
	controller, err := InitDTController(c)
	if err != nil {
		log.LOGGER.Errorf("Start device twin failed, due to %v", err)
	}
	dt.dtcontroller = controller
	dt.context = c
	err = controller.Start()
	if err != nil {
		log.LOGGER.Errorf("Start device twin failed, due to %v", err)
	}
}

//Cleanup clean resource after quit
func (dt *DeviceTwin) Cleanup() {
	dt.dtcontroller.Stop <- true
	dt.context.Cleanup(dt.Name())
}
