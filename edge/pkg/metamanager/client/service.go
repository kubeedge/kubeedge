package client

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"

	v1 "k8s.io/api/core/v1"
)

// ServiceGetter interface
type ServiceGetter interface {
	Services(namespace string) ServiceInterface
}

// ServiceInterface is an interface
type ServiceInterface interface {
	Create(*v1.Service) (*v1.Service, error)
	Update(service *v1.Service) error
	Delete(name string) error
	Get(name string) (*v1.Service, error)
	GetPods(name string) ([]v1.Pod, error)
}

type services struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newServices(namespace string, c *context.Context, s SendInterface) *services {
	return &services{
		namespace: namespace,
		context:   c,
		send:      s,
	}
}

func (s *services) Create(*v1.Service) (*v1.Service, error) {
	return &v1.Service{}, nil
}

func (s *services) Update(service *v1.Service) error {
	return nil
}

func (s *services) Delete(name string) error {
	return nil
}

func (s *services) GetPods(name string) ([]v1.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", s.namespace, model.ResourceTypePodlist, name)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.QueryOperation, nil)
	respMsg, err := s.send.SendSync(msg)
	if err != nil {
		return nil, fmt.Errorf("get service podlist from metaManager failed, err: %v", err)
	}
	var content []byte
	switch respMsg.Content.(type) {
	case []byte:
		content = respMsg.GetContent().([]byte)
	default:
		content, err = json.Marshal(respMsg.Content)
		if err != nil {
			return nil, fmt.Errorf("marshal message to service podlist failed, err: %v", err)
		}
	}

	if respMsg.GetOperation() == model.ResponseOperation {
		return handlerServicePodListFromMetaDB(content)
	}
	return handlerServicePodListFromMetaManager(content)
}

func handlerServicePodListFromMetaDB(content []byte) ([]v1.Pod, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("Service length from meta db is %d", len(lists))
	}

	var ps []v1.Pod
	err = json.Unmarshal([]byte(lists[0]), &ps)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service podlist from db failed, err: %v", err)
	}
	return ps, nil
}

func handlerServicePodListFromMetaManager(content []byte) ([]v1.Pod, error) {
	var ps []v1.Pod
	err := json.Unmarshal(content, &ps)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service podlist failed, err: %v", err)
	}
	return ps, nil
}

func (s *services) Get(name string) (*v1.Service, error) {
	resource := fmt.Sprintf("%s/%s/%s", s.namespace, constants.ResourceTypeService, name)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.QueryOperation, nil)
	respMsg, err := s.send.SendSync(msg)
	if err != nil {
		return nil, fmt.Errorf("get service from metaManager failed, err: %v", err)
	}
	var content []byte
	switch respMsg.Content.(type) {
	case []byte:
		content = respMsg.GetContent().([]byte)
	default:
		content, err = json.Marshal(respMsg.Content)
		if err != nil {
			return nil, fmt.Errorf("marshal message to configmap failed, err: %v", err)
		}
	}

	if respMsg.GetOperation() == model.ResponseOperation {
		return handlerServiceFromMetaDB(content)
	}
	return handleServiceFromMetaManager(content)
}

func handlerServiceFromMetaDB(content []byte) (*v1.Service, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("Service length from meta db is %d", len(lists))
	}

	var s v1.Service
	err = json.Unmarshal([]byte(lists[0]), &s)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service from db failed, err: %v", err)
	}
	return &s, nil
}

func handleServiceFromMetaManager(content []byte) (*v1.Service, error) {
	var s v1.Service
	err := json.Unmarshal(content, &s)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service failed, err: %v", err)
	}
	return &s, nil
}
