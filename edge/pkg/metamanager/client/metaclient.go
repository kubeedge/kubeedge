package client

import (
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	syncPeriod         = 10 * time.Millisecond
	syncMsgRespTimeout = 1 * time.Minute
)

//CoreInterface is interface of mataclient
type CoreInterface interface {
	PodsGetter
	PodStatusGetter
	ConfigMapsGetter
	NodesGetter
	NodeStatusGetter
	SecretsGetter
	EndpointsGetter
	ServiceGetter
}

type metaClient struct {
	context *context.Context
	send    SendInterface
}

func (m *metaClient) Pods(namespace string) PodsInterface {
	return newPods(namespace, m.context, m.send)
}

func (m *metaClient) ConfigMaps(namespace string) ConfigMapsInterface {
	return newConfigMaps(namespace, m.context, m.send)
}

func (m *metaClient) Nodes(namespace string) NodesInterface {
	return newNodes(namespace, m.context, m.send)
}

func (m *metaClient) NodeStatus(namespace string) NodeStatusInterface {
	return newNodeStatus(namespace, m.context, m.send)
}

func (m *metaClient) Secrets(namespace string) SecretsInterface {
	return newSecrets(namespace, m.context, m.send)
}

func (m *metaClient) PodStatus(namespace string) PodStatusInterface {
	return newPodStatus(namespace, m.context, m.send)
}

//New creates a new metaclient
func (m *metaClient) Endpoints(namespace string) EndpointsInterface {
	return newEndpoints(namespace, m.context, m.send)
}

// New Services metaClient
func (m *metaClient) Services(namespace string) ServiceInterface {
	return newServices(namespace, m.context, m.send)
}

// New creates new metaclient
func New(c *context.Context) CoreInterface {
	return &metaClient{
		context: c,
		send:    newSend(c),
	}
}

//SendInterface is to sync interface
type SendInterface interface {
	SendSync(message *model.Message) (*model.Message, error)
}

type send struct {
	context *context.Context
}

func newSend(c *context.Context) SendInterface {
	return &send{c}
}

func (s *send) SendSync(message *model.Message) (*model.Message, error) {
	var err error
	var resp model.Message
	retries := 0
	err = wait.Poll(syncPeriod, syncMsgRespTimeout, func() (bool, error) {
		resp, err = s.context.SendSync(metamanager.MetaManagerModuleName, *message, syncMsgRespTimeout)
		retries++
		if err == nil {
			log.LOGGER.Infof("send sync message %s successed and response: %v", message.GetResource(), resp)
			return true, nil
		}
		if retries < 3 {
			log.LOGGER.Errorf("send sync message %s failed, error:%v, retries: %d", message.GetResource(), err, retries)
			return false, nil
		}
		return true, err

	})
	return &resp, err
}
