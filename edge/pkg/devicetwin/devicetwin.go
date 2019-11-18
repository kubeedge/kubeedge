package devicetwin

import (
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
)

//DeviceTwin the module
type DeviceTwin struct {
	HeartBeatToModule map[string]chan interface{}
	DTContexts        *dtcontext.DTContext
	DTModules         map[string]dtmodule.DTModule
}

func newDeviceTwin() *DeviceTwin {
	return &DeviceTwin{
		HeartBeatToModule: make(map[string]chan interface{}),
		DTModules:         make(map[string]dtmodule.DTModule),
	}
}

// Register register devicetwin
func Register() {
	dtclient.InitDBTable()
	core.Register(newDeviceTwin())
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
func (dt *DeviceTwin) Start() {
	dtContexts, _ := dtcontext.InitDTContext()
	dt.DTContexts = dtContexts
	err := SyncSqlite(dt.DTContexts)
	if err != nil {
		klog.Errorf("Start DeviceTwin Failed, Sync Sqlite error:%v", err)
		return
	}
	dt.runDeviceTwin()
}
