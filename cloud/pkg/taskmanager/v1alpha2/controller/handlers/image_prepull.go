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

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

// Check ImagePrePullJobHandler implements Handler interface
var _ Handler = (*ImagePrePullJobHandler)(nil)

type ImagePrePullJobHandler struct {
}

func NewImagePrePullJobHandler() *ImagePrePullJobHandler {
	return &ImagePrePullJobHandler{}
}

func (h *ImagePrePullJobHandler) Name() string {
	return operationsv1alpha2.ResourceImagePrePullJob
}

func (ImagePrePullJobHandler) Informer() cache.SharedIndexInformer {
	return informers.GetInformersManager().
		GetKubeEdgeInformerFactory().
		Operations().
		V1alpha2().
		ImagePrePullJobs().
		Informer()
}

func (h *ImagePrePullJobHandler) UpdateNodeActionStatus(ctx context.Context, msg model.Message) error {
	// TODO: ...
	return nil
}

func (h *ImagePrePullJobHandler) OnAdd(obj any, isInInitialList bool) {
	// TODO: ...
}

func (h *ImagePrePullJobHandler) OnUpdate(old, new any) {
	// TODO: ...
}

func (h *ImagePrePullJobHandler) OnDelete(obj any) {
	// TODO: ...
}
