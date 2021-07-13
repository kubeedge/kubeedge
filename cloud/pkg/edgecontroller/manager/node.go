package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

const (
	NodeRoleKey   = "node-role.kubernetes.io/edge"
	NodeRoleValue = ""
)

// NodesManager manage all events of nodes by SharedInformer
type NodesManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch nodes change
func (nm *NodesManager) Events() chan watch.Event {
	return nm.events
}

// NewNodesManager create NodesManager by kube clientset and namespace
func NewNodesManager(si cache.SharedIndexInformer) (*NodesManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.NodeEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &NodesManager{events: events}, nil
}
