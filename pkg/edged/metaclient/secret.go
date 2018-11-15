package metaclient

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/common/message"

	api "k8s.io/api/core/v1"
)

type SecretsGetter interface {
	Secrets(namespace string) SecretsInterface
}

type SecretsInterface interface {
	Create(*api.Secret) (*api.Secret, error)
	Update(*api.Secret) error
	Delete(name string) error
	Get(name string) (*api.Secret, error)
}

type secrets struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newSecrets(namespace string, c *context.Context, s SendInterface) *secrets {
	return &secrets{
		context:   c,
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
	secretMsg := message.BuildMsg(core.MetaGroup, "", core.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(secretMsg)
	if err != nil {
		return nil, fmt.Errorf("get secret from metaManager failed, err: %v", err)
	}

	content, err := json.Marshal(msg.GetContent())
	if err != nil {
		return nil, fmt.Errorf("marshal message content failed, err: %v", err)
	}

	//op := msg.GetOperation()
	if msg.GetOperation() == model.ResponseOperation {
		return handleSecretFromMetaDB(content)
	} else {
		return handleSecretFromMetaManager(content)
	}
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
