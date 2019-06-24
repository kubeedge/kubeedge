package manager

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const (
	NodeRoleKey   = "node-role.kubernetes.io/edge"
	NodeRoleValue = ""
)

// NodesManager manage all events of nodes by SharedInformer
type NodesManager struct {
	events <-chan watch.Event
}

// Events return the channel save events from watch nodes change
func (nm *NodesManager) Events() <-chan watch.Event {
	return nm.events
}

// NewNodesManager create NodesManager by kube clientset and namespace
func NewNodesManager(kubeClient *kubernetes.Clientset, namespace string) (*NodesManager, error) {
	nodes := kubeClient.CoreV1().Nodes()
	watchInterface, err := nodes.Watch(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	events := watchInterface.ResultChan()
	nm := &NodesManager{events: events}
	return nm, nil
}
