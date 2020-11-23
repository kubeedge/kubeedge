package devicetwin

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	deviceconfig "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/config"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

//DeviceTwin the module
type DeviceTwin struct {
	HeartBeatToModule map[string]chan interface{}
	DTContexts        *dtcontext.DTContext
	DTModules         map[string]dtmodule.DTModule
	enable            bool
}

func newDeviceTwin(enable bool) *DeviceTwin {
	return &DeviceTwin{
		HeartBeatToModule: make(map[string]chan interface{}),
		DTModules:         make(map[string]dtmodule.DTModule),
		enable:            enable,
	}
}

// Register register devicetwin
func Register(deviceTwin *v1alpha1.DeviceTwin, nodeName string) {
	deviceconfig.InitConfigure(deviceTwin, nodeName)
	dt := newDeviceTwin(deviceTwin.Enable)
	dtclient.InitDBTable(dt)
	core.Register(dt)
}

// Name get name of the module
func (dt *DeviceTwin) Name() string {
	return modules.DeviceTwinModuleName
}

// Group get group of the module
func (dt *DeviceTwin) Group() string {
	return modules.TwinGroup
}

// Enable indicates whether this module is enabled
func (dt *DeviceTwin) Enable() bool {
	return dt.enable
}

// Start run the module
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
