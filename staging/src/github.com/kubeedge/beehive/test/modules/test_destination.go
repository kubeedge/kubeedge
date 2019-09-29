package modules

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
)

//Constants for module name and group
const (
	DestinationModule = "destinationmodule"
	DestinationGroup  = "destinationgroup"
)

type testModuleDest struct {
	context *context.Context
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

func (m *testModuleDest) Start(c *context.Context) {
	m.context = c
	message, err := c.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	message, err = c.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	resp := message.NewRespByMessage(&message, "fine")
	if message.IsSync() {
		c.SendResp(*resp)
	}

	message, err = c.Receive(DestinationModule)
	fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	if message.IsSync() {
		resp = message.NewRespByMessage(&message, "fine")
		c.SendResp(*resp)
	}

	//message, err = c.Receive(DestinationModule)
	//fmt.Printf("destination module receive message:%v error:%v\n", message, err)
	//if message.IsSync() {
	//	resp = message.NewRespByMessage(&message, "20 years old")
	//	c.SendResp(*resp)
	//}
}

func (m *testModuleDest) Cleanup() {
	m.context.Cleanup(m.Name())
}
