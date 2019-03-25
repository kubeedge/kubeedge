package dtmodule

import (
	"strings"

	"github.com/kubeedge/beehive/pkg/common/log"
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

	if strings.Compare(dm.Name, "MemModule") == 0 {
		dm.Worker = dtmanager.MemWorker{
			Group: "MemModule",
			Worker: dtmanager.Worker{ReceiverChan: recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext}}
	} else if strings.Compare(dm.Name, "TwinModule") == 0 {
		dm.Worker = dtmanager.TwinWorker{
			Group: "TwinModule",
			Worker: dtmanager.Worker{ReceiverChan: recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext}}
	} else if strings.Compare(dm.Name, "DeviceModule") == 0 {
		dm.Worker = dtmanager.DeviceWorker{
			Group: "DeviceModule",
			Worker: dtmanager.Worker{ReceiverChan: recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext}}
	} else if strings.Compare(dm.Name, "CommModule") == 0 {
		dm.Worker = dtmanager.CommWorker{
			Group: "CommModule",
			Worker: dtmanager.Worker{ReceiverChan: recv,
				ConfirmChan:   confirm,
				HeartBeatChan: heartBeat,
				DTContexts:    dtContext}}
	}
}

//Start module, actual worker start
func (dm DTModule) Start() {
	defer func() {
		if err := recover(); err != nil {
			log.LOGGER.Infof("%s in twin panic", dm.Name)
		}
	}()
	dm.Worker.Start()
}
