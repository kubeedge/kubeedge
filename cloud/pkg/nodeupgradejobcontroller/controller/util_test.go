/*
Copyright 2022 The KubeEdge Authors.

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
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
)

func TestFilterVersion(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		expected     string
		expectResult bool
	}{
		{
			name:         "not match expected version",
			version:      "v1.22.6-kubeedge-v1.9.0",
			expected:     "v1.10.0",
			expectResult: false,
		},
		{
			name:         "not match expected version",
			version:      "v1.22.6-kubeedge-v1.10.0-beta.0.194+77ea462f402efb",
			expected:     "v1.10.0",
			expectResult: false,
		},
		{
			name:         "no right format version",
			version:      "v1.22.6",
			expected:     "v1.10.0",
			expectResult: false,
		},
		{
			name:         "match expected version",
			version:      "v1.22.6-kubeedge-v1.10.0",
			expected:     "v1.10.0",
			expectResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := filterVersion(test.version, test.expected)
			if result != test.expectResult {
				t.Errorf("Got = %v, Want = %v", result, test.expectResult)
			}
		})
	}
}

func TestRemoveDuplicateElement(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "case 1",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "case 2",
			input:    []string{"a", "a", "b", "c", "b", "a", "a"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "case 3",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RemoveDuplicateElement(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Got = %v, Want = %v", result, test.expected)
			}
		})
	}
}

func TestUpdateUpgradeStatus(t *testing.T) {
	upgrade := v1alpha1.NodeUpgradeJob{
		Status: v1alpha1.NodeUpgradeJobStatus{
			Status: []v1alpha1.UpgradeStatus{
				{
					NodeName: "edge-node",
					State:    v1alpha1.Completed,
					History: v1alpha1.History{
						Reason: "the first upgrade",
					},
				},
			},
		},
	}
	upgrade2 := upgrade.DeepCopy()
	upgrade2.Status.Status[0].History = v1alpha1.History{
		Reason: "the second upgrade",
	}

	upgrade3 := upgrade.DeepCopy()
	upgrade3.Status.Status = append(upgrade3.Status.Status, v1alpha1.UpgradeStatus{
		NodeName: "edge-node2",
		State:    v1alpha1.Completed,
		History: v1alpha1.History{
			Reason: "the first upgrade",
		},
	})

	tests := []struct {
		name     string
		upgrade  *v1alpha1.NodeUpgradeJob
		status   *v1alpha1.UpgradeStatus
		expected *v1alpha1.NodeUpgradeJob
	}{
		{
			name:    "case1: first add one node",
			upgrade: &v1alpha1.NodeUpgradeJob{},
			status: &v1alpha1.UpgradeStatus{
				NodeName: "edge-node",
				State:    v1alpha1.Completed,
				History: v1alpha1.History{
					Reason: "the first upgrade",
				},
			},
			expected: upgrade.DeepCopy(),
		},
		{
			name:    "case2: add to one NOT exist node record",
			upgrade: upgrade.DeepCopy(),
			status: &v1alpha1.UpgradeStatus{
				NodeName: "edge-node2",
				State:    v1alpha1.Completed,
				History: v1alpha1.History{
					Reason: "the first upgrade",
				},
			},
			expected: upgrade3,
		},
		{
			name:    "case3: add to one exist node record",
			upgrade: upgrade.DeepCopy(),
			status: &v1alpha1.UpgradeStatus{
				NodeName: "edge-node",
				State:    v1alpha1.Completed,
				History: v1alpha1.History{
					Reason: "the second upgrade",
				},
			},
			expected: upgrade2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newValue := UpdateNodeUpgradeJobStatus(test.upgrade, test.status)
			if !reflect.DeepEqual(newValue, test.expected) {
				t.Errorf("Got = %v, Want = %v", newValue, test.expected)
			}
		})
	}
}

func TestMergeAnnotationUpgradeHistory(t *testing.T) {
	tests := []struct {
		name        string
		origin      string
		fromVersion string
		toVersion   string
		expected    string
	}{
		{
			name:        "case 1: no history record exist",
			origin:      "",
			fromVersion: "v1.10.0",
			toVersion:   "v1.10.1",
			expected:    "v1.10.0->v1.10.1",
		},
		{
			name:        "case 2: 1 history record exist",
			origin:      "v1.10.0->v1.10.1",
			fromVersion: "v1.10.1",
			toVersion:   "v1.10.2",
			expected:    "v1.10.0->v1.10.1;v1.10.1->v1.10.2",
		},
		{
			name:        "case 2: 3 history record exist",
			origin:      "1.10.0->v1.10.1;v1.10.1->v1.10.2;v1.10.2->v1.10.3",
			fromVersion: "v1.10.3",
			toVersion:   "v1.10.4",
			expected:    "v1.10.1->v1.10.2;v1.10.2->v1.10.3;v1.10.3->v1.10.4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := mergeAnnotationUpgradeHistory(test.origin, test.fromVersion, test.toVersion)
			if result != test.expected {
				t.Errorf("Got = %v, Want = %v", result, test.expected)
			}
		})
	}
}
