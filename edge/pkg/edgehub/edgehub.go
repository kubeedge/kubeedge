package edgehub

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
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
	ctx           context.Context
	cancel        context.CancelFunc
	chClient      clients.Adapter
	config        *config.ControllerConfig
	reconnectChan chan struct{}
	syncKeeper    map[string]chan model.Message
	keeperLock    sync.RWMutex
}

func newEdgeHub() *EdgeHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &EdgeHub{
		ctx:           ctx,
		cancel:        cancel,
		config:        &config.GetConfig().CtrConfig,
		reconnectChan: make(chan struct{}),
		syncKeeper:    make(map[string]chan model.Message),
	}
}

// Register register edgehub
func Register() {
	core.Register(newEdgeHub())
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
func (eh *EdgeHub) Start() {
	config.InitEdgehubConfig()

	for {
		select {
		case <-eh.ctx.Done():
			klog.Warning("EdgeHub stop")
			return
		default:

		}
		err := eh.initial()
		if err != nil {
			klog.Fatalf("failed to init controller: %v", err)
			return
		}
		err = eh.chClient.Init()
		if err != nil {
			klog.Errorf("connection error, try again after 60s: %v", err)
			time.Sleep(waitConnectionPeriod)
			continue
		}
		// execute hook func after connect
		eh.pubConnectInfo(true)
		go eh.routeToEdge()
		go eh.routeToCloud()
		go eh.keepalive()

		// wait the stop singal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.Uninit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		time.Sleep(eh.config.HeartbeatPeriod * 2)

		// clean channel
	clean:
		for {
			select {
			case <-eh.reconnectChan:
			default:
				break clean
			}
		}
	}
}

//Cleanup sets up context cleanup through Edgehub name
func (eh *EdgeHub) Cancel() {
	eh.cancel()
}
