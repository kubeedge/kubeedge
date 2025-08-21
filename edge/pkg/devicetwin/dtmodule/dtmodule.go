package dtmodule

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmanager"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

// DTModule module for devicetwin
type DTModule struct {
	Name   string
	Worker dtmanager.DTWorker
}

// InitWorker init worker for module
func (dm *DTModule) InitWorker(recv chan interface{}, confirm chan interface{}, heartBeat chan interface{}, dtContext *dtcontext.DTContext) {
	worker := dtmanager.Worker{
		ReceiverChan:  recv,
		ConfirmChan:   confirm,
		HeartBeatChan: heartBeat,
		DTContexts:    dtContext,
	}
	switch dm.Name {
	case dtcommon.MemModule:
		dm.Worker = dtmanager.MemWorker{
			Group:  dtcommon.MemModule,
			Worker: worker,
		}
	case dtcommon.TwinModule:
		dm.Worker = dtmanager.TwinWorker{
			Group:  dtcommon.TwinModule,
			Worker: worker,
		}
	case dtcommon.DeviceModule:
		dm.Worker = dtmanager.DeviceWorker{
			Group:  dtcommon.DeviceModule,
			Worker: worker,
		}
	case dtcommon.CommModule:
		dm.Worker = dtmanager.CommWorker{
			Group:  dtcommon.CommModule,
			Worker: worker,
		}
	case dtcommon.DMIModule:
		dm.Worker = dtmanager.DMIWorker{
			Group: dtcommon.DMIModule,
			Worker: dtmanager.Worker{
				ReceiverChan:  recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext,
			},
			DeviceService: dbclient.NewDeviceService(),
			MetaService:   dbclient.NewMetaService(),
		}
	}
}

// Start module, actual worker start
func (dm DTModule) Start() {
	defer func() {
		if err := recover(); err != nil {
			klog.Errorf("%s in twin panic with err: %v", dm.Name, err)
		}
	}()
	dm.Worker.Start()
}
