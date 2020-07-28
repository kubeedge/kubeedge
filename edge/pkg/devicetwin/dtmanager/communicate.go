package dtmanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	// ActionCallBack map for action to callback
	ActionCallBack map[string]CallBack
)

// CommWorker deal app response event
type CommWorker struct {
	Worker
	Group string
}

// Start worker
func (cw CommWorker) Start() {
	initActionCallBack()
	for {
		select {
		case msg, ok := <-cw.ReceiverChan:
			klog.V(3).Info("[DtManager] receive msg")
			if !ok {
				klog.V(3).Infof("[DtManager] failed to get msg")
				return
			}
			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := ActionCallBack[dtMsg.Action]; exist {
					_, err := fn(cw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("[DtManager] failed to handle msg %v with %s action, err: %v", dtMsg.Msg, dtMsg.Action, err)
					}
				} else {
					klog.Errorf("[DtManager] failed to handle msg %v with %s action, err: invalid action", dtMsg.Msg, dtMsg.Action)
				}
			}

		case <-time.After(time.Duration(60) * time.Second):
			_, err := cw.checkConfirm(cw.DTContexts, nil)
			klog.V(2).Infof("[DtManager] failed to check confirm, err: %v", err)
		case v, ok := <-cw.HeartBeatChan:
			if !ok {
				klog.V(3).Infof("[DtManager] failed to get heartbeat info")
				return
			}
			if err := cw.DTContexts.HeartBeat(cw.Group, v); err != nil {
				klog.V(2).Infof("[DtManager] failed to handle heartbeat info %v, err: %v", v, err)
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
	beehiveContext.Send(dtcommon.EventHubModule, *msg.(*model.Message))
	return nil, nil
}
func dealSendToCloud(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	if strings.Compare(context.State, dtcommon.Disconnected) == 0 {
		klog.Infof("[DtManager] skip send msg to cloud since disconnected")
		return nil, nil
	}
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, fmt.Errorf("invalid msg format, value: %v", msg)
	}
	beehiveContext.Send(dtcommon.HubModule, *message)
	msgID := message.GetID()
	context.ConfirmMap.Store(msgID, &dttype.DTMessage{Msg: message, Action: dtcommon.SendToCloud, Type: dtcommon.CommModule})
	return nil, nil
}
func dealLifeCycle(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.V(3).Infof("[DtManager] deal lifecycle event")
	message, ok := msg.(*model.Message)
	if !ok {
		return nil, fmt.Errorf("invalid msg format, value: %v", msg)
	}
	connectedInfo := message.Content.(string)
	switch connectedInfo {
	case connect.CloudConnected:
		if strings.Compare(context.State, dtcommon.Disconnected) == 0 {
			_, err := detailRequest(context, msg)
			if err != nil {
				klog.Errorf("[DtManager] detail request: %v", err)
				return nil, err
			}
		}
		context.State = dtcommon.Connected
	case connect.CloudDisconnected:
		context.State = dtcommon.Disconnected
	}

	return nil, nil
}
func dealConfirm(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	klog.V(3).Infof("[DtManager] deal confirm event")
	value, ok := msg.(*model.Message)

	if !ok {
		return nil, fmt.Errorf("invalid msg format, value: %v", msg)
	}

	parentMsgID := value.GetParentID()
	klog.V(2).Infof("[DtManager] confirm msg, id: %s", parentMsgID)
	context.ConfirmMap.Delete(parentMsgID)

	return nil, nil
}

func detailRequest(context *dtcontext.DTContext, msg interface{}) (interface{}, error) {
	// TODO eventid uuid
	getDetail := dttype.GetDetailNode{
		EventType: "group_membership_event",
		EventID:   "123",
		Operation: "detail",
		GroupID:   context.NodeName,
		TimeStamp: time.Now().UnixNano() / 1000000}
	getDetailJSON, marshalErr := json.Marshal(getDetail)
	if marshalErr != nil {
		klog.Errorf("[DtManager] failed to marshal request, err: %#v", marshalErr)
		return nil, marshalErr
	}

	message := context.BuildModelMessage("resource", "", "membership/detail", "get", string(getDetailJSON))
	msgID := message.GetID()
	context.ConfirmMap.Store(msgID, &dttype.DTMessage{Msg: message, Action: dtcommon.SendToCloud, Type: dtcommon.CommModule})
	beehiveContext.Send(dtcommon.HubModule, *message)
	return nil, nil
}

func (cw CommWorker) checkConfirm(context *dtcontext.DTContext, msg interface{}) (interface{}, error) {
	klog.Infof("[DtManager] check confirm event")
	context.ConfirmMap.Range(func(key interface{}, value interface{}) bool {
		dtMsg, ok := value.(*dttype.DTMessage)
		if !ok {
			klog.Errorf("[DtManager] invalid msg format, key: %v, value: %v", key, value)
			context.ConfirmMap.Delete(key)
			return true
		}

		klog.Info("[DtManager] redo task due to no confirm info")
		if fn, exist := ActionCallBack[dtMsg.Action]; exist {
			_, err := fn(cw.DTContexts, dtMsg.Identity, dtMsg.Msg)
			if err != nil {
				klog.Errorf("[DtManager] failed to handle msg %v with %s action, err: %v", dtMsg.Msg, dtMsg.Action, err)
			}
		} else {
			klog.Errorf("[DtManager] failed to handle msg %v with %s action, err: invalid action", dtMsg.Msg, dtMsg.Action)
		}
		return true
	})
	return nil, nil
}
