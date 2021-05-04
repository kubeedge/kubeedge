package modules

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

//Constant for test module destination group name
const (
	DestinationGroupModule = "destinationgroupmodule"
)

type testModuleDestGroup struct {
}

func (m *testModuleDestGroup) Enable() bool {
	return true
}

func init() {
	core.Register(&testModuleDestGroup{})
}

func (*testModuleDestGroup) Name() string {
	return DestinationGroupModule
}

func (*testModuleDestGroup) Group() string {
	return DestinationGroup
}

func (m *testModuleDestGroup) Start() {
	message, err := beehiveContext.Receive(DestinationGroupModule)
	fmt.Printf("destination group module receive message:%v error:%v\n", message, err)
	if message.IsSync() {
		resp := message.NewRespByMessage(&message, "10 years old")
		beehiveContext.SendResp(*resp)
	}
}
