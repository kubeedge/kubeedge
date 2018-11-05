package dtmanager

import "kubeedge/pkg/devicetwin/dtcontext"

//DTWorker worker for devicetwin
type DTWorker interface {
	PreDeal(interface{}) (interface{}, error)
	Deal(interface{}) (interface{}, error)
	PostDeal(interface{}) (interface{}, error)
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
