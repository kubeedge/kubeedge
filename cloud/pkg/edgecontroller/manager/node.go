package manager

import (
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
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
func NewNodesManager() (*NodesManager, error) {
	events := make(chan watch.Event)
	rh := NewCommonResourceEventHandler(events)
	si := informers.GetGlobalInformers().EdgeNode()
	si.AddEventHandler(rh)

	return &NodesManager{events: events}, nil
}
