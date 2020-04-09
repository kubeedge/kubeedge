package modules

import (
	"fmt"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

//Constants for module source and group
const (
	SourceModule = "sourcemodule"
	SourceGroup  = "sourcegroup"
)

type testModuleSource struct {
}

func init() {
	core.Register(&testModuleSource{})
}

func (m *testModuleSource) Enable() bool {
	return true
}

func (*testModuleSource) Name() string {
	return SourceModule
}

func (*testModuleSource) Group() string {
	return SourceGroup
}

func (m *testModuleSource) Start() {
	message := model.NewMessage("").SetRoute(SourceModule, "").
		SetResourceOperation("test", model.InsertOperation).FillBody("hello")
	beehiveContext.Send(DestinationModule, *message)

	message = model.NewMessage("").SetRoute(SourceModule, "").
		SetResourceOperation("test", model.UpdateOperation).FillBody("how are you")
	resp, err := beehiveContext.SendSync(DestinationModule, *message, 5*time.Second)
	if err != nil {
		fmt.Printf("failed to send sync message, error:%v\n", err)
	} else {
		fmt.Printf("get resp: %v\n", resp)
	}

	message = model.NewMessage("").SetRoute(SourceModule, DestinationGroup).
		SetResourceOperation("test", model.DeleteOperation).FillBody("fine")
	beehiveContext.SendToGroup(DestinationGroup, *message)
}
