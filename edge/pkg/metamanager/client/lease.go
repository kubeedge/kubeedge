package client

import (
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

var leaseClientTimeout = 10 * time.Second

// TODO:
// Find a better way to set timeout of SendSync for lease msg.
func SetLeaseClientTimeout(timeout time.Duration) {
	leaseClientTimeout = timeout
}

type LeaseGetter interface {
	Lease(namespace string) LeaseInterface
}

type LeaseInterface interface {
	Create(*coordinationv1.Lease) (*coordinationv1.Lease, error)
	Update(*coordinationv1.Lease) error
	Delete(name string) error
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

func (l *leases) Create(lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	return nil, fmt.Errorf("create operation of lease is not supported")
}

func (l *leases) Update(lease *coordinationv1.Lease) error {
	resource := fmt.Sprintf("%s/%s/%s", l.namespace, model.ResourceTypeLease, lease.Name)
	leaseMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, lease)
	// Update should not use default timeout. In most cases, syncMsgTimeout is much longer than
	// NodeStatusUpdateFrequency. When the heatbeat timestamp of the nodelease msg expires, stop send it to
	// avoid influence on the subsequent update messages.
	_, err := l.send.SendSync(leaseMsg, false, &leaseClientTimeout)
	if err != nil {
		return fmt.Errorf("failed to update lease, %v", err)
	}
	return nil
}

func (l *leases) Delete(name string) error {
	return fmt.Errorf("delete operation of lease is not supported")
}

func (l *leases) Get(name string) (*coordinationv1.Lease, error) {
	return nil, fmt.Errorf("get operation of lease is not supported")
}
