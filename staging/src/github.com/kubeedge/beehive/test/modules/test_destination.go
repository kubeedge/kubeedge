package modules

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

//Constants for module name and group
const (
	DestinationModule = "destinationmodule"
	DestinationGroup  = "destinationgroup"
)

type testModuleDest struct {
}

func (m *testModuleDest) Enable() bool {
	return true
}

func init() {
	core.Register(&testModuleDest{})
}

func (*testModuleDest) Name() string {
	return DestinationModule
}

func (*testModuleDest) Group() string {
	return DestinationGroup
}

func (m *testModuleDest) Start() {
	message, err := beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	message, err = beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	resp := message.NewRespByMessage(&message, "fine")
	if message.IsSync() {
		beehiveContext.SendResp(*resp)
	}

	message, err = beehiveContext.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	if message.IsSync() {
		resp = message.NewRespByMessage(&message, "fine")
		beehiveContext.SendResp(*resp)
	}

	//message, err = c.Receive(DestinationModule)
	//fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	//if message.IsSync() {
	//	resp = message.NewRespByMessage(&message, "20 years old")
	//	c.SendResp(*resp)
	//}
}
