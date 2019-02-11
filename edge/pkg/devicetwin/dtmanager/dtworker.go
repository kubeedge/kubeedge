package dtmanager

import "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"

//DTWorker worker for devicetwin
type DTWorker interface {
	Start()
}

//Worker actual
type Worker struct {
	ReceiverChan  chan interface{}
	ConfirmChan   chan interface{}
	HeartBeatChan chan interface{}
	DTContexts    *dtcontext.DTContext
}

//CallBack for deal
type CallBack func(*dtcontext.DTContext, string, interface{}) (interface{}, error)
