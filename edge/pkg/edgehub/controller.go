package edgehub

import (
	"fmt"
	"time"

	"k8s.io/klog"

	bhconfig "github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

const (
	waitConnectionPeriod = time.Minute
	authEventType        = "auth_info_event"
)

var groupMap = map[string]string{
	"resource": modules.MetaGroup,
	"twin":     modules.TwinGroup,
	"func":     modules.MetaGroup,
	"user":     modules.BusGroup,
}

func (eh *EdgeHub) initial() (err error) {
	config.GetConfig().WSConfig.URL, err = bhconfig.CONFIG.GetValue("edgehub.websocket.url").ToString()
	if err != nil {
		klog.Warningf("failed to get cloud hub url, error:%+v", err)
		return err
	}

	cloudHubClient, err := clients.GetClient(eh.config.Protocol, config.GetConfig())
	if err != nil {
		return err
	}

	eh.chClient = cloudHubClient

	return nil
}

//Start will start EdgeHub
func (eh *EdgeHub) start() {
	config.InitEdgehubConfig()
	for {
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
		<-eh.stopChan
		eh.chClient.Uninit()

		// execute hook fun after disconnect
		eh.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		time.Sleep(eh.config.HeartbeatPeriod * 2)

		// clean channel
	clean:
		for {
			select {
			case <-eh.stopChan:
			default:
				break clean
			}
		}
	}
}

func (eh *EdgeHub) addKeepChannel(msgID string) chan model.Message {
	eh.keeperLock.Lock()
	defer eh.keeperLock.Unlock()

	tempChannel := make(chan model.Message)
	eh.syncKeeper[msgID] = tempChannel

	return tempChannel
}

func (eh *EdgeHub) deleteKeepChannel(msgID string) {
	eh.keeperLock.Lock()
	defer eh.keeperLock.Unlock()

	delete(eh.syncKeeper, msgID)
}

func (eh *EdgeHub) isSyncResponse(msgID string) bool {
	eh.keeperLock.RLock()
	defer eh.keeperLock.RUnlock()

	_, exist := eh.syncKeeper[msgID]
	return exist
}

func (eh *EdgeHub) sendToKeepChannel(message model.Message) error {
	eh.keeperLock.RLock()
	defer eh.keeperLock.RUnlock()
	channel, exist := eh.syncKeeper[message.GetParentID()]
	if !exist {
		klog.Errorf("failed to get sync keeper channel, messageID:%+v", message)
		return fmt.Errorf("failed to get sync keeper channel, messageID:%+v", message)
	}
	// send response into synckeep channel
	select {
	case channel <- message:
	default:
		klog.Errorf("failed to send message to sync keep channel")
		return fmt.Errorf("failed to send message to sync keep channel")
	}
	return nil
}

func (eh *EdgeHub) dispatch(message model.Message) error {
	// TODO: dispatch message by the message type
	md, ok := groupMap[message.GetGroup()]
	if !ok {
		klog.Warningf("msg_group not found")
		return fmt.Errorf("msg_group not found")
	}

	isResponse := eh.isSyncResponse(message.GetParentID())
	if !isResponse {
		eh.context.SendToGroup(md, message)
		return nil
	}
	return eh.sendToKeepChannel(message)
}

func (eh *EdgeHub) routeToEdge() {
	for {
		message, err := eh.chClient.Receive()
		if err != nil {
			klog.Errorf("websocket read error: %v", err)
			eh.stopChan <- struct{}{}
			return
		}

		klog.Infof("received msg from cloud-hub:%+v", message)
		err = eh.dispatch(message)
		if err != nil {
			klog.Errorf("failed to dispatch message, discard: %v", err)
		}
	}
}

func (eh *EdgeHub) sendToCloud(message model.Message) error {
	eh.keeperLock.Lock()
	err := eh.chClient.Send(message)
	eh.keeperLock.Unlock()
	if err != nil {
		klog.Errorf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	syncKeep := func(message model.Message) {
		tempChannel := eh.addKeepChannel(message.GetID())
		sendTimer := time.NewTimer(eh.config.HeartbeatPeriod)
		select {
		case response := <-tempChannel:
			sendTimer.Stop()
			eh.context.SendResp(response)
			eh.deleteKeepChannel(response.GetParentID())
		case <-sendTimer.C:
			klog.Warningf("timeout to receive response for message: %+v", message)
			eh.deleteKeepChannel(message.GetID())
		}
	}

	if message.IsSync() {
		go syncKeep(message)
	}

	return nil
}

func (eh *EdgeHub) routeToCloud() {
	for {
		message, err := eh.context.Receive(ModuleNameEdgeHub)
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// post message to cloud hub
		err = eh.sendToCloud(message)
		if err != nil {
			klog.Errorf("failed to send message to cloud: %v", err)
			eh.stopChan <- struct{}{}
			return
		}
	}
}

func (eh *EdgeHub) keepalive() {
	for {
		msg := model.NewMessage("").
			BuildRouter(ModuleNameEdgeHub, "resource", "node", "keepalive").
			FillBody("ping")

		// post message to cloud hub
		err := eh.sendToCloud(*msg)
		if err != nil {
			klog.Errorf("websocket write error: %v", err)
			eh.stopChan <- struct{}{}
			return
		}

		time.Sleep(eh.config.HeartbeatPeriod)
	}
}

func (eh *EdgeHub) pubConnectInfo(isConnected bool) {
	// var info model.Message
	content := connect.CloudConnected
	if !isConnected {
		content = connect.CloudDisconnected
	}

	for _, group := range groupMap {
		message := model.NewMessage("").BuildRouter(message.SourceNodeConnection, group,
			message.ResourceTypeNodeConnection, message.OperationNodeConnection).FillBody(content)
		eh.context.SendToGroup(group, *message)
	}
}
