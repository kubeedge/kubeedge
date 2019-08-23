package client

import (
	"encoding/json"
	"fmt"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

// PersistentVolumeClaimsGetter is interface to get client PersistentVolumeClaims
type PersistentVolumeClaimsGetter interface {
	PersistentVolumeClaims(namespace string) PersistentVolumeClaimsInterface
}

// PersistentVolumeClaimsInterface is interface for client PersistentVolumeClaims
type PersistentVolumeClaimsInterface interface {
	Create(*api.PersistentVolumeClaim) (*api.PersistentVolumeClaim, error)
	Update(*api.PersistentVolumeClaim) error
	Delete(name string) error
	Get(name string, options metav1.GetOptions) (*api.PersistentVolumeClaim, error)
}

type persistentvolumeclaims struct {
	namespace string
	context   *context.Context
	send      SendInterface
}

func newPersistentVolumeClaims(n string, c *context.Context, s SendInterface) *persistentvolumeclaims {
	return &persistentvolumeclaims{
		namespace: n,
		context:   c,
		send:      s,
	}
}

func (c *persistentvolumeclaims) Create(pvc *api.PersistentVolumeClaim) (*api.PersistentVolumeClaim, error) {
	return nil, nil
}

func (c *persistentvolumeclaims) Update(pvc *api.PersistentVolumeClaim) error {
	return nil
}

func (c *persistentvolumeclaims) Delete(name string) error {
	return nil
}

func (c *persistentvolumeclaims) Get(name string, options metav1.GetOptions) (*api.PersistentVolumeClaim, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, "persistentvolumeclaim", name)
	pvcMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(pvcMsg)
	if err != nil {
		return nil, fmt.Errorf("get persistentvolumeclaim from metaManager failed, err: %v", err)
	}

	var content []byte
	switch msg.Content.(type) {
	case []byte:
		content = msg.GetContent().([]byte)
	default:
		content, err = json.Marshal(msg.GetContent())
		if err != nil {
			return nil, fmt.Errorf("marshal message to persistentvolumeclaim failed, err: %v", err)
		}
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handlePersistentVolumeClaimFromMetaDB(content)
	}
	return handlePersistentVolumeClaimFromMetaManager(content)
}

func handlePersistentVolumeClaimFromMetaDB(content []byte) (*api.PersistentVolumeClaim, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to persistentvolumeclaim list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("persistentvolumeclaim length from meta db is %d", len(lists))
	}

	var pvc *api.PersistentVolumeClaim
	err = json.Unmarshal([]byte(lists[0]), &pvc)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to persistentvolumeclaim from db failed, err: %v", err)
	}
	return pvc, nil
}

func handlePersistentVolumeClaimFromMetaManager(content []byte) (*api.PersistentVolumeClaim, error) {
	var pvc *api.PersistentVolumeClaim
	err := json.Unmarshal(content, &pvc)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to persistentvolumeclaim failed, err: %v", err)
	}
	return pvc, nil
}
