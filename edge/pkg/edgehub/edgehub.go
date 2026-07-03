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

// reconnectBackoff returns the backoff used between reconnect attempts.
// It starts small and grows exponentially with jitter, capped at 30s, so
// that a quickly-recovered network surfaces a NotReady node back to Ready
// within seconds instead of waiting heartbeat*2 (up to 4 minutes when
// heartbeat is configured to 120s).
func reconnectBackoff() wait.Backoff {
	return wait.Backoff{
		Duration: 2 * time.Second,
		Factor:   2.0,
		Jitter:   0.2,
		Cap:      30 * time.Second,
		// Steps only bounds how many times Duration may grow — it never
		// stops Step() from returning values: once Cap is reached, Step()
		// keeps returning a jittered value in [Cap, Cap*(1+Jitter))
		// indefinitely. MaxInt32 just ensures growth is never cut short.
		Steps: math.MaxInt32,
	}
}

// EdgeHub defines edgehub object structure
type EdgeHub struct {
	certManager   certificate.CertManager
	chClient      clients.Adapter
	reconnectChan chan struct{}
	rotateChan    chan struct{}
	rateLimiter   flowcontrol.RateLimiter
	keeperLock    sync.RWMutex
	enable        bool
}

var _ core.Module = (*EdgeHub)(nil)

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
		enable: enable,
		// Buffered(1) so that any sender can deliver a reconnect signal
		// without blocking when one is already pending. Multiple senders
		// (routeToEdge / routeToCloud / keepalive) all converge on this
		// channel; coalescing them is intentional.
		reconnectChan: make(chan struct{}, 1),
		// rotateChan carries certificate-rotation signals separately from
		// reconnectChan. A rotation must always be followed by a (re)connect
		// that loads the new certificate from disk: while connected, a
		// rotation signal is never discarded (the post-connect drain only
		// touches reconnectChan) and is consumed by the reconnect wait in
		// Start; a stale signal is dropped only while disconnected, right
		// before an Init() that reads the newest certificate anyway.
		rotateChan: make(chan struct{}, 1),
		rateLimiter: flowcontrol.NewTokenBucketRateLimiter(
			float32(config.Config.EdgeHub.MessageQPS),
			int(config.Config.EdgeHub.MessageBurst)),
	}
}

// triggerReconnect requests a reconnect without blocking. If a reconnect
// signal is already pending, the call is a no-op since one signal is
// sufficient to drive the reconnect loop.
func (eh *EdgeHub) triggerReconnect() {
	select {
	case eh.reconnectChan <- struct{}{}:
	default:
	}
}

// drainReconnect discards a pending transport reconnect signal, if any.
// It deliberately does not touch rotateChan: dropping a rotation signal
// could leave a connection running on a stale certificate until the next
// natural disconnect.
func (eh *EdgeHub) drainReconnect() {
	select {
	case <-eh.reconnectChan:
	default:
	}
}

// shouldResetBackoff reports whether the connection stayed up long enough to
// consider the previous outage over. Resetting on every successful handshake
// would let a connect-then-die loop (e.g. an overloaded CloudHub or a broken
// LB) retry at the initial ~2s interval forever — a connection storm across
// a large fleet. Surviving longer than the backoff cap is taken as healthy.
func shouldResetBackoff(connectedAt time.Time, b wait.Backoff) bool {
	return time.Since(connectedAt) > b.Cap
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

	backoff := reconnectBackoff()
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

		// A rotation that completed while we were disconnected is already
		// satisfied by the upcoming Init(), which reads the newest
		// certificate from disk (certManager writes the files before
		// signaling). Discard such a stale signal now — while still
		// disconnected — so it does not trigger a redundant reconnect right
		// after the connection is established. A rotation completing after
		// this point sends a fresh signal that survives to the wait below.
		select {
		case <-eh.rotateChan:
		default:
		}
		err = eh.chClient.Init()
		if err != nil {
			sleep := backoff.Step()
			klog.Errorf("connection failed: %v, will reconnect after %s", err, sleep.String())
			time.Sleep(sleep)
			continue
		}
		// Drain any stale transport-error signal queued by the previous
		// generation of goroutines while Init was in flight. Without this,
		// the stale signal would be received immediately by the wait below,
		// triggering an unnecessary reconnect cycle right after a successful
		// connect. Certificate-rotation signals are deliberately not
		// drained (see rotateChan).
		eh.drainReconnect()
		connectedAt := time.Now()
		// execute hook func after connect
		eh.pubConnectInfo(true)
		go eh.routeToEdge()
		go eh.routeToCloud()
		go eh.keepalive()

		// wait the stop signal
		// stop authinfo manager/websocket connection
		rotated := false
		select {
		case <-eh.reconnectChan:
		case <-eh.rotateChan:
			// The certificate was rotated (possibly while the connect above
			// was in flight). Re-establish the connection so chClient.Init()
			// reloads the new certificate from disk.
			rotated = true
		}
		eh.chClient.UnInit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// Reset the backoff only after the connection proved healthy for a
		// while, so a connect-then-die loop keeps backing off instead of
		// hammering the cloud at the initial interval.
		if shouldResetBackoff(connectedAt, backoff) {
			backoff = reconnectBackoff()
		}
		sleep := backoff.Step()
		if rotated {
			// Intentional disconnect: not a transport failure, so do not
			// alarm operators with a broken-connection warning.
			klog.Infof("certificate rotated, will reconnect after %s to reload it", sleep.String())
		} else {
			klog.Warningf("connection is broken, will reconnect after %s", sleep.String())
		}
		time.Sleep(sleep)

		// reconnectChan is buffered(1) and triggerReconnect is non-blocking,
		// so at most one queued signal can remain. A single non-blocking
		// receive is sufficient to start the next cycle from a clean state.
		eh.drainReconnect()
	}
}
