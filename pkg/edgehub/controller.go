package edgehub

import (
	"fmt"
	"sync"
	"time"

	"kubeedge/beehive/pkg/common/log"
	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
	"kubeedge/beehive/pkg/core/model"

	"kubeedge/pkg/common/message"
	"kubeedge/pkg/edgehub/clients"
	"kubeedge/pkg/edgehub/config"
)

const (
	waitConnectionPeriod = time.Minute
)

var (
	authEventType = "auth_info_event"
	groupMap      = map[string]string{"resource": core.MetaGroup,
		"twin": core.TwinGroup, "app": "sync",
		"func": core.MetaGroup, "user": core.BusGroup}

	// clear the number of data of the stop channel
	times = 2
)

type EdgeHubController struct {
	context    *context.Context
	chClient   clients.Adapter
	config     *config.ControllerConfig
	stopChan   chan struct{}
	syncKeeper map[string]chan model.Message
	keeperLock sync.RWMutex
}

func NewEdgeHubController() *EdgeHubController {
	return &EdgeHubController{
		config:     &config.GetConfig().CtrConfig,
		stopChan:   make(chan struct{}),
		syncKeeper: make(map[string]chan model.Message),
	}
}

func (ehc *EdgeHubController) initial(ctx *context.Context) error {

	if ehc.config.ProjectID != "" && ehc.config.NodeId != "" {
		// TODO: set url gracefully
		config.GetConfig().WSConfig.Url = ehc.config.CloudhubURL
	} else {
		log.LOGGER.Warnf("use the config url for testing")
	}

	cloudHubClient := clients.GetClient(clients.ClientTypeWebSocket, config.GetConfig())
	if cloudHubClient == nil {
		log.LOGGER.Errorf("failed to get web socket client")
		return fmt.Errorf("failed to get web socket client")
	}

	ehc.context = ctx
	ehc.chClient = cloudHubClient

	return nil
}

func (ehc *EdgeHubController) Start(ctx *context.Context) {
	for {
		err := ehc.initial(ctx)
		if err != nil {
			log.LOGGER.Fatalf("failed to init contorller: %v", err)
			return
		}

		err = ehc.chClient.Init()
		if err != nil {
			log.LOGGER.Errorf("connection error, try again after 60s: %v", err)
			time.Sleep(waitConnectionPeriod)
			continue
		}

		// execute hook func after connect
		ehc.pubConnectInfo(true)

		go ehc.routeToEdge()
		go ehc.routeToCloud()
		go ehc.keepalive()

		// wait the stop singal
		// stop authinfo manager/websocket connection
		<-ehc.stopChan
		ehc.chClient.Uninit()

		// execute hook fun after disconnect
		ehc.pubConnectInfo(false)

		// sleep one period of heartbeat, then try to connect cloud hub again
		time.Sleep(ehc.config.HeartbeatPeroid * 2)

		// clean channel
		for i := 0; i < times; i++ {
			select {
			case <-ehc.stopChan:
				continue
			default:
			}
		}
	}
}

func (ehc *EdgeHubController) addKeepChannel(msgID string) chan model.Message {
	ehc.keeperLock.Lock()
	defer ehc.keeperLock.Unlock()

	tempChannel := make(chan model.Message)
	ehc.syncKeeper[msgID] = tempChannel

	return tempChannel
}

func (ehc *EdgeHubController) deleteKeepChannel(msgID string) {
	ehc.keeperLock.Lock()
	defer ehc.keeperLock.Unlock()

	delete(ehc.syncKeeper, msgID)
}

func (ehc *EdgeHubController) isSyncResponse(msgID string) bool {
	ehc.keeperLock.RLock()
	defer ehc.keeperLock.RUnlock()

	_, exist := ehc.syncKeeper[msgID]
	return exist
}

func (ehc *EdgeHubController) sendToKeepChannel(message model.Message) error {
	ehc.keeperLock.RLock()
	defer ehc.keeperLock.RUnlock()

	channel, exist := ehc.syncKeeper[message.GetParentID()]
	if !exist {
		log.LOGGER.Errorf("failed to get sync keeper channel, messageID:%+v", message)
		return fmt.Errorf("failed to get sync keeper channel, messageID:%+v", message)
	}

	// send response into synckeep channel
	select {
	case channel <- message:
	default:
		log.LOGGER.Errorf("failed to send message to sync keep channel")
		return fmt.Errorf("failed to send message to sync keep channel")
	}

	return nil
}

func (ehc *EdgeHubController) dispatch(message model.Message) error {

	// TODO: dispatch message by the message type
	md, ok := groupMap[message.GetGroup()]
	if !ok {
		log.LOGGER.Warnf("msg_group not found")
		return fmt.Errorf("msg_group not found")
	}

	isResponse := ehc.isSyncResponse(message.GetParentID())
	if !isResponse {
		ehc.context.Send2Group(md, message)
		return nil
	}

	return ehc.sendToKeepChannel(message)
}

func (ehc *EdgeHubController) routeToEdge() {
	for {
		message, err := ehc.chClient.Receive()
		if err != nil {
			log.LOGGER.Errorf("websocket read error: %v", err)
			ehc.stopChan <- struct{}{}
			return
		}

		log.LOGGER.Infof("received msg from cloud-hub:%#v", message)
		err = ehc.dispatch(message)
		if err != nil {
			log.LOGGER.Errorf("failed to dispatch message, discard: %v", err)
		}
	}
}

func (ehc *EdgeHubController) sendToCloud(message model.Message) error {
	err := ehc.chClient.Send(message)
	if err != nil {
		log.LOGGER.Errorf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	syncKeep := func(message model.Message) {
		tempChannel := ehc.addKeepChannel(message.GetID())
		sendTimer := time.NewTimer(ehc.config.HeartbeatPeroid)
		select {
		case response := <-tempChannel:
			sendTimer.Stop()
			ehc.context.SendResp(response)
			ehc.deleteKeepChannel(response.GetParentID())
		case <-sendTimer.C:
			log.LOGGER.Warnf("timeout to receive response for message: %+v", message)
			ehc.deleteKeepChannel(message.GetID())
		}
	}

	if message.IsSync() {
		go syncKeep(message)
	}

	return nil
}

func (ehc *EdgeHubController) routeToCloud() {
	for {
		message, err := ehc.context.Receive(ModuleNameEdgeHub)
		if err != nil {
			log.LOGGER.Errorf("failed to receive message from edge: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// post message to cloud hub
		err = ehc.sendToCloud(message)
		if err != nil {
			log.LOGGER.Errorf("failed to send message to cloud: %v", err)
			ehc.stopChan <- struct{}{}
			return
		}
	}
}

func (ehc *EdgeHubController) keepalive() {
	for {
		msg := model.NewMessage("").
			BuildRouter(ModuleNameEdgeHub, "resource", "node", "keepalive").
			FillBody("ping")
		err := ehc.chClient.Send(*msg)
		if err != nil {
			log.LOGGER.Errorf("websocket write error: %v", err)
			ehc.stopChan <- struct{}{}
			return
		}
		time.Sleep(ehc.config.HeartbeatPeroid)
	}
}

func (ehc *EdgeHubController) pubConnectInfo(isConnected bool) {
	// var info model.Message
	content := model.CLOUD_CONNECTED
	if !isConnected {
		content = model.CLOUD_DISCONNECTED
	}

	for _, group := range groupMap {
		message := model.NewMessage("").BuildRouter(message.SourceNodeConnection, group,
			message.ResourceTypeNodeConnection, message.OperationNodeConnection).FillBody(content)
		ehc.context.Send2Group(group, *message)
	}
}
