package edgehub

import (
	"context"
	"sync"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

//define edgehub module name
const (
	ModuleNameEdgeHub = "websocket"
)

//EdgeHub defines edgehub object structure
type EdgeHub struct {
	context    *beehiveContext.Context
	chClient   clients.Adapter
	config     *config.ControllerConfig
	retryChan  chan struct{}
	cancel     context.CancelFunc
	syncKeeper map[string]chan model.Message
	keeperLock sync.RWMutex
}

// Register register edgehub
func Register() {
	core.Register(&EdgeHub{
		config:     &config.GetConfig().CtrConfig,
		retryChan:  make(chan struct{}),
		syncKeeper: make(map[string]chan model.Message),
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
func (eh *EdgeHub) Start(c *beehiveContext.Context) {
	var ctx context.Context
	eh.context = c
	ctx, eh.cancel = context.WithCancel(context.Background())
	eh.start(ctx)
}

//Cleanup sets up context cleanup through Edgehub name
func (eh *EdgeHub) Cleanup() {
	eh.cancel()
	eh.context.Cleanup(eh.Name())
}
