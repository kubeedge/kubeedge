package edgehub

import (
	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
)

const (
	ModuleNameEdgeHub = "websocket"
)

type EdgeHub struct {
	context    *context.Context
	controller *EdgeHubController
}

func init() {
	core.Register(&EdgeHub{
		controller: NewEdgeHubController(),
	})
}

func (eh *EdgeHub) Name() string {
	return ModuleNameEdgeHub
}

func (eh *EdgeHub) Group() string {
	return core.HubGroup
}

func (eh *EdgeHub) Start(c *context.Context) {
	eh.context = c
	eh.controller.Start(c)
}

func (eh *EdgeHub) Cleanup() {
	eh.context.Cleanup(eh.Name())
}
