package edgehub

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	bhconfig "github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	http_utils "github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

const (
	waitConnectionPeriod = time.Minute
	defaultPlacement     = true
)

var (
	authEventType = "auth_info_event"
	groupMap      = map[string]string{"resource": modules.MetaGroup,
		"twin": modules.TwinGroup, "app": "sync",
		"func": modules.MetaGroup, "user": modules.BusGroup}

	// clear the number of data of the stop channel
	times = 2
)

//Controller is EdgeHub controller object
type Controller struct {
	context    *context.Context
	chClient   clients.Adapter
	config     *config.ControllerConfig
	stopChan   chan struct{}
	syncKeeper map[string]chan model.Message
	keeperLock sync.RWMutex
}

//NewEdgeHubController creates and returns a EdgeHubController object
func NewEdgeHubController() *Controller {
	return &Controller{
		config:     &config.GetConfig().CtrConfig,
		stopChan:   make(chan struct{}),
		syncKeeper: make(map[string]chan model.Message),
	}
}

func (ehc *Controller) initial(ctx *context.Context) error {
	getURL := func() string {
		for {
			url, err := ehc.getCloudHubURL()
			if err != nil {
				log.LOGGER.Warnf("failed to get cloud hub url, error:%+v", err)
				time.Sleep(time.Minute)
				continue
			}
			return url
		}
	}

	if ehc.config.ProjectID != "" && ehc.config.NodeID != "" {
		cloudHubURL := getURL()
		// TODO: set url gracefully
		config.GetConfig().WSConfig.URL = cloudHubURL
	} else {
		log.LOGGER.Warnf("use the config url for testing")
	}

	cloudHubClient, err := clients.GetClient(ehc.config.Protocol, config.GetConfig())
	if err != nil {
		return err
	}

	ehc.context = ctx
	ehc.chClient = cloudHubClient

	return nil
}

//Start will start EdgeHub
func (ehc *Controller) Start(ctx *context.Context) {
	for {
		err := ehc.initial(ctx)
		if err != nil {
			log.LOGGER.Fatalf("failed to init controller: %v", err)
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
		time.Sleep(ehc.config.HeartbeatPeriod * 2)

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

func (ehc *Controller) addKeepChannel(msgID string) chan model.Message {
	ehc.keeperLock.Lock()
	defer ehc.keeperLock.Unlock()

	tempChannel := make(chan model.Message)
	ehc.syncKeeper[msgID] = tempChannel

	return tempChannel
}

func (ehc *Controller) deleteKeepChannel(msgID string) {
	ehc.keeperLock.Lock()
	defer ehc.keeperLock.Unlock()

	delete(ehc.syncKeeper, msgID)
}

func (ehc *Controller) isSyncResponse(msgID string) bool {
	ehc.keeperLock.RLock()
	defer ehc.keeperLock.RUnlock()

	_, exist := ehc.syncKeeper[msgID]
	return exist
}

func (ehc *Controller) sendToKeepChannel(message model.Message) error {
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

func (ehc *Controller) dispatch(message model.Message) error {
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

func (ehc *Controller) routeToEdge() {
	for {
		message, err := ehc.chClient.Receive()
		if err != nil {
			log.LOGGER.Errorf("websocket read error: %v", err)
			ehc.stopChan <- struct{}{}
			return
		}

		log.LOGGER.Infof("received msg from cloud-hub:%+v", message)
		err = ehc.dispatch(message)
		if err != nil {
			log.LOGGER.Errorf("failed to dispatch message, discard: %v", err)
		}
	}
}

func (ehc *Controller) sendToCloud(message model.Message) error {
	err := ehc.chClient.Send(message)
	if err != nil {
		log.LOGGER.Errorf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message, error: %v", err)
	}

	syncKeep := func(message model.Message) {
		tempChannel := ehc.addKeepChannel(message.GetID())
		sendTimer := time.NewTimer(ehc.config.HeartbeatPeriod)
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

func (ehc *Controller) routeToCloud() {
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

func (ehc *Controller) keepalive() {
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
		time.Sleep(ehc.config.HeartbeatPeriod)
	}
}

func (ehc *Controller) pubConnectInfo(isConnected bool) {
	// var info model.Message
	content := connect.CloudConnected
	if !isConnected {
		content = connect.CloudDisconnected
	}

	for _, group := range groupMap {
		message := model.NewMessage("").BuildRouter(message.SourceNodeConnection, group,
			message.ResourceTypeNodeConnection, message.OperationNodeConnection).FillBody(content)
		ehc.context.Send2Group(group, *message)
	}
}

func (ehc *Controller) postURLRequst(client *http.Client) (string, error) {
	req, err := http_utils.BuildRequest(http.MethodGet, ehc.config.PlacementURL, nil, "")
	if err != nil {
		log.LOGGER.Errorf("failed to build request: %v", err)
		return "", err
	}

	for {
		resp, err := http_utils.SendRequest(req, client)
		if err != nil {
			log.LOGGER.Errorf("%v", err)
			time.Sleep(time.Minute)
			continue
		}
		switch resp.StatusCode {
		case http.StatusOK:
			defer resp.Body.Close()
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			url := fmt.Sprintf("%s/%s/%s/events", string(bodyBytes), ehc.config.ProjectID, ehc.config.NodeID)
			log.LOGGER.Infof("successfully to get cloudaccess url: %s", url)
			return url, nil
		case http.StatusBadRequest:
			log.LOGGER.Errorf("no retry on error code: %d, failed to get cloudaccess url", resp.StatusCode)
			return "", fmt.Errorf("bad request")
		default:
			log.LOGGER.Errorf("get cloudaccess with Error code: %d", resp.StatusCode)
		}
		time.Sleep(time.Minute)
	}
}

func (ehc *Controller) getCloudHubURL() (string, error) {
	placement, err := bhconfig.CONFIG.GetValue("edgehub.controller.placement").ToBool()
	if err != nil {
		placement = defaultPlacement
	}
	if placement {
		// TODO: get the file path gracefully
		certFile := config.GetConfig().WSConfig.CertFilePath
		keyFile := config.GetConfig().WSConfig.KeyFilePath
		placementClient, err := http_utils.NewHTTPSclient(certFile, keyFile)
		if err != nil {
			log.LOGGER.Warnf("failed to new https client for placement, error: %+v", err)
			return "", fmt.Errorf("failed to new https client for placement, error: %+v", err)
		}

		cloudHubURL, err := ehc.postURLRequst(placementClient)
		if err != nil {
			log.LOGGER.Warnf("failed to get cloud hub url, error: %+v", err)
			return "", fmt.Errorf("failed to new https client for placement, error: %+v", err)
		}

		return cloudHubURL, nil
	}
	return bhconfig.CONFIG.GetValue("edgehub.websocket.url").ToString()
}
