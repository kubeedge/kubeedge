package client

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
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
	PersistentVolumesGetter
	PersistentVolumeClaimsGetter
	VolumeAttachmentsGetter
	ListenerGetter
}

type metaClient struct {
	send SendInterface
}

func (m *metaClient) Pods(namespace string) PodsInterface {
	return newPods(namespace, m.send)
}

func (m *metaClient) ConfigMaps(namespace string) ConfigMapsInterface {
	return newConfigMaps(namespace, m.send)
}

func (m *metaClient) Nodes(namespace string) NodesInterface {
	return newNodes(namespace, m.send)
}

func (m *metaClient) NodeStatus(namespace string) NodeStatusInterface {
	return newNodeStatus(namespace, m.send)
}

func (m *metaClient) Secrets(namespace string) SecretsInterface {
	return newSecrets(namespace, m.send)
}

func (m *metaClient) PodStatus(namespace string) PodStatusInterface {
	return newPodStatus(namespace, m.send)
}

//New creates a new metaclient
func (m *metaClient) Endpoints(namespace string) EndpointsInterface {
	return newEndpoints(namespace, m.send)
}

// New Services metaClient
func (m *metaClient) Services(namespace string) ServiceInterface {
	return newServices(namespace, m.send)
}

// New PersistentVolumes metaClient
func (m *metaClient) PersistentVolumes(namespace string) PersistentVolumesInterface {
	return newPersistentVolumes(namespace, m.send)
}

// New PersistentVolumeClaims metaClient
func (m *metaClient) PersistentVolumeClaims(namespace string) PersistentVolumeClaimsInterface {
	return newPersistentVolumeClaims(namespace, m.send)
}

// New VolumeAttachments metaClient
func (m *metaClient) VolumeAttachments(namespace string) VolumeAttachmentsInterface {
	return newVolumeAttachments(namespace, m.send)
}

// New Listener metaClient
func (m *metaClient) Listener() ListenInterface {
	return newListener(m.send)
}

// New creates new metaclient
func New() CoreInterface {
	return &metaClient{
		send: newSend(),
	}
}

//SendInterface is to sync interface
type SendInterface interface {
	SendSync(message *model.Message) (*model.Message, error)
	Send(message *model.Message)
}

type send struct {
}

func newSend() SendInterface {
	return &send{}
}

func (s *send) SendSync(message *model.Message) (*model.Message, error) {
	var err error
	var resp model.Message
	retries := 0
	err = wait.Poll(syncPeriod, syncMsgRespTimeout, func() (bool, error) {
		resp, err = beehiveContext.SendSync(metamanager.MetaManagerModuleName, *message, syncMsgRespTimeout)
		retries++
		if err == nil {
			klog.V(2).Infof("send sync message %s successed and response: %v", message.GetResource(), resp)
			return true, nil
		}
		if retries < 3 {
			klog.Errorf("send sync message %s failed, error:%v, retries: %d", message.GetResource(), err, retries)
			return false, nil
		}
		return true, err
	})
	return &resp, err
}

func (s *send) Send(message *model.Message) {
	beehiveContext.Send(metamanager.MetaManagerModuleName, *message)
}
