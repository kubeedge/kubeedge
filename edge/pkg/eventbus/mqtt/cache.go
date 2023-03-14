package mqtt

import (
	"fmt"
	"path"
	"sync"

	"k8s.io/klog/v2"
)

type ResponseTopic struct {
	callbackTopic sync.Map
}

var responseHandler = &ResponseTopic{}

func AddCallbackTopic(messageID, topic string) {
	ackTopic := path.Join(topic, "result")
	klog.V(2).InfoS("add callback topic", "msgID", messageID, "ackTopic", ackTopic)
	responseHandler.callbackTopic.Store(messageID, ackTopic)
}

func LoadAndDeleteCallbackTopic(messageID string) (string, error) {
	v, exist := responseHandler.callbackTopic.LoadAndDelete(messageID)
	if !exist {
		return "", fmt.Errorf("ackTopic not found")
	}
	ackTopic, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("ackTopic is not string type")
	}
	return ackTopic, nil
}
