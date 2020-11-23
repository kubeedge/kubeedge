package dtmodule

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmanager"
)

//DTModule module for devicetwin
type DTModule struct {
	Name   string
	Worker dtmanager.DTWorker
}

// InitWorker init worker
func (dm *DTModule) InitWorker(recv chan interface{}, confirm chan interface{}, heartBeat chan interface{}, dtContext *dtcontext.DTContext) {
	switch dm.Name {
	case dtcommon.MemModule:
		dm.Worker = dtmanager.MemWorker{
			Group: dtcommon.MemModule,
			Worker: dtmanager.Worker{
				ReceiverChan:  recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext,
			},
		}
	case dtcommon.TwinModule:
		dm.Worker = dtmanager.TwinWorker{
			Group: dtcommon.TwinModule,
			Worker: dtmanager.Worker{
				ReceiverChan:  recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext,
			},
		}
	case dtcommon.DeviceModule:
		dm.Worker = dtmanager.DeviceWorker{
			Group: dtcommon.DeviceModule,
			Worker: dtmanager.Worker{
				ReceiverChan:  recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext,
			},
		}
	case dtcommon.CommModule:
		dm.Worker = dtmanager.CommWorker{
			Group: dtcommon.CommModule,
			Worker: dtmanager.Worker{
				ReceiverChan:  recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext,
			},
		}
	}
}

//Start module, actual worker start
func (dm DTModule) Start() {
	defer func() {
		if err := recover(); err != nil {
			klog.Infof("%s in twin panic", dm.Name)
		}
	}()
	dm.Worker.Start()
}
