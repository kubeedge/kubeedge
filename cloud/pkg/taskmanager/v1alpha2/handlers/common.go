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

package handlers

import (
	"context"

	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// Handler is the operation abstraction of the node task controller.
type Handler interface {
	// Name returns name of node task
	Name() string

	// Informer returns node task CR of informer for downstream.
	Informer() cache.SharedIndexInformer

	// UpdateNodeActionStatus uses to update the status of node action when obtaining upstream message.
	UpdateNodeActionStatus(ctx context.Context, msg model.Message) error

	// Handle notifications for events of node task CR, OnAdd, OnUpdate and OnDelete.
	cache.ResourceEventHandler
}
