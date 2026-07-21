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

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewFakeInformerManager(t *testing.T) {
	manager := NewFakeInformerManager()
	if manager == nil {
		t.Fatalf("NewFakeInformerManager() returned nil")
	}

	fm, ok := manager.(*fakeManager)
	if !ok {
		t.Fatalf("NewFakeInformerManager() did not return *fakeManager")
	}

	if fm.dynamicClient == nil {
		t.Errorf("fakeManager.dynamicClient is nil")
	}
	if fm.kubeClient == nil {
		t.Errorf("fakeManager.kubeClient is nil")
	}
	if fm.kubeEdgeClient == nil {
		t.Errorf("fakeManager.kubeEdgeClient is nil")
	}
}

func TestGetKubeInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	factory := manager.GetKubeInformerFactory()
	if factory == nil {
		t.Errorf("GetKubeInformerFactory() returned nil")
	}
}

func TestGetKubeEdgeInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	factory := manager.GetKubeEdgeInformerFactory()
	if factory == nil {
		t.Errorf("GetKubeEdgeInformerFactory() returned nil")
	}
}

func TestGetDynamicInformerFactory(t *testing.T) {
	manager := NewFakeInformerManager()
	factory := manager.GetDynamicInformerFactory()
	if factory == nil {
		t.Errorf("GetDynamicInformerFactory() returned nil")
	}
}

func TestStart(t *testing.T) {
	manager := NewFakeInformerManager()
	// Start is a no-op, just ensure it doesn't panic
	manager.Start(nil)
}

func TestGetInformerPair(t *testing.T) {
	tests := []struct {
		name    string
		gvr     schema.GroupVersionResource
		wantNil bool
	}{
		{
			name:    "valid pods gvr",
			gvr:     schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			wantNil: false,
		},
		{
			name:    "invalid gvr",
			gvr:     schema.GroupVersionResource{Group: "invalid", Version: "v1", Resource: "invalid"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFakeInformerManager()
			pair, err := manager.GetInformerPair(tt.gvr)
			if err != nil {
				t.Fatalf("GetInformerPair() unexpected error: %v", err)
			}
			if tt.wantNil && pair != nil {
				t.Errorf("GetInformerPair() expected nil, got %v", pair)
			}
			if !tt.wantNil && pair == nil {
				t.Errorf("GetInformerPair() expected non-nil pair")
			}
			
			// Test caching by calling it again
			if !tt.wantNil {
				pair2, err2 := manager.GetInformerPair(tt.gvr)
				if err2 != nil {
					t.Errorf("GetInformerPair() unexpected error on second call: %v", err2)
				}
				if pair != pair2 {
					t.Errorf("GetInformerPair() did not return cached pair")
				}
			}
		})
	}
}

func TestGetLister(t *testing.T) {
	tests := []struct {
		name    string
		gvr     schema.GroupVersionResource
		wantNil bool
	}{
		{
			name:    "valid pods gvr",
			gvr:     schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			wantNil: false,
		},
		{
			name:    "invalid gvr",
			gvr:     schema.GroupVersionResource{Group: "invalid", Version: "v1", Resource: "invalid"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewFakeInformerManager()
			lister, err := manager.GetLister(tt.gvr)
			if err != nil {
				t.Fatalf("GetLister() unexpected error: %v", err)
			}
			if tt.wantNil && lister != nil {
				t.Errorf("GetLister() expected nil, got %v", lister)
			}
			if !tt.wantNil && lister == nil {
				t.Errorf("GetLister() expected non-nil lister")
			}
		})
	}
}

func TestEdgeNode(t *testing.T) {
	manager := NewFakeInformerManager()
	// EdgeNode logs an error and returns nil
	node := manager.EdgeNode()
	if node != nil {
		t.Errorf("EdgeNode() expected nil, got %v", node)
	}
}
