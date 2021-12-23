package client

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

//SecretsGetter is interface to get client secrets
type SecretsGetter interface {
	Secrets(namespace string) SecretsInterface
}

//SecretsInterface is interface for client secret
type SecretsInterface interface {
	Create(*api.Secret) (*api.Secret, error)
	Update(*api.Secret) error
	Delete(name string) error
	Get(name string) (*api.Secret, error)
}

type secrets struct {
	namespace string
	send      SendInterface
}

func newSecrets(namespace string, s SendInterface) *secrets {
	return &secrets{
		send:      s,
		namespace: namespace,
	}
}

func (c *secrets) Create(cm *api.Secret) (*api.Secret, error) {
	return nil, nil
}

func (c *secrets) Update(cm *api.Secret) error {
	return nil
}

func (c *secrets) Delete(name string) error {
	return nil
}

func (c *secrets) Get(name string) (*api.Secret, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeSecret, name)
	secretMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(secretMsg)
	if err != nil {
		return nil, fmt.Errorf("get secret from metaManager failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to secret failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handleSecretFromMetaDB(content)
	}
	//else
	return handleSecretFromMetaManager(content)
}

func handleSecretFromMetaDB(content []byte) (*api.Secret, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to secret list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("secret length from meta db is %d", len(lists))
	}

	var secret api.Secret
	err = json.Unmarshal([]byte(lists[0]), &secret)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to secret from db failed, err: %v", err)
	}
	return &secret, nil
}

func handleSecretFromMetaManager(content []byte) (*api.Secret, error) {
	var secret api.Secret
	err := json.Unmarshal(content, &secret)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to secret failed, err: %v", err)
	}
	return &secret, nil
}
