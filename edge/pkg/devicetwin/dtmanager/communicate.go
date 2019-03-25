package dtmanager

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"

	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	//ActionCallBack map for action to callback
	ActionCallBack map[string]CallBack
)

//CommWorker deal app response event
type CommWorker struct {
	Worker
	Group string
}

//Start worker
func (cw CommWorker) Start() {
	initActionCallBack()
	for {
		select {
		case msg, ok := <-cw.ReceiverChan:
			log.LOGGER.Info("receive msg commModule")
			if !ok {
				return
			}
			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := ActionCallBack[dtMsg.Action]; exist {
					_, err := fn(cw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						log.LOGGER.Errorf("CommModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					log.LOGGER.Errorf("CommModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}

		case <-time.After(time.Duration(60) * time.Second):
			cw.checkConfirm(cw.DTContexts, nil)
		case v, ok := <-cw.HeartBeatChan:
			if !ok {
				return
			}
			if err := cw.DTContexts.HeartBeat(cw.Group, v); err != nil {
				return
			}
		}
	}
}

func initActionCallBack() {
	ActionCallBack = make(map[string]CallBack)
	ActionCallBack[dtcommon.SendToCloud] = dealSendToCloud
	ActionCallBack[dtcommon.SendToEdge] = dealSendToEdge
	ActionCallBack[dtcommon.LifeCycle] = dealLifeCycle
	ActionCallBack[dtcommon.Confirm] = dealConfirm
}

func dealSendToEdge(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	context.ModulesContext.Send(dtcommon.EventHubModule, *msg.(*model.Message))
	return nil, nil
}
func dealSendToCloud(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	if strings.Compare(context.State, dtcommon.Disconnected) == 0 {
		log.LOGGER.Infof("Disconnected with cloud,not send msg to cloud")
		return nil, nil
	}
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}
	context.ModulesContext.Send(dtcommon.HubModule, *message)
	msgID := message.GetID()
	context.ConfirmMap.Store(msgID, &dttype.DTMessage{Msg: message, Action: dtcommon.SendToCloud, Type: dtcommon.CommModule})
	return nil, nil
}
func dealLifeCycle(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	log.LOGGER.Infof("CONNECTED EVENT")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, errors.New("msg not Message type")
	}
	connectedInfo, _ := (message.Content.(string))
	if strings.Compare(connectedInfo, connect.CloudConnected) == 0 {
		if strings.Compare(context.State, dtcommon.Disconnected) == 0 {
			_, err := detailRequest(context, msg)
			if err != nil {
				log.LOGGER.Errorf("detail request: %v", err)
				return nil, err
			}
		}
		context.State = dtcommon.Connected
	} else if strings.Compare(connectedInfo, connect.CloudDisconnected) == 0 {
		context.State = dtcommon.Disconnected
	}
	return nil, nil
}
func dealConfirm(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	log.LOGGER.Infof("CONFIRM EVENT")
	value, ok := msg.(*model.Message)

	if ok {
		parentMsgID := value.GetParentID()
		log.LOGGER.Infof("CommModule deal confirm msgID %s", parentMsgID)
		context.ConfirmMap.Delete(parentMsgID)
	} else {
		return nil, errors.New("CommModule deal confirm, type not correct")
	}
	return nil, nil
}

func detailRequest(context *dtcontext.DTContext, msg interface{}) (interface{}, error) {
	//todo eventid uuid
	getDetail := dttype.GetDetailNode{
		EventType: "group_membership_event",
		EventID:   "123",
		Operation: "detail",
		GroupID:   context.NodeID,
		TimeStamp: time.Now().UnixNano() / 1000000}
	getDetailJSON, marshalErr := json.Marshal(getDetail)
	if marshalErr != nil {
		log.LOGGER.Errorf("Marshal request error while request detail, err: %#v", marshalErr)
		return nil, marshalErr
	}

	message := context.BuildModelMessage("resource", "", "membership/detail", "get", string(getDetailJSON))
	log.LOGGER.Info("Request detail")
	msgID := message.GetID()
	context.ConfirmMap.Store(msgID, &dttype.DTMessage{Msg: message, Action: dtcommon.SendToCloud, Type: dtcommon.CommModule})
	context.ModulesContext.Send(dtcommon.HubModule, *message)
	return nil, nil
}

func (cw CommWorker) checkConfirm(context *dtcontext.DTContext, msg interface{}) (interface{}, error) {
	log.LOGGER.Info("CheckConfirm")
	context.ConfirmMap.Range(func(key interface{}, value interface{}) bool {
		dtmsg, ok := value.(*dttype.DTMessage)
		log.LOGGER.Info("has msg")
		if !ok {

		} else {
			log.LOGGER.Info("redo task due to no recv")
			if fn, exist := ActionCallBack[dtmsg.Action]; exist {
				_, err := fn(cw.DTContexts, dtmsg.Identity, dtmsg.Msg)
				if err != nil {
					log.LOGGER.Errorf("CommModule deal %s event failed: %v", dtmsg.Action, err)
				}
			} else {
				log.LOGGER.Errorf("CommModule deal %s event failed, not found callback", dtmsg.Action)
			}

		}
		return true
	})
	return nil, nil
}
