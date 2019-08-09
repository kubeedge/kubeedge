package manager

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
func NewNodesManager(kubeClient *kubernetes.Clientset, namespace string) (*NodesManager, error) {
	set := labels.Set{NodeRoleKey: NodeRoleValue}
	selector := labels.SelectorFromSet(set)
	optionModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = selector.String()
	}
	lw := cache.NewFilteredListWatchFromClient(kubeClient.CoreV1().RESTClient(), "nodes", namespace, optionModifier)
	events := make(chan watch.Event)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Node{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &NodesManager{events: events}, nil
}
