package eventbus

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
)

const (
	publishOperation = "publish"
)

type eventbusFactory struct{}

type EventBus struct {
	pubTopic  string
	subTopic  string
	nodeName  string
	namespace string
}

func init() {
	factory := &eventbusFactory{}
	provider.RegisterSource(factory)
	provider.RegisterTarget(factory)
}

func (factory *eventbusFactory) Type() v1.RuleEndpointTypeDef {
	return v1.RuleEndpointTypeEventBus
}

func (factory *eventbusFactory) GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) provider.Source {
	subTopic, exist := sourceResource[constants.Topic]
	if !exist {
		klog.Errorf("source resource attributes \"topic\" does not exist")
		return nil
	}
	nodeName, exist := sourceResource[constants.NodeName]
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
	listener.MessageHandlerInstance.AddListener(path.Join("bus/node", eb.nodeName, eb.namespace, eb.subTopic), handle)
	msg := model.NewMessage("")
	msg.SetResourceOperation(path.Join("node", eb.nodeName, eb.namespace, eb.subTopic), "subscribe")
	msg.SetRoute("router_eventbus", modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	return nil
}

func (eb *EventBus) UnregisterListener() {
	msg := model.NewMessage("")
	msg.SetResourceOperation(path.Join("node", eb.nodeName, eb.namespace, eb.subTopic), "unsubscribe")
	msg.SetRoute("router_eventbus", modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	listener.MessageHandlerInstance.RemoveListener(path.Join("bus/node", eb.nodeName, eb.namespace, eb.subTopic))
}

func (factory *eventbusFactory) GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) provider.Target {
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
	return constants.EventbusProvider
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
		klog.Errorf("message is send to target failed. msgID: %s, target: %s, err:%v", message.GetID(), target.Name(), err)
		return nil, err
	}
	klog.Infof("message is send to target successfully. msgID: %s, target: %s", message.GetID(), target.Name())
	return resp, nil
}

func (eb *EventBus) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	messageID, ok := data["messageID"].(string)
	if !ok {
		return nil, buildAndLogError("messageID")
	}
	nodeName, ok := data["nodeName"].(string)
	if !ok {
		return nil, buildAndLogError("nodeName")
	}
	body, ok := data["data"].([]byte)
	if !ok {
		return nil, buildAndLogError("data body")
	}
	// use zero value if not found param
	param, _ := data["param"].(string)

	msg := model.NewMessage("")
	msg.BuildHeader(messageID, "", msg.GetTimestamp())
	resource := "node/" + nodeName + "/"
	if param == "" {
		resource = resource + eb.pubTopic
	} else {
		resource = resource + strings.TrimSuffix(eb.pubTopic, "/") + "/" + strings.TrimPrefix(param, "/")
	}
	msg.SetResourceOperation(resource, publishOperation)
	msg.FillBody(string(body))
	msg.SetRoute("router_eventbus", modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	return nil, nil
}

func buildAndLogError(key string) error {
	err := fmt.Errorf("data transform failed, %s type is not matched or value is nil", key)
	klog.Error(err.Error())
	return err
}
