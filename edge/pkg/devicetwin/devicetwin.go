package devicetwin

import (
	"context"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
)

//DeviceTwin the module
type DeviceTwin struct {
	Context           *beehiveContext.Context
	HeartBeatToModule map[string]chan interface{}
	DTContexts        *dtcontext.DTContext
	DTModules         map[string]dtmodule.DTModule
	cancel            context.CancelFunc
}

// Register register devicetwin
func Register() {
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
func (dt *DeviceTwin) Start(c *beehiveContext.Context) {
	var ctx context.Context
	dtContexts, _ := dtcontext.InitDTContext(c)
	dt.HeartBeatToModule = make(map[string]chan interface{})
	dt.DTModules = make(map[string]dtmodule.DTModule)
	dt.DTContexts = dtContexts
	dt.Context = c
	ctx, dt.cancel = context.WithCancel(context.Background())
	err := SyncSqlite(dt.DTContexts)
	if err != nil {
		klog.Errorf("Start DeviceTwin Failed, Sync Sqlite error:%v", err)
		return
	}
	dt.runDeviceTwin(ctx)
}

//Cleanup clean resource after quit
func (dt *DeviceTwin) Cleanup() {
	dt.cancel()
	dt.Context.Cleanup(dt.Name())
}
