package edgehub

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

//define edgehub module name
const (
	ModuleNameEdgeHub = "websocket"
)

//EdgeHub defines edgehub object structure
type EdgeHub struct {
	context    *context.Context
	controller *Controller
}

func init() {
	core.Register(&EdgeHub{
		controller: NewEdgeHubController(),
	})
}

//Name returns the name of EdgeHub module
func (eh *EdgeHub) Name() string {
	return ModuleNameEdgeHub
}

//Group returns EdgeHub group
func (eh *EdgeHub) Group() string {
	return modules.HubGroup
}

//Start sets context and starts the controller
func (eh *EdgeHub) Start(c *context.Context) {
	eh.context = c
	eh.controller.Start(c)
}

//Cleanup sets up context cleanup through Edgehub name
func (eh *EdgeHub) Cleanup() {
	eh.context.Cleanup(eh.Name())
}
