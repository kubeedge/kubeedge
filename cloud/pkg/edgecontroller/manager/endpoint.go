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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
)

// EndpointsManager manage all events of endpoints by SharedInformer
type EndpointsManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch endpoints change
func (sm *EndpointsManager) Events() chan watch.Event {
	return sm.events
}

// NewEndpointsManager create EndpointsManager by kube clientset and namespace
func NewEndpointsManager(kubeClient *kubernetes.Clientset, namespace string) (*EndpointsManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "endpoints", namespace, fields.Everything())
	events := make(chan watch.Event, config.Config.Buffer.EndpointsEvent)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.Endpoints{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &EndpointsManager{events: events}, nil
}
