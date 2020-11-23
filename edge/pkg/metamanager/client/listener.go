package client

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/constant"
)

// Listener is only used by EdgeMesh. It stores
// the fakeIP of EdgeMesh into edge db. One fakeIP for
// one service.
const (
	DefaultNamespace = "default"
)

// ListenerGetter interface
type ListenerGetter interface {
	Listener() ListenInterface
}

// ListenInterface is an interface
type ListenInterface interface {
	Add(key interface{}, value interface{}) error
	Del(key interface{}) error
	Get(key interface{}) (interface{}, error)
}

type listener struct {
	send SendInterface
}

func newListener(s SendInterface) *listener {
	return &listener{
		send: s,
	}
}

func (ln *listener) Add(key interface{}, value interface{}) error {
	svcName, ok := key.(string)
	if !ok {
		return fmt.Errorf("the key type is invalid")
	}
	resource := fmt.Sprintf("%s/%s/%s", DefaultNamespace, constants.ResourceTypeListener, svcName)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.InsertOperation, value)
	_, err := ln.send.SendSync(msg)
	return err
}

func (ln *listener) Del(key interface{}) error {
	svcName, ok := key.(string)
	if !ok {
		return fmt.Errorf("the key type is invalid")
	}
	resource := fmt.Sprintf("%s/%s/%s", DefaultNamespace, constants.ResourceTypeListener, svcName)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.DeleteOperation, nil)
	_, err := ln.send.SendSync(msg)
	return err
}

func (ln *listener) Get(key interface{}) (interface{}, error) {
	svcName, ok := key.(string)
	if !ok {
		return nil, fmt.Errorf("the key type is invalid")
	}
	resource := fmt.Sprintf("%s/%s/%s", DefaultNamespace, constants.ResourceTypeListener, svcName)
	msg := message.BuildMsg(modules.MetaGroup, "", constant.ModuleNameEdgeMesh, resource, model.QueryOperation, nil)
	respMsg, err := ln.send.SendSync(msg)
	if err != nil {
		return nil, err
	}

	return respMsg.Content, nil
}
