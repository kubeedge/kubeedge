package edgehub

import (
	"math"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	msghandler "github.com/kubeedge/kubeedge/edge/pkg/edgehub/messagehandler"
	"github.com/kubeedge/kubeedge/pkg/features"
)

// EdgeHub defines edgehub object structure
type EdgeHub struct {
	certManager   certificate.CertManager
	chClient      clients.Adapter
	reconnectChan chan struct{}
	rateLimiter   flowcontrol.RateLimiter
	keeperLock    sync.RWMutex
	enable        bool
}

var _ core.Module = (*EdgeHub)(nil)

const (
	// reconnectBaseDelay is the initial wait before the first reconnect attempt.
	reconnectBaseDelay = time.Second
	// reconnectFactor is the exponential growth factor between attempts.
	reconnectFactor = 2.0
	// reconnectJitter randomizes each wait so that many edge nodes losing the
	// connection to the same cloudcore at once do not reconnect in lockstep.
	reconnectJitter = 0.2
)

// newReconnectBackoff builds the exponential backoff used between reconnect
// attempts. It is capped at the previous fixed wait (Heartbeat*2), so the
// longest wait never regresses while early attempts happen much sooner.
func newReconnectBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: reconnectBaseDelay,
		Factor:   reconnectFactor,
		Jitter:   reconnectJitter,
		Cap:      time.Duration(config.Config.Heartbeat) * time.Second * 2,
		Steps:    math.MaxInt32,
	}
}

var certSync map[string]chan bool

func GetCertSyncChannel() map[string]chan bool {
	return certSync
}

func NewCertSyncChannel() map[string]chan bool {
	certSync = make(map[string]chan bool, 1)
	certSync[modules.EdgeStreamModuleName] = make(chan bool, 1)
	return certSync
}

func newEdgeHub(enable bool) *EdgeHub {
	NewCertSyncChannel()
	return &EdgeHub{
		enable:        enable,
		reconnectChan: make(chan struct{}),
		rateLimiter: flowcontrol.NewTokenBucketRateLimiter(
			float32(config.Config.EdgeHub.MessageQPS),
			int(config.Config.EdgeHub.MessageBurst)),
	}
}

// Register register edgehub
func Register(eh *v1alpha2.EdgeHub, nodeName string) {
	// Initialize the hub configuration
	config.InitConfigure(eh, nodeName)
	// Initialize the message handler
	msghandler.RegisterHandlers()
	// Register self to beehive modules
	core.Register(newEdgeHub(eh.Enable))
}

// Name returns the name of EdgeHub module
func (eh *EdgeHub) Name() string {
	return modules.EdgeHubModuleName
}

// Group returns EdgeHub group
func (eh *EdgeHub) Group() string {
	return modules.HubGroup
}

// Enable indicates whether this module is enabled
func (eh *EdgeHub) Enable() bool {
	return eh.enable
}

func (eh *EdgeHub) RestartPolicy() *core.ModuleRestartPolicy {
	if !features.DefaultFeatureGate.Enabled(features.ModuleRestart) {
		return nil
	}
	return &core.ModuleRestartPolicy{
		RestartType:            core.RestartTypeOnFailure,
		IntervalTimeGrowthRate: 2.0,
	}
}

// Start sets context and starts the controller
func (eh *EdgeHub) Start() {
	eh.certManager = certificate.NewCertManager(config.Config.EdgeHub, config.Config.NodeName)
	eh.certManager.Start()
	for _, v := range GetCertSyncChannel() {
		v <- true
		close(v)
	}

	go eh.ifRotationDone()

	backoff := newReconnectBackoff()
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub stop")
			return
		default:
		}
		err := eh.initial()
		if err != nil {
			klog.Exitf("failed to init controller: %v", err)
			return
		}

		err = eh.chClient.Init()
		if err != nil {
			waitTime := backoff.Step()
			klog.Errorf("connection failed: %v, will reconnect after %s", err, waitTime.String())
			time.Sleep(waitTime)
			continue
		}
		// execute hook func after connect
		eh.pubConnectInfo(true)
		connectedAt := time.Now()
		go eh.routeToEdge()
		go eh.routeToCloud()
		go eh.keepalive()

		// wait the stop signal
		// stop authinfo manager/websocket connection
		<-eh.reconnectChan
		eh.chClient.UnInit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// a connection that stayed up at least one full backoff cap is treated as
		// healthy, so the backoff resets and the next reconnect starts quickly; a
		// connection that keeps flapping keeps backing off to spare cloudcore.
		if time.Since(connectedAt) >= backoff.Cap {
			backoff = newReconnectBackoff()
		}
		waitTime := backoff.Step()
		klog.Warningf("connection is broken, will reconnect after %s", waitTime.String())
		time.Sleep(waitTime)

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
