package edgehub

import (
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

//define edgehub module name
const (
	ModuleNameEdgeHub = "websocket"
)

var HasTLSTunnelCerts = make(chan bool, 1)

//EdgeHub defines edgehub object structure
type EdgeHub struct {
	certManager   certificate.CertManager
	chClient      clients.Adapter
	reconnectChan chan struct{}
	syncKeeper    map[string]chan model.Message
	keeperLock    sync.RWMutex
	enable        bool
}

func newEdgeHub(enable bool) *EdgeHub {
	return &EdgeHub{
		reconnectChan: make(chan struct{}),
		syncKeeper:    make(map[string]chan model.Message),
		enable:        enable,
	}
}

// Register register edgehub
func Register(eh *v1alpha1.EdgeHub, nodeName string) {
	config.InitConfigure(eh, nodeName)
	core.Register(newEdgeHub(eh.Enable))
}

//Name returns the name of EdgeHub module
func (eh *EdgeHub) Name() string {
	return ModuleNameEdgeHub
}

//Group returns EdgeHub group
func (eh *EdgeHub) Group() string {
	return modules.HubGroup
}

//Enable indicates whether this module is enabled
func (eh *EdgeHub) Enable() bool {
	return eh.enable
}

//Start sets context and starts the controller
func (eh *EdgeHub) Start() {
	eh.certManager = certificate.NewCertManager(config.Config.EdgeHub, config.Config.NodeName)
	eh.certManager.Start()

	HasTLSTunnelCerts <- true
	close(HasTLSTunnelCerts)

	go eh.ifRotationDone()

	for {
		select {
		case <-beehiveContext.Done():
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

		// wait the stop signal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.UnInit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		time.Sleep(time.Duration(config.Config.Heartbeat) * time.Second * 2)

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
