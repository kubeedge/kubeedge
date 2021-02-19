package servicebus

import (
	"errors"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	"k8s.io/klog/v2"
	"strings"
)

type servicebusFactory struct{}

type ServiceBus struct {
	targetPath string
	namespace string
	servicePort string
}

func init()  {
	factory := &servicebusFactory{}
	provider.RegisterTarget(factory)
}

func (factory *servicebusFactory) Type() string {
	return constants.ServicebusEndpoint
}

func (factory *servicebusFactory) GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) provider.Target {
	targetPath, exist := targetResource["path"]
	if !exist {
		klog.Errorf("target resource attributes \"targetPath\" does not exist")
		return nil
	}
	cli := &ServiceBus{
		targetPath: targetPath,
		namespace: ep.Namespace,
		servicePort: ep.Spec.Properties["service_port"],
	}
	return cli
}

func (eb *ServiceBus) Name() string {
	return constants.ServicebusProvider
}

func (eb *ServiceBus) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	var response *model.Message
	messageID, ok := data["messageID"].(string)
	body, ok := data["data"].([]byte)
	param, ok := data["param"].(string)
	nodeName, ok := data["nodeName"].(string)
	operation, ok := data["operation"].(string)
	if !ok {
		err := errors.New("data transform failed")
		klog.Error(err.Error())
		return nil, err
	}
	msg := model.NewMessage("")
	msg.BuildHeader(messageID, "", msg.GetTimestamp())
	resource := "node/" + nodeName + "/"+eb.servicePort+":"
	if !ok || param == "" {
		resource = resource + eb.targetPath
	} else {
		resource = resource + strings.TrimSuffix(eb.targetPath, "/") + "/" + strings.TrimPrefix(param, "/")
	}
	msg.SetResourceOperation(resource, operation)
	msg.FillBody(string(body))
	msg.SetRoute("router_eventbus", modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	if stop != nil {
		listener.MessageHandlerInstance.SetCallback(messageID, func(message *model.Message) {
			response = message
			stop <- struct{}{}
		})
		<-stop
		listener.MessageHandlerInstance.DelCallback(messageID)
	}
	return response, nil
}


