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
	"github.com/kubeedge/kubeedge/pkg/features"
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
		return err
	}

	eh.chClient = cloudHubClient

	return nil
}

func (eh *EdgeHub) dispatch(message model.Message) error {
	return msghandler.ProcessHandler(message, eh.chClient)
}

func (eh *EdgeHub) routeToEdge() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub RouteToEdge stop")
			return
		default:
		}
		message, err := eh.chClient.Receive()
		if err != nil {
			klog.Errorf("websocket read error: %v", err)
			eh.reconnectChan <- struct{}{}
			return
		}
		klog.V(4).Infof("[edgehub/routeToEdge] receive msg from cloud, msg: %+v", message)
		if err = eh.dispatch(message); err != nil {
			klog.Error(err)
		}
	}
}

func (eh *EdgeHub) sendToCloud(message model.Message) error {
	eh.keeperLock.Lock()
	klog.V(4).Infof("[edgehub/sendToCloud] send msg to cloud, msg: %+v", message)
	err := eh.chClient.Send(message)
	eh.keeperLock.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	return nil
}

func (eh *EdgeHub) routeToCloud() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub RouteToCloud stop")
			return
		default:
		}
		message, err := beehiveContext.Receive(modules.EdgeHubModuleName)
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if features.DefaultFeatureGate.Enabled(features.MessagePriorityQueues) {
			classifyPriorityEdge(&message)
			// enqueue for prioritized sending; throttling applied in sender
			eh.sendPQ.Add(message)
			continue
		}
		// fallback path: direct send
		_ = eh.tryThrottle(message.GetID())
		if err := eh.sendToCloud(message); err != nil {
			klog.Errorf("failed to send message to cloud: %v", err)
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

		if features.DefaultFeatureGate.Enabled(features.MessagePriorityQueues) {
			// enqueue keepalive to priority queue
			classifyPriorityEdge(msg)
			eh.sendPQ.Add(*msg)
		} else {
			_ = eh.tryThrottle(msg.GetID())
			if err := eh.sendToCloud(*msg); err != nil {
				klog.Errorf("failed to send keepalive to cloud: %v", err)
				return
			}
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

// classifyPriorityEdge: apply rule table to override default priority, except for responses.
func classifyPriorityEdge(msg *model.Message) {
	// response inherit: keep as-is
	if msg.GetOperation() == model.ResponseOperation {
		return
	}
	// simple rules for edge->cloud (customize as needed)
	if msg.GetOperation() == model.DeleteOperation {
		msg.SetPriority(model.PriorityImportant)
		return
	}
	// keepalive should be important to maintain session health
	if msg.GetOperation() == messagepkg.OperationKeepalive {
		msg.SetPriority(model.PriorityUrgent)
		return
	}
	// other messages keep default PriorityNormal (already set by NewMessage)
}
