package client

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// ConfigMapsGetter has a method to return a ConfigMapInterface.
// A group's client should implement this interface.
type ConfigMapsGetter interface {
	ConfigMaps(namespace string) ConfigMapsInterface
}

// ConfigMapsInterface has methods to work with ConfigMap resources.
type ConfigMapsInterface interface {
	Create(*api.ConfigMap) (*api.ConfigMap, error)
	Update(*api.ConfigMap) error
	Delete(name string) error
	Get(name string) (*api.ConfigMap, error)
}

type configMaps struct {
	namespace string
	send      SendInterface
}

func newConfigMaps(namespace string, s SendInterface) *configMaps {
	return &configMaps{
		send:      s,
		namespace: namespace,
	}
}

func (c *configMaps) Create(*api.ConfigMap) (*api.ConfigMap, error) {
	return nil, nil
}

func (c *configMaps) Update(*api.ConfigMap) error {
	return nil
}

func (c *configMaps) Delete(string) error {
	return nil
}

func (c *configMaps) Get(name string) (*api.ConfigMap, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeConfigmap, name)
	remoteGet := func() (*api.ConfigMap, error) {
		configMapMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
		msg, err := c.send.SendSync(configMapMsg)
		if err != nil {
			return nil, fmt.Errorf("get configmap from metaManager failed, err: %v", err)
		}
		errContent, ok := msg.GetContent().(error)
		if ok {
			return nil, errContent
		}
		content, err := msg.GetContentData()
		if err != nil {
			return nil, fmt.Errorf("parse message to configmap failed, err: %v", err)
		}
		return handleConfigMapFromMetaManager(content)
	}

	metas, err := dao.QueryMeta("key", resource)
	if err != nil || len(*metas) == 0 {
		return remoteGet()
	}

	return handleConfigMapFromMetaDB(metas)
}

func handleConfigMapFromMetaDB(lists *[]string) (*api.ConfigMap, error) {
	if len(*lists) != 1 {
		return nil, fmt.Errorf("ConfigMap length from meta db is %d", len(*lists))
	}

	var configMap api.ConfigMap
	err := json.Unmarshal([]byte((*lists)[0]), &configMap)
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
