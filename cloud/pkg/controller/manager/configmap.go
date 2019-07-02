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

// Package manager
package manager

import (
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ConfigMapManager manage all events of configmap by SharedInformer
type ConfigMapManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch configmap change
func (cmm *ConfigMapManager) Events() chan watch.Event {
	return cmm.events
}

// NewConfigMapManager create ConfigMapManager by kube clientset and namespace
func NewConfigMapManager(kubeClient *kubernetes.Clientset, namespace string) (*ConfigMapManager, error) {
	lw := cache.NewListWatchFromClient(kubeClient.CoreV1().RESTClient(), "configmaps", namespace, fields.Everything())
	events := make(chan watch.Event, config.ConfigMapEventBuffer)
	rh := NewCommonResourceEventHandler(events)
	si := cache.NewSharedInformer(lw, &v1.ConfigMap{}, 0)
	si.AddEventHandler(rh)
	stopNever := make(chan struct{})
	go si.Run(stopNever)

	return &ConfigMapManager{events: events}, nil
}
