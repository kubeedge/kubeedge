/*
Copyright 2022 The KubeEdge Authors.

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

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// LeasesGetter to get lease interface
type LeasesGetter interface {
	Leases(namespace string) LeasesInterface
}

// LeasesInterface is interface for client leases
type LeasesInterface interface {
	Create(lease *coordinationv1.Lease) (*coordinationv1.Lease, error)
	Update(lease *coordinationv1.Lease) (*coordinationv1.Lease, error)
	Get(name string) (*coordinationv1.Lease, error)
}

type leases struct {
	namespace string
	send      SendInterface
}

func newLeases(namespace string, s SendInterface) *leases {
	return &leases{
		send:      s,
		namespace: namespace,
	}
}

// LeaseResp represents lease response from the api-server
type LeaseResp struct {
	Object *coordinationv1.Lease
	Err    *apierrors.StatusError
}

func (c *leases) Create(lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeLease, lease.Name)
	leaseMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, lease)
	resp, err := c.send.SendSync(leaseMsg)
	if err != nil {
		return nil, fmt.Errorf("create lease failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to lease failed, err: %v", err)
	}
	return handleLeaseResp(content)
}

func (c *leases) Update(lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeLease, lease.Name)
	leaseMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, lease)
	resp, err := c.send.SendSync(leaseMsg)
	if err != nil {
		return nil, fmt.Errorf("update lease failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to lease failed, err: %v", err)
	}
	return handleLeaseResp(content)
}

func (c *leases) Get(name string) (*coordinationv1.Lease, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeLease, name)
	leaseMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	resp, err := c.send.SendSync(leaseMsg)
	if err != nil {
		return nil, fmt.Errorf("query lease failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to lease failed, err: %v", err)
	}
	return handleLeaseResp(content)
}

func handleLeaseResp(content []byte) (*coordinationv1.Lease, error) {
	var leaseResp LeaseResp
	err := json.Unmarshal(content, &leaseResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to lease failed, err: %v", err)
	}

	return leaseResp.Object, leaseResp.Err
}
