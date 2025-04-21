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

package downstream

import (
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
)

// NodeJobEventHandler is a common resource event handler for node job
type NodeJobEventHandler struct {
	logger     logr.Logger
	downstream chan<- wrap.NodeJob
}

// Check that NodeJobEventHandler implements the ResourceEventHandler interface
var _ cache.ResourceEventHandler = (*NodeJobEventHandler)(nil)

// NewNodeJobEventHandler creates a new NodeJobEventHandler
func NewNodeJobEventHandler(logger logr.Logger, downstream chan<- wrap.NodeJob) *NodeJobEventHandler {
	return &NodeJobEventHandler{
		logger:     logger,
		downstream: downstream,
	}
}

// OnAdd gets the watched node job addition event, and uses CanDownstreamPhase
// method to determine whether to send the node job wrap to downstream channel.
func (h *NodeJobEventHandler) OnAdd(obj any, isInInitialList bool) {
	downstreamHandler, err := MustGetHandlerWithObj(obj)
	if err != nil {
		h.logger.Error(err, "failed to get downstream handler")
		return
	}
	if isInInitialList && downstreamHandler.CanDownstreamPhase(obj) {
		job, err := wrap.WithEventObj(obj)
		if err != nil {
			h.logger.Error(err, "failed to convert event object to node job")
			return
		}
		h.downstream <- job
	}
}

// OnUpdate gets the watched node job update event, and uses CanDownstreamPhase
// method to determine whether to send the node job wrap to downstream channel.
func (h *NodeJobEventHandler) OnUpdate(_oldObj, newObj any) {
	downstreamHandler, err := MustGetHandlerWithObj(newObj)
	if err != nil {
		h.logger.Error(err, "failed to get downstream handler")
		return
	}
	if downstreamHandler.CanDownstreamPhase(newObj) {
		job, err := wrap.WithEventObj(newObj)
		if err != nil {
			h.logger.Error(err, "failed to convert event object to node job")
			return
		}
		h.downstream <- job
	}
}

// OnDelete gets the watched node job deletion event, and uses InterruptExecutor
// method to interrupt the downstream executor.
func (h *NodeJobEventHandler) OnDelete(obj any) {
	downstreamHandler, err := MustGetHandlerWithObj(obj)
	if err != nil {
		h.logger.Error(err, "failed to get downstream handler")
		return
	}
	downstreamHandler.InterruptExecutor(obj)
}
