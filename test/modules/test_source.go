package modules

import (
	"fmt"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

//Constants for module source and group
const (
	SourceModule = "sourcemodule"
	SourceGroup  = "sourcegroup"
)

type testModuleSource struct {
	context *context.Context
}

func init() {
	core.Register(&testModuleSource{})
}

func (*testModuleSource) Name() string {
	return SourceModule
}

func (*testModuleSource) Group() string {
	return SourceGroup
}

func (m *testModuleSource) Start(c *context.Context) {
	m.context = c
	message := model.NewMessage("").SetRoute(SourceModule, "").
		SetResourceOperation("test", model.InsertOperation).FillBody("hello")
	c.Send(DestinationModule, *message)

	message = model.NewMessage("").SetRoute(SourceModule, "").
		SetResourceOperation("test", model.UpdateOperation).FillBody("how are you")
	resp, err := c.SendSync(DestinationModule, *message, 5*time.Second)
	if err != nil {
		fmt.Printf("failed to send sync message, error:%v\n", err)
	} else {
		fmt.Printf("get resp: %v\n", resp)
	}

	message = model.NewMessage("").SetRoute(SourceModule, DestinationGroup).
		SetResourceOperation("test", model.DeleteOperation).FillBody("fine")
	c.Send2Group(DestinationGroup, *message)
}

func (m *testModuleSource) Cleanup() {
	m.context.Cleanup(m.Name())
}
