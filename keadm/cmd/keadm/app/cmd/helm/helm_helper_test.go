/*
Copyright 2024 The KubeEdge Authors.

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
package helm

import (
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func TestMergeProfileValues(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		sets    []string
		wantErr bool
		errMsg  string
	}{
		{
			name: "load cloudcore values.yaml",
			file: "charts/cloudcore/values.yaml",
		},
		{
			name: "load profile values",
			file: "profiles/version.yaml",
		},
		{
			name:    "invalid profile path",
			file:    "nonexistent.yaml",
			wantErr: true,
			errMsg:  "failed to read build in profile",
		},
		{
			name: "with valid set values",
			file: "charts/cloudcore/values.yaml",
			sets: []string{"key1=value1"},
		},
		{
			name:    "with invalid set syntax",
			file:    "charts/cloudcore/values.yaml",
			sets:    []string{"key1..[]=value1"}, // Invalid path syntax
			wantErr: true,
			errMsg:  "failed to parse --set data",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vals, err := MergeProfileValues(c.file, c.sets)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for case %s but got nil", c.name)
				}
				if c.errMsg != "" && !contains(err.Error(), c.errMsg) {
					t.Fatalf("expected error containing %q, got %v", c.errMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to load helm values, err: %v", err)
			}
			if len(vals) == 0 {
				t.Fatal("the value returned is empty")
			}
		})
	}
}

func TestNewHelper(t *testing.T) {
	cases := []struct {
		name       string
		kubeconfig string
		namespace  string
		wantErr    bool
	}{
		{
			name:       "valid basic config",
			kubeconfig: "",
			namespace:  "default",
			wantErr:    false,
		},
		{
			name:       "empty namespace",
			kubeconfig: "",
			namespace:  "",
			wantErr:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			helper, err := NewHelper(c.kubeconfig, c.namespace)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if helper == nil {
				t.Fatal("helper is nil")
			}
			if helper.GetConfig() == nil {
				t.Fatal("config is nil")
			}
		})
	}
}

func TestHelper_GetValues(t *testing.T) {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "memory", debug); err != nil {
		t.Fatalf("failed to initialize action configuration: %v", err)
	}

	helper := &Helper{cfg: actionConfig}
	_, err := helper.GetValues("test-release")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestMergeExternValues(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		sets    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nonexistent file",
			file:    "nonexistent.yaml",
			wantErr: true,
			errMsg:  "failed to read build in profile",
		},
		{
			name:    "with nonexistent values file",
			file:    "test/values.yaml",
			sets:    []string{"key1=value1"},
			wantErr: true,
			errMsg:  "failed to read build in profile",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := MergeExternValues(c.file, c.sets)
			if !c.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for case %s but got nil", c.name)
			}
			if c.errMsg != "" && !contains(err.Error(), c.errMsg) {
				t.Fatalf("expected error containing %q, got %v", c.errMsg, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}

// Debug function for Helm configuration
func debug(_ string, _ ...interface{}) {}
