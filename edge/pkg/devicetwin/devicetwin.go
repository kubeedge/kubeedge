package devicetwin

import (
	"context"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
)

//DeviceTwin the module
type DeviceTwin struct {
	ctx               context.Context
	cancel            context.CancelFunc
	HeartBeatToModule map[string]chan interface{}
	DTContexts        *dtcontext.DTContext
	DTModules         map[string]dtmodule.DTModule
}

func newDeviceTwin() *DeviceTwin {
	ctx, cancel := context.WithCancel(context.Background())
	return &DeviceTwin{
		ctx:    ctx,
		cancel: cancel,
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
	dt.HeartBeatToModule = make(map[string]chan interface{})
	dt.DTModules = make(map[string]dtmodule.DTModule)
	dt.DTContexts = dtContexts
	err := SyncSqlite(dt.DTContexts)
	if err != nil {
		klog.Errorf("Start DeviceTwin Failed, Sync Sqlite error:%v", err)
		return
	}
	dt.runDeviceTwin()
}

//Cancel clean resource after quit
func (dt *DeviceTwin) Cancel() {
	dt.cancel()
}
