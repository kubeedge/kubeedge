package listener

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"k8s.io/klog/v2"
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

func (mh *MessageHandler) getHandler(source string, resource string) (Handle, error) {
	rs := strings.Split(resource, "/")
	if len(rs) >= 2 && (rs[0] == "streamruleendpoint" || rs[0] == "streamrule") {
		resource = rs[0]
	}
	key := fmt.Sprintf("%s/%s", source, resource)
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
			klog.Info("streamrulecontroller module stop dispatch message")
			return
		default:
		}
		msg, err := beehiveContext.Receive(module)
		if err != nil {
			klog.Errorf("streamrulecontroller module receive message error: %v", err)
			continue
		}
		klog.Infof("streamrulecontroller module receive message: %v", msg)
		err = MessageHandlerInstance.HandleMessage(&msg)
		if err != nil {
			klog.Errorf("streamrulecontroller module handle message error: %v", err)
		}
	}
}

func (mh *MessageHandler) HandleMessage(msg *model.Message) error {
	if msg == nil {
		return fmt.Errorf("nil message error")
	}
	if msg.GetParentID() != "" {
		// skip the response message
		mh.callback(msg)
		return nil
	}
	handler, err := mh.getHandler(msg.GetSource(), msg.GetResource())
	if err != nil {
		klog.Errorf("get handler for message error: %v", err)
		return err
	}
	go func(msg *model.Message) {
		resp, err := handler(msg)
		if err != nil {
			klog.Errorf("handle message error: %v", err)
		}
		if resp != nil {
			if err = resp.(*http.Response).Body.Close(); err != nil {
				klog.Errorf("close response occur error, msgID: %s, reason: %s", msg.GetID(), err.Error())
			}
		}
	}(msg)
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
