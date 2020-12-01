package client

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"
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
	ListAll() ([]v1.Service, error)
}

type services struct {
	namespace string
	send      SendInterface
}

func newServices(namespace string, s SendInterface) *services {
	return &services{
		namespace: namespace,
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
		return handleServicePodListFromMetaDB(content)
	}
	return handleServicePodListFromMetaManager(content)
}

func handleServicePodListFromMetaDB(content []byte) ([]v1.Pod, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("service length from meta db is %d", len(lists))
	}

	var ps []v1.Pod
	err = json.Unmarshal([]byte(lists[0]), &ps)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service podlist from db failed, err: %v", err)
	}
	return ps, nil
}

func handleServicePodListFromMetaManager(content []byte) ([]v1.Pod, error) {
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
		return handleServiceFromMetaDB(content)
	}
	return handleServiceFromMetaManager(content)
}

func handleServiceFromMetaDB(content []byte) (*v1.Service, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Service list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("service length from meta db is %d", len(lists))
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

func (s *services) ListAll() ([]v1.Service, error) {
	resource := fmt.Sprintf("%s/%s", s.namespace, constants.ResourceTypeService)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.QueryOperation, nil)
	respMsg, err := s.send.SendSync(msg)
	if err != nil {
		return nil, fmt.Errorf("get service list from metaManager failed, err: %v", err)
	}
	var content []byte
	switch respMsg.Content.(type) {
	case []byte:
		content = respMsg.GetContent().([]byte)
	default:
		content, err = json.Marshal(respMsg.Content)
		if err != nil {
			return nil, fmt.Errorf("marshal message to service list failed, err: %v", err)
		}
	}

	if respMsg.GetOperation() == model.ResponseOperation {
		return handleServiceListFromMetaDB(content)
	}
	return handleServiceListFromMetaManager(content)
}

func handleServiceListFromMetaDB(content []byte) ([]v1.Service, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service list from edge db failed, err: %v", err)
	}

	var serviceList []v1.Service
	for i := range lists {
		var s v1.Service
		err = json.Unmarshal([]byte(lists[i]), &s)
		if err != nil {
			return nil, fmt.Errorf("unmarshal message to service from edge db failed, err: %v", err)
		}
		serviceList = append(serviceList, s)
	}
	return serviceList, nil
}

func handleServiceListFromMetaManager(content []byte) ([]v1.Service, error) {
	var serviceList []v1.Service
	err := json.Unmarshal(content, &serviceList)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service list failed, err: %v", err)
	}
	return serviceList, nil
}
