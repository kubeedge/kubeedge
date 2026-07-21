/*
Copyright 2026 The KubeEdge Authors.

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

package context

import (
	"context"
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestWithEdgeNode(t *testing.T) {
	nodeID := "test-node"
	ctx := context.Background()

	newCtx := WithEdgeNode(ctx, nodeID)

	userVal := newCtx.Value(authenticationv1.ImpersonateUserHeader)
	if userVal != constants.NodesUserPrefix+nodeID {
		t.Errorf("WithEdgeNode() user = %v, want %v", userVal, constants.NodesUserPrefix+nodeID)
	}

	groupVal := newCtx.Value(authenticationv1.ImpersonateGroupHeader)
	if groupVal != constants.NodesGroup {
		t.Errorf("WithEdgeNode() group = %v, want %v", groupVal, constants.NodesGroup)
	}
}

func TestFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		nodeID   string
		wantNode bool
	}{
		{
			name:     "valid node resource",
			resource: "node/node1/pod/pod1",
			nodeID:   "node1",
			wantNode: true,
		},
		{
			name:     "invalid resource prefix",
			resource: "pod/pod1/node/node1",
			wantNode: false,
		},
		{
			name:     "invalid length",
			resource: "node",
			wantNode: false,
		},
		{
			name:     "empty resource",
			resource: "",
			wantNode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := model.Message{
				Router: model.MessageRoute{
					Resource: tt.resource,
				},
			}
			ctx := context.Background()
			newCtx := FromMessage(ctx, msg)

			userVal := newCtx.Value(authenticationv1.ImpersonateUserHeader)
			if tt.wantNode {
				if userVal != constants.NodesUserPrefix+tt.nodeID {
					t.Errorf("FromMessage() user = %v, want %v", userVal, constants.NodesUserPrefix+tt.nodeID)
				}
			} else {
				if userVal != nil {
					t.Errorf("FromMessage() expected no node injection, got user = %v", userVal)
				}
			}
		})
	}
}
