package metaclient

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/core/v1"

	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
	"kubeedge/beehive/pkg/core/model"
	"kubeedge/pkg/common/message"
)

type ConfigMapsGetter interface {
	ConfigMaps(namespace string) ConfigMapsInterface
}

type ConfigMapsInterface interface {
	Create(*api.ConfigMap) (*api.ConfigMap, error)
	Update(*api.ConfigMap) error
	Delete(name string) error
	Get(name string) (*api.ConfigMap, error)
}

type configMaps struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newConfigMaps(namespace string, c *context.Context, s SendInterface) *configMaps {
	return &configMaps{
		context:   c,
		send:      s,
		namespace: namespace,
	}
}

func (c *configMaps) Create(cm *api.ConfigMap) (*api.ConfigMap, error) {
	return nil, nil
}

func (c *configMaps) Update(cm *api.ConfigMap) error {
	return nil
}

func (c *configMaps) Delete(name string) error {
	return nil
}

func (c *configMaps) Get(name string) (*api.ConfigMap, error) {

	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeConfigmap, name)
	configMapMsg := message.BuildMsg(core.MetaGroup, "", core.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(configMapMsg)
	if err != nil {
		return nil, fmt.Errorf("get configmap from metaManager failed, err: %v", err)
	}

	content, err := json.Marshal(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal message to configmap failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation {
		return handleConfigMapFromMetaDB(content)
	} else {
		return handleConfigMapFromMetaManager(content)
	}
}

func handleConfigMapFromMetaDB(content []byte) (*api.ConfigMap, error) {
	var lists []string
	err := json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to ConfigMap list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("ConfigMap length from meta db is %d", len(lists))
	}

	var configMap api.ConfigMap
	err = json.Unmarshal([]byte(lists[0]), &configMap)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to ConfigMap from db failed, err: %v", err)
	}
	return &configMap, nil
}

func handleConfigMapFromMetaManager(content []byte) (*api.ConfigMap, error) {
	var configMap api.ConfigMap
	err := json.Unmarshal(content, &configMap)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to ConfigMap failed, err: %v", err)
	}
	return &configMap, nil
}
