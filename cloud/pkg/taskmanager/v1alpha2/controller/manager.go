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

package controller

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/controller/handlers"
)

type Manager struct {
	handlers   map[string]handlers.Handler
	statusChan <-chan model.Message
}

func NewManager(statusChan <-chan model.Message) *Manager {
	return &Manager{
		statusChan: statusChan,
		handlers:   make(map[string]handlers.Handler),
	}
}

func (m *Manager) Registry(h handlers.Handler) *Manager {
	m.handlers[h.Name()] = h
	return m
}

func (m *Manager) RegisterEventHandler() error {
	for k, v := range m.handlers {
		if _, err := v.Informer().AddEventHandler(v); err != nil {
			return fmt.Errorf("failed to add event handler of %s, err: %v", k, err)
		}
	}
	return nil
}

func (m *Manager) DoUpstream(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Info("stop watching upstream messages of node task")
			return
		case msg, ok := <-m.statusChan:
			if !ok {
				klog.Info("the upstream status channel has been closed")
				return
			}
			handler, ok := m.handlers[msg.GetOperation()]
			if !ok {
				klog.Warningf("invalid node task operation %s", msg.GetOperation())
			}
			if err := handler.UpdateNodeActionStatus(ctx, msg); err != nil {
				klog.Warningf("failed to update node task status, err: %v", err)
				continue
			}
		}
	}
}
