package edgehub

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	msghandler "github.com/kubeedge/kubeedge/edge/pkg/edgehub/messagehandler"
)

var (
	// longThrottleLatency defines threshold for logging requests. All requests being
	// throttled (via the provided rateLimiter) for more than longThrottleLatency will
	// be logged.
	longThrottleLatency = 1 * time.Second
)

func (eh *EdgeHub) initial() (err error) {
	cloudHubClient, err := clients.GetClient()
	if err != nil {
		klog.ErrorS(err, "Failed to get cloud hub client in initial")
		return err
	}

	eh.chClient = cloudHubClient

	klog.InfoS("Cloud hub client initialized successfully in initial")
	return nil
}

func (eh *EdgeHub) dispatch(message model.Message) error {
	return msghandler.ProcessHandler(message, eh.chClient)
}

func (eh *EdgeHub) routeToEdge() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warningf("EdgeHub RouteToEdge stop")
			return
		default:
		}
		message, err := eh.chClient.Receive()
		if err != nil {
			klog.ErrorS(err, "Websocket read error in routeToEdge")
			eh.reconnectChan <- struct{}{}
			return
		}
		klog.V(4).InfoS("Received message from cloud in routeToEdge", "messageID", message.GetID())
		if err = eh.dispatch(message); err != nil {
			klog.ErrorS(err, "Failed to dispatch message in routeToEdge", "messageID", message.GetID())
		}
	}
}

func (eh *EdgeHub) sendToCloud(message model.Message) error {
	eh.keeperLock.Lock()
	klog.V(4).InfoS("Sending message to cloud", "messageID", message.GetID())
	err := eh.chClient.Send(message)
	eh.keeperLock.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	klog.V(4).InfoS("Message sent to cloud successfully", "messageID", message.GetID())
	return nil
}

func (eh *EdgeHub) routeToCloud() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warningf("EdgeHub RouteToCloud stop")
			return
		default:
		}
		message, err := beehiveContext.Receive(modules.EdgeHubModuleName)
		if err != nil {
			klog.ErrorS(err, "Failed to receive message from edge in routeToCloud")
			time.Sleep(time.Second)
			continue
		}

		err = eh.tryThrottle(message.GetID())
		if err != nil {
			klog.ErrorS(err, "Client rate limiter returned an error in routeToCloud", "messageID", message.GetID())
			continue
		}

		// post message to cloud hub
		err = eh.sendToCloud(message)
		if err != nil {
			klog.ErrorS(err, "Failed to send message to cloud in routeToCloud", "messageID", message.GetID())
			eh.reconnectChan <- struct{}{}
			return
		}
	}
}

func (eh *EdgeHub) keepalive() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub KeepAlive stop")
			return
		default:
		}
		msg := model.NewMessage("").
			BuildRouter(modules.EdgeHubModuleName, "resource", "node", messagepkg.OperationKeepalive).
			FillBody("ping")

		// post message to cloud hub
		err := eh.sendToCloud(*msg)
		if err != nil {
			klog.Errorf("websocket write error: %v", err)
			eh.reconnectChan <- struct{}{}
			return
		}

		time.Sleep(time.Duration(config.Config.Heartbeat) * time.Second)
	}
}

var pubGroups = []string{modules.TwinGroup, modules.MetaGroup, modules.BusGroup, modules.TaskManagerGroup}

func (eh *EdgeHub) pubConnectInfo(isConnected bool) {
	// update connected info
	connect.SetConnected(isConnected)

	// var info model.Message
	content := connect.CloudConnected
	if !isConnected {
		content = connect.CloudDisconnected
	}

	for _, group := range pubGroups {
		message := model.NewMessage("").BuildRouter(messagepkg.SourceNodeConnection, group,
			messagepkg.ResourceTypeNodeConnection, messagepkg.OperationNodeConnection).FillBody(content)
		beehiveContext.SendToGroup(group, *message)
	}
}

func (eh *EdgeHub) ifRotationDone() {
	if eh.certManager.RotateCertificates {
		for {
			<-eh.certManager.Done
			eh.reconnectChan <- struct{}{}
		}
	}
}

func (eh *EdgeHub) tryThrottle(msgID string) error {
	now := time.Now()

	err := eh.rateLimiter.Wait(context.TODO())
	if err != nil {
		return err
	}

	latency := time.Since(now)

	message := fmt.Sprintf("Waited for %v due to client-side throttling, msgID: %s", latency, msgID)
	if latency > longThrottleLatency {
		klog.V(2).Info(message)
	}

	return nil
}
