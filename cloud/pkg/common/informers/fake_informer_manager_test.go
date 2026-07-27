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

package informers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewFakeInformerManager(t *testing.T) {
	manager := NewFakeInformerManager()
	assert.NotNil(t, manager)
}

func TestGetKubeInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	assert.NotNil(t, manager.GetKubeInformerFactory())
}

func TestGetKubeEdgeInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	assert.NotNil(t, manager.GetKubeEdgeInformerFactory())
}

func TestGetDynamicInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	assert.NotNil(t, manager.GetDynamicInformerFactory())
}

func TestStart(t *testing.T) {
	manager := NewFakeInformerManager()
	// Start is a no-op in fakeManager, ensuring it doesn't panic.
	manager.Start(nil)
}

func TestGetInformerPair(t *testing.T) {
	tests := []struct {
		name    string
		gvr     schema.GroupVersionResource
		wantErr bool
	}{
		{
			name:    "valid pods gvr",
			gvr:     schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			wantErr: false,
		},
		{
			name:    "invalid gvr",
			gvr:     schema.GroupVersionResource{Group: "invalid", Version: "v1", Resource: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFakeInformerManager()
			pair, err := manager.GetInformerPair(tt.gvr)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, pair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pair)
				
				// Test caching by calling it again
				pair2, err2 := manager.GetInformerPair(tt.gvr)
				assert.NoError(t, err2)
				assert.Same(t, pair, pair2, "GetInformerPair() did not return cached pair")
			}
		})
	}
}

func TestGetLister(t *testing.T) {
	tests := []struct {
		name    string
		gvr     schema.GroupVersionResource
		wantErr bool
	}{
		{
			name:    "valid pods gvr",
			gvr:     schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			wantErr: false,
		},
		{
			name:    "invalid gvr",
			gvr:     schema.GroupVersionResource{Group: "invalid", Version: "v1", Resource: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFakeInformerManager()
			lister, err := manager.GetLister(tt.gvr)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, lister)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, lister)
			}
		})
	}
}

func TestEdgeNode(t *testing.T) {
	manager := NewFakeInformerManager()
	// EdgeNode logs an error and returns nil
	node := manager.EdgeNode()
	assert.Nil(t, node)
}
