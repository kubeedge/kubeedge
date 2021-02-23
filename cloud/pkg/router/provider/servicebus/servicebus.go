package servicebus

import (
	"errors"
	"net/http"
	"strings"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	commonType "github.com/kubeedge/kubeedge/common/types"
)

type servicebusFactory struct{}

type ServiceBus struct {
	targetPath  string
	namespace   string
	servicePort string
}

func init() {
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
		targetPath:  targetPath,
		namespace:   ep.Namespace,
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
	param, ok := data["param"].(string)
	nodeName, ok := data["nodeName"].(string)
	request := commonType.HTTPRequest{}
	request.Method, ok = data["method"].(string)
	request.Header, ok = data["header"].(http.Header)
	request.Body, ok = data["data"].([]byte)
	if !ok {
		err := errors.New("data transform failed")
		klog.Error(err.Error())
		return nil, err
	}
	msg := model.NewMessage("")
	msg.BuildHeader(messageID, "", msg.GetTimestamp())
	resource := "node/" + nodeName + "/" + eb.servicePort + ":"
	if !ok || param == "" {
		resource = resource + eb.targetPath
	} else {
		resource = resource + strings.TrimSuffix(eb.targetPath, "/") + "/" + strings.TrimPrefix(param, "/")
	}
	msg.SetResourceOperation(resource, request.Method)
	msg.FillBody(request)
	msg.SetRoute("router_servicebus", modules.UserGroup)
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
