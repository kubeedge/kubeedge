package edgehub

import (
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
)

const (
	waitConnectionPeriod = time.Minute
	authEventType        = "auth_info_event"
	caURL                = "/ca.crt"
	certURL              = "/edge.crt"
)

var groupMap = map[string]string{
	"resource": modules.MetaGroup,
	"twin":     modules.TwinGroup,
	"func":     modules.MetaGroup,
	"user":     modules.BusGroup,
}

// applyCerts get edge certificate to communicate with cloudcore
func (eh *EdgeHub) applyCerts() error {
	// get ca.crt
	url := config.Config.HttpServer + caURL
	cacert, err := certutil.GetCACert(url)
	if err != nil {
		klog.Errorf("failed to get CA certificate, err: %v", err)
		return fmt.Errorf("failed to get CA certificate, err: %v", err)
	}

	// validate the CA certificate by hashcode
	tokenParts := strings.Split(config.Config.Token, " ")
	if len(tokenParts) != 2 {
		return fmt.Errorf("token credentials are in the wrong format")
	}
	ok, hash, newHash := certutil.ValidateCACerts(cacert, tokenParts[0])
	if !ok {
		klog.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s", hash, newHash)
		return fmt.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s", hash, newHash)
	}
	// save the ca.crt to file
	ca, err := x509.ParseCertificate(cacert)
	if err != nil {
		klog.Errorf("failed to parse the CA certificate, error: %v", err)
		return fmt.Errorf("failed to parse the CA certificate, error: %v", err)
	}

	if err = certutil.WriteCert(config.Config.TLSCAFile, ca); err != nil {
		klog.Errorf("failed to save the CA certificate to file: %s, error: %v", config.Config.TLSCAFile, err)
		return fmt.Errorf("failed to save the CA certificate to file: %s, error: %v", config.Config.TLSCAFile, err)
	}

	// get the edge.crt
	url = config.Config.HttpServer + certURL
	edgecert, err := certutil.GetEdgeCert(url, cacert, tokenParts[1])
	if err != nil {
		klog.Errorf("failed to get edge certificate from the cloudcore, error: %v", err)
		return fmt.Errorf("failed to get edge certificate from the cloudcore, error: %v", err)
	}
	// save the edge.crt to the file
	cert, _ := x509.ParseCertificate(edgecert)
	if err = certutil.WriteCert(config.Config.TLSCertFile, cert); err != nil {
		klog.Errorf("failed to save the edge certificate to file: %s, error: %v", config.Config.TLSCertFile, err)
		return fmt.Errorf("failed to save the edge certificate to file: %s, error: %v", config.Config.TLSCertFile, err)
	}
	return nil
}

func (eh *EdgeHub) initial() (err error) {

	cloudHubClient, err := clients.GetClient()
	if err != nil {
		return err
	}

	eh.chClient = cloudHubClient

	return nil
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
		beehiveContext.SendToGroup(md, message)
		return nil
	}
	return eh.sendToKeepChannel(message)
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
		sendTimer := time.NewTimer(time.Duration(config.Config.Heartbeat) * time.Second)
		select {
		case response := <-tempChannel:
			sendTimer.Stop()
			beehiveContext.SendResp(response)
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
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeHub RouteToCloud stop")
			return
		default:
		}
		message, err := beehiveContext.Receive(ModuleNameEdgeHub)
		if err != nil {
			klog.Errorf("failed to receive message from edge: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// post message to cloud hub
		err = eh.sendToCloud(message)
		if err != nil {
			klog.Errorf("failed to send message to cloud: %v", err)
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
			BuildRouter(ModuleNameEdgeHub, "resource", "node", "keepalive").
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

func (eh *EdgeHub) pubConnectInfo(isConnected bool) {
	// var info model.Message
	content := connect.CloudConnected
	if !isConnected {
		content = connect.CloudDisconnected
	}

	for _, group := range groupMap {
		message := model.NewMessage("").BuildRouter(message.SourceNodeConnection, group,
			message.ResourceTypeNodeConnection, message.OperationNodeConnection).FillBody(content)
		beehiveContext.SendToGroup(group, *message)
	}
}
