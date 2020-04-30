package edgehub

import (
	"crypto/tls"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
)

//define edgehub module name
const (
	ModuleNameEdgeHub = "websocket"
)

//EdgeHub defines edgehub object structure
type EdgeHub struct {
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
	// if there is no manual certificate setting or the setting has problems, then the edge applies for the certificate
	if validation.FileIsExist(config.Config.TLSCAFile) && validation.FileIsExist(config.Config.TLSCertFile) && validation.FileIsExist(config.Config.TLSPrivateKeyFile) {
		_, err := tls.LoadX509KeyPair(config.Config.TLSCertFile, config.Config.TLSPrivateKeyFile)
		if err != nil {
			if err := eh.applyCerts(); err != nil {
				klog.Fatalf("failed to apply for edge certificate, error: %v", err)
				return
			}
		}
	} else {
		if err := eh.applyCerts(); err != nil {
			klog.Fatalf("failed to apply for edge certificate, error: %v", err)
			return
		}
	}

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

		// wait the stop singal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.Uninit()

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
