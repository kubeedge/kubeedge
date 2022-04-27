package servicebus

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	commonType "github.com/kubeedge/kubeedge/common/types"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
)

type servicebusFactory struct{}

type ServiceBus struct {
	targetPath  string
	servicePort string
	nodeName    string
	TargetURL   string
}

func init() {
	factory := &servicebusFactory{}
	provider.RegisterSource(factory)
	provider.RegisterTarget(factory)
}

func (sf *servicebusFactory) Type() v1.RuleEndpointTypeDef {
	return v1.RuleEndpointTypeServiceBus
}

func (sf *servicebusFactory) GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) provider.Source {
	targetURL, exist := sourceResource[constants.TargetURL]
	if !exist {
		klog.Errorf("source resource attributes \"target_url\" does not exist")
		return nil
	}
	nodeName, exist := sourceResource[constants.NodeName]
	if !exist {
		klog.Errorf("source resource attributes \"node_name\" does not exist")
		return nil
	}
	cli := &ServiceBus{
		nodeName:  nodeName,
		TargetURL: targetURL,
	}

	return cli
}

func (sb *ServiceBus) RegisterListener(handle listener.Handle) error {
	listener.MessageHandlerInstance.AddListener(fmt.Sprintf("servicebus/%v/%v", path.Join("node", sb.nodeName), sb.TargetURL), handle)
	msg := model.NewMessage("")
	msg.SetResourceOperation(fmt.Sprintf("%v/%v", path.Join("node", sb.nodeName), sb.TargetURL), "start")
	msg.SetRoute(modules.RouterSourceServiceBus, modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	return nil
}

func (sb *ServiceBus) UnregisterListener() {
	msg := model.NewMessage("")
	msg.SetResourceOperation(path.Join("node", sb.nodeName, sb.TargetURL), "stop")
	msg.SetRoute(modules.RouterSourceServiceBus, modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	listener.MessageHandlerInstance.RemoveListener(path.Join("servicebus/node", sb.nodeName, sb.TargetURL))
}

func (sb *ServiceBus) Name() string {
	return constants.ServicebusProvider
}

func (sb *ServiceBus) Forward(target provider.Target, data interface{}) (response interface{}, err error) {
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
	httpResp, ok := resp.(*http.Response)
	if ok {
		byteData, _ := io.ReadAll(httpResp.Body)
		beehiveContext.SendToGroup(modules.CloudHubModuleGroup, *message.NewRespByMessage(message, string(byteData)))
	}
	return resp, nil
}

func (sf *servicebusFactory) GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) provider.Target {
	targetPath, exist := targetResource["path"]
	if !exist {
		klog.Errorf("target resource attributes \"targetPath\" does not exist")
		return nil
	}
	cli := &ServiceBus{
		targetPath:  targetPath,
		servicePort: ep.Spec.Properties["service_port"],
	}
	return cli
}

func (sb *ServiceBus) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
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
	resource := "node/" + nodeName + "/" + sb.servicePort + ":"
	if !ok || param == "" {
		resource = resource + sb.targetPath
	} else {
		resource = resource + strings.TrimSuffix(sb.targetPath, "/") + "/" + strings.TrimPrefix(param, "/")
	}
	msg.SetResourceOperation(resource, request.Method)
	msg.FillBody(request)
	msg.SetRoute(modules.RouterSourceServiceBus, modules.UserGroup)
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
