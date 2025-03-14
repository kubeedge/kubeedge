package listener

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
)

var MessageHandlerInstance = &MessageHandler{}

type MessageHandler struct {
	handlers         sync.Map
	callbackHandlers sync.Map
}

func (mh *MessageHandler) AddListener(key interface{}, han Handle) {
	resource, ok := key.(string)
	if !ok {
		return
	}
	mh.handlers.Store(resource, han)
}
func (mh *MessageHandler) RemoveListener(key interface{}) {
	resource, ok := key.(string)
	if !ok {
		return
	}
	mh.handlers.Delete(resource)
}

func (mh *MessageHandler) matchedMqttTopic(topic string) (string, bool) {
	var candidateRes string
	mh.handlers.Range(func(key, value interface{}) bool {
		pathReg, ok := key.(string)
		if !ok {
			return true
		}
		if utils.IsMqttTopicMatch(pathReg, topic) {
			candidateRes = pathReg
			return false
		}
		return true
	})
	if candidateRes == "" {
		return candidateRes, false
	}
	return candidateRes, true
}

func (mh *MessageHandler) getHandler(source string, resource string) (Handle, error) {
	rs := strings.Split(resource, "/")
	if len(rs) >= 2 && (rs[0] == model.ResourceTypeRuleEndpoint || rs[0] == model.ResourceTypeRule) {
		resource = rs[0]
	}
	key := fmt.Sprintf("%s/%s", source, resource)
	if source == constants.EventBusResource {
		candidate, exists := mh.matchedMqttTopic(key)
		if !exists {
			return nil, fmt.Errorf("no rule match for key %s", key)
		}
		key = candidate
	}

	v, exist := mh.handlers.Load(key)
	if !exist {
		return nil, errors.New("no handler for message")
	}
	handle, ok := v.(Handle)
	if !ok {
		return nil, fmt.Errorf("handler invalid, key is %s", key)
	}
	return handle, nil
}

func Process(module string) {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("router module stop dispatch message")
			return
		default:
		}
		msg, err := beehiveContext.Receive(module)
		if err != nil {
			klog.Errorf("get a message, header:%+v router:%+v, err: %v", msg.Header, msg.Router, err)
			continue
		}
		klog.Infof("get a message, header:%+v router:%+v", msg.Header, msg.Router)
		err = MessageHandlerInstance.HandleMessage(&msg)
		if err != nil {
			klog.Errorf("Process msg error: %s.", err.Error())
		}
	}
}

func (mh *MessageHandler) HandleMessage(message *model.Message) error {
	if message == nil {
		return fmt.Errorf("nil message error")
	}
	if message.GetParentID() != "" {
		mh.callback(message)
		return nil
	}
	handler, err := mh.getHandler(message.GetSource(), message.GetResource())
	if err != nil {
		klog.Errorf("No handler for message.msgID: %s, source: %s, resource %s can't find candidate", message.GetID(), message.GetSource(), message.GetResource())
		return err
	}
	go func(message *model.Message) {
		resp, err := handler(message)
		if err != nil {
			klog.Errorf("handle message occur error, msgID: %s, reason: %s", message.GetID(), err.Error())
		}
		if resp != nil {
			if err = resp.(*http.Response).Body.Close(); err != nil {
				klog.Errorf("close response occur error, msgID: %s, reason: %s", message.GetID(), err.Error())
			}
		}
	}(message)

	return nil
}

func (mh *MessageHandler) SetCallback(messageID string, callback func(message *model.Message)) {
	mh.callbackHandlers.Store(messageID, callback)
}

func (mh *MessageHandler) DelCallback(messageID string) {
	mh.callbackHandlers.Delete(messageID)
}

func (mh *MessageHandler) callback(message *model.Message) {
	pID := message.GetParentID()
	v, exist := mh.callbackHandlers.Load(pID)
	if exist {
		callback, ok := v.(func(message *model.Message))
		if !ok {
			klog.Warningf("invalid convert to model.Message")
			return
		}
		callback(message)
	}
	mh.callbackHandlers.Delete(pID)
}
