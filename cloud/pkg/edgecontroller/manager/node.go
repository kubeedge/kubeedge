/*
Copyright 2019 The KubeEdge Authors.

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
