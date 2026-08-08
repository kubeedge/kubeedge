/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"encoding/json"
	"fmt"

	nodev1 "k8s.io/api/node/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// RuntimeClassesGetter has a method to return a RuntimeClassInterface.
// A group's client should implement this interface.
type RuntimeClassesGetter interface {
	RuntimeClasses() RuntimeClassInterface
}

// RuntimeClassInterface has methods to work with RuntimeClass resources.
// RuntimeClass is cluster-scoped, so there is no namespace parameter.
type RuntimeClassInterface interface {
	Get(name string) (*nodev1.RuntimeClass, error)
}

type runtimeClasses struct {
	send        SendInterface
	metaService MetaServiceInterface
}

func newRuntimeClasses(s SendInterface) *runtimeClasses {
	return &runtimeClasses{
		send:        s,
		metaService: dbclient.NewMetaService(),
	}
}

func (c *runtimeClasses) Get(name string) (*nodev1.RuntimeClass, error) {
	// RuntimeClass is cluster-scoped: use NullNamespace as the namespace segment.
	resource := fmt.Sprintf("%s/%s/%s", models.NullNamespace, model.ResourceTypeRuntimeClass, name)

	remoteGet := func() (*nodev1.RuntimeClass, error) {
		rcMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
		msg, err := c.send.SendSync(rcMsg)
		if err != nil {
			return nil, fmt.Errorf("get runtimeclass from metaManager failed, err: %v", err)
		}
		errContent, ok := msg.GetContent().(error)
		if ok {
			return nil, errContent
		}
		content, err := msg.GetContentData()
		if err != nil {
			return nil, fmt.Errorf("parse message to runtimeclass failed, err: %v", err)
		}
		return handleRuntimeClassFromMetaManager(content)
	}

	metas, err := c.metaService.QueryMeta("key", resource)
	if err != nil || len(*metas) == 0 {
		return remoteGet()
	}
	return handleRuntimeClassFromMetaDB(metas)
}

func handleRuntimeClassFromMetaDB(lists *[]string) (*nodev1.RuntimeClass, error) {
	if len(*lists) != 1 {
		return nil, fmt.Errorf("runtimeclass length from meta db is %d", len(*lists))
	}

	var rc nodev1.RuntimeClass
	if err := json.Unmarshal([]byte((*lists)[0]), &rc); err != nil {
		return nil, fmt.Errorf("unmarshal message to runtimeclass from db failed, err: %v", err)
	}
	return &rc, nil
}

func handleRuntimeClassFromMetaManager(content []byte) (*nodev1.RuntimeClass, error) {
	var rc nodev1.RuntimeClass
	if err := json.Unmarshal(content, &rc); err != nil {
		return nil, fmt.Errorf("unmarshal message to runtimeclass failed, err: %v", err)
	}
	return &rc, nil
}
