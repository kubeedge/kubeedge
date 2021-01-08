package eventbus

import (
	"errors"
	"fmt"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	"k8s.io/klog/v2"
	"strings"
)

type EventBusFactory struct{}

type EventBus struct {
	pubTopic  string
	subTopic  string
	nodeName  string
	namespace string
}

func init() {
	factory := &EventBusFactory{}
	provider.RegisterSource(factory)
	provider.RegisterTarget(factory)
}

func (factory *EventBusFactory) Type() string {
	return "eventbus"
}

func (factory *EventBusFactory) GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) provider.Source {

	subTopic, exist := sourceResource["topic"]
	if !exist {
		klog.Errorf("source resource attributes \"topic\" does not exist")
		return nil
	}
	nodeName, exist := sourceResource["node_name"]
	if !exist {
		klog.Errorf("source resource attributes \"node_name\" does not exist")
		return nil
	}
	cli := &EventBus{
		subTopic:  subTopic,
		namespace: ep.Namespace,
		nodeName:  nodeName,
	}

	return cli
}

func (eb *EventBus) RegisterListener(handle listener.Handle) error {
	listener.MessageHandlerInstance.AddListener(fmt.Sprintf("%s/node/%s/%s/%s", "bus", eb.nodeName, eb.namespace, eb.subTopic), handle)
	msg := model.NewMessage("")
	msg.SetResourceOperation(fmt.Sprintf("node/%s/%s/%s", eb.nodeName, eb.namespace, eb.subTopic), "subscribe")
	msg.SetRoute("router_eventbus", "user")
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	return nil
}

func (eb *EventBus) UnregisterListener() {
	msg := model.NewMessage("")
	msg.SetResourceOperation(fmt.Sprintf("node/%s/%s/%s", eb.nodeName, eb.namespace, eb.subTopic), "unsubscribe")
	msg.SetRoute("router_eventbus", "user")
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	listener.MessageHandlerInstance.RemoveListener(fmt.Sprintf("%s/node/%s/%s/%s", "bus", eb.nodeName, eb.namespace, eb.subTopic))
}

func (factory *EventBusFactory) GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) provider.Target {
	pubTopic, exist := targetResource["topic"]
	if !exist {
		klog.Errorf("target resource attributes \"topic\" does not exist")
		return nil
	}
	cli := &EventBus{
		pubTopic:  pubTopic,
		namespace: ep.Namespace,
	}
	return cli
}

func (eb *EventBus) Name() string {
	return "eventbus"
}

func (*EventBus) Forward(target provider.Target, data interface{}) (response interface{}, err error) {
	message := data.(*model.Message)
	res := make(map[string]interface{})
	v, ok := message.Content.(string)
	if !ok {
		return nil, errors.New("message content invalid convert to string")
	}
	res["data"] = []byte(v)
	resp, err := target.GoToTarget(res, nil)
	if err != nil {
		klog.Errorf("message is send to target fail. msgID: %s, target: %s, err:%v", message.GetID(), target.Name(), err)
		return nil, err
	}
	klog.Infof("message is send to target successfully. msgID: %s, target: %s", message.GetID(), target.Name())
	return resp, nil
}

func (eb *EventBus) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	var response *model.Message
	messageId, ok := data["messageID"].(string)
	body, ok := data["data"].([]byte)
	param, ok := data["param"].(string)
	nodeName, ok := data["nodeName"].(string)
	if !ok {
		err := errors.New("data transform failed")
		klog.Error(err.Error())
		return nil, err
	}
	msg := model.NewMessage("")
	msg.BuildHeader(messageId, "", msg.GetTimestamp())
	resource := "node/" + nodeName + "/"
	if !ok || param == "" {
		resource = resource + eb.pubTopic
	} else {
		resource = resource + strings.TrimSuffix(eb.pubTopic, "/") + "/" + strings.TrimPrefix(param, "/")
	}
	msg.SetResourceOperation(resource, "publish")
	msg.FillBody(string(body))
	msg.SetRoute("router_eventbus", "user")
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	if stop != nil {
		listener.MessageHandlerInstance.SetCallback(messageId, func(message *model.Message) {
			response = message
			stop <- struct{}{}
		})
		select {
		case <-stop:
			listener.MessageHandlerInstance.DelCallback(messageId)
		}
	}
	return response, nil
}
