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

package admissioncontroller

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/api/apis/operations/v1alpha1"
)

func TestAdmitNodeUpgradeJob(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		operation       admissionv1.Operation
		upgrade         *v1alpha1.NodeUpgradeJob
		oldUpgrade      *v1alpha1.NodeUpgradeJob
		expectedAllowed bool
		expectedError   string
	}{
		{
			name:      "Valid Create",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: true,
		},
		{
			name:      "Invalid Version",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "1.0.0",
					NodeNames: []string{"node1"},
				},
			},
			expectedAllowed: false,
			expectedError:   "invalid version 1.0.0",
		},
		{
			name:      "Invalid Semver",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0",
					NodeNames: []string{"node1"},
				},
			},
			expectedAllowed: false,
			expectedError:   "invalid version v1.0",
		},
		{
			name:      "Invalid image",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version: "v1.0.0",
					Image:   "invalid-image",
				},
			},
			expectedAllowed: false,
			expectedError:   "invalid image repo invalid-image",
		},
		{
			name:      "No NodeNames and LabelSelector",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version: "v1.0.0",
				},
			},
			expectedAllowed: false,
			expectedError:   "both NodeNames and LabelSelector are NOT specified",
		},
		{
			name:      "Both NodeNames and LabelSelector",
			operation: admissionv1.Create,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:       "v1.0.0",
					NodeNames:     []string{"node1"},
					LabelSelector: &metav1.LabelSelector{},
				},
			},
			expectedAllowed: false,
			expectedError:   "both NodeNames and LabelSelector are specified",
		},
		{
			name:      "Valid Update",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: true,
		},
		{
			name:      "Invalid Update - Non-version Spec Change",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2", "node3"},
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: false,
			expectedError:   "spec fields other than version and allowDowngrade are not allowed to update once it's created",
		},
		{
			name:      "Valid Update - Version Upgrade",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.1",
					NodeNames: []string{"node1", "node2"},
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: true,
		},
		{
			name:      "Invalid Update - Version Downgrade Without AllowDowngrade",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.16.3",
					NodeNames: []string{"node1", "node2"},
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.17.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: false,
			expectedError:   "version change from v1.17.0 to v1.16.3 is a downgrade",
		},
		{
			name:      "Valid Update - Version Downgrade With AllowDowngrade",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:        "v1.16.3",
					NodeNames:      []string{"node1", "node2"},
					AllowDowngrade: true,
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.17.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedAllowed: true,
		},
		{
			name:      "Invalid Update - Version Change After Execution Started",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.1",
					NodeNames: []string{"node1", "node2"},
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
				Status: v1alpha1.NodeUpgradeJobStatus{
					State: "Checking",
				},
			},
			expectedAllowed: false,
			expectedError:   "version cannot be changed once the upgrade has started executing (current state: Checking)",
		},
		{
			name:      "Valid Update - AllowDowngrade Only Change After Execution Started",
			operation: admissionv1.Update,
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:        "v1.0.0",
					NodeNames:      []string{"node1", "node2"},
					AllowDowngrade: true,
				},
			},
			oldUpgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
				Status: v1alpha1.NodeUpgradeJobStatus{
					State: "Upgrading",
				},
			},
			expectedAllowed: true,
		},
		{
			name:            "Valid Delete",
			operation:       admissionv1.Delete,
			expectedAllowed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: tc.operation,
				},
			}

			if tc.upgrade != nil {
				raw, _ := json.Marshal(tc.upgrade)
				review.Request.Object = runtime.RawExtension{Raw: raw}
			}

			if tc.oldUpgrade != nil {
				raw, _ := json.Marshal(tc.oldUpgrade)
				review.Request.OldObject = runtime.RawExtension{Raw: raw}
			}

			response := admitNodeUpgradeJob(review)

			assert.Equal(tc.expectedAllowed, response.Allowed)
			if tc.expectedError != "" {
				assert.Contains(response.Result.Message, tc.expectedError)
			} else {
				assert.Nil(response.Result)
			}
		})
	}
}

func TestValidateNodeUpgradeJob(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name        string
		upgrade     *v1alpha1.NodeUpgradeJob
		expectedErr string
	}{
		{
			name: "Valid upgrade job",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1", "node2"},
				},
			},
			expectedErr: "",
		},
		{
			name: "Invalid version",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "1.0.0",
					NodeNames: []string{"node1"},
				},
			},
			expectedErr: "invalid version 1.0.0",
		},
		{
			name: "Invalid image",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version: "v1.0.0",
					Image:   "invalid-image",
				},
			},
			expectedErr: "invalid image repo invalid-image",
		},
		{
			name: "Invalid version (not semver compatible)",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0",
					NodeNames: []string{"node1"},
				},
			},
			expectedErr: "invalid version v1.0",
		},
		{
			name: "Missing both NodeNames and LabelSelector",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version: "v1.0.0",
				},
			},
			expectedErr: "both NodeNames and LabelSelector are NOT specified",
		},
		{
			name: "Both NodeNames and LabelSelector specified",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:       "v1.0.0",
					NodeNames:     []string{"node1"},
					LabelSelector: &metav1.LabelSelector{},
				},
			},
			expectedErr: "both NodeNames and LabelSelector are specified",
		},
		{
			name: "Valid upgrade job",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:       "v1.0.0",
					LabelSelector: &metav1.LabelSelector{},
				},
			},
			expectedErr: "",
		},
		{
			name: "Valid strategy type",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
					Strategy:  &v1alpha1.UpdateStrategy{Type: v1alpha1.CanaryUpdateStrategyType},
				},
			},
			expectedErr: "",
		},
		{
			name: "Invalid strategy type",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
					Strategy:  &v1alpha1.UpdateStrategy{Type: "Unknown"},
				},
			},
			expectedErr: "invalid strategy type Unknown, must be one of AtOnce, Rolling, Canary",
		},
		{
			name: "Valid strategy maxUnavailable",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
					Strategy: &v1alpha1.UpdateStrategy{
						Type:           v1alpha1.RollingUpdateStrategyType,
						MaxUnavailable: func() *int32 { v := int32(2); return &v }(),
					},
				},
			},
			expectedErr: "",
		},
		{
			name: "Zero strategy maxUnavailable",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
					Strategy: &v1alpha1.UpdateStrategy{
						MaxUnavailable: func() *int32 { v := int32(0); return &v }(),
					},
				},
			},
			expectedErr: "invalid strategy maxUnavailable 0, must be at least 1",
		},
		{
			name: "Negative strategy maxUnavailable",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
					Strategy: &v1alpha1.UpdateStrategy{
						MaxUnavailable: func() *int32 { v := int32(-1); return &v }(),
					},
				},
			},
			expectedErr: "invalid strategy maxUnavailable -1, must be at least 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNodeUpgradeJob(tc.upgrade)
			if tc.expectedErr == "" {
				assert.NoError(err)
			} else {
				assert.Error(err)
				assert.Contains(err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestAdmissionResponse(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		inputError      error
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name:            "No error",
			inputError:      nil,
			expectedAllowed: true,
			expectedMessage: "",
		},
		{
			name:            "With error",
			inputError:      errors.New("validation failed"),
			expectedAllowed: false,
			expectedMessage: "validation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := admissionResponse(tc.inputError)

			assert.NotNil(response)
			assert.Equal(tc.expectedAllowed, response.Allowed)

			if tc.inputError != nil {
				assert.NotNil(response.Result)
				assert.Equal(tc.expectedMessage, response.Result.Message)
			} else {
				assert.Nil(response.Result)
			}
		})
	}
}

func TestMutatingNodeUpgradeJob(t *testing.T) {
	assert := assert.New(t)

	upgrade := &v1alpha1.NodeUpgradeJob{
		Spec: v1alpha1.NodeUpgradeJobSpec{
			Version:   "v1.0.0",
			NodeNames: []string{"node1", "node2"},
		},
	}
	raw, err := json.Marshal(upgrade)
	assert.NoError(err)

	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: raw},
		},
	}
	response := mutatingNodeUpgradeJob(review)

	assert.True(response.Allowed)
	assert.NotNil(response.Patch)
	assert.Equal(admissionv1.PatchTypeJSONPatch, *response.PatchType)

	// Unmarshal and check the patch
	var patch []map[string]interface{}
	err = json.Unmarshal(response.Patch, &patch)
	assert.NoError(err)
	assert.Len(patch, 3)
	assert.Equal("add", patch[0]["op"])
	assert.Equal("/spec/concurrency", patch[0]["path"])
	assert.Equal(float64(1), patch[0]["value"])
	assert.Equal("add", patch[1]["op"])
	assert.Equal("/spec/timeoutSeconds", patch[1]["path"])
	assert.Equal(float64(300), patch[1]["value"])
	assert.Equal("add", patch[2]["op"])
	assert.Equal("/spec/strategy", patch[2]["path"])
	assert.Equal(map[string]interface{}{
		"type":           "Rolling",
		"maxUnavailable": float64(1),
	}, patch[2]["value"])
}

func TestGenerateNodeUpgradeJobPatch(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		spec          v1alpha1.NodeUpgradeJobSpec
		expectedPatch []patchValue
	}{
		{
			name: "Concurrency, TimeoutSeconds and Strategy all specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				Concurrency:    2,
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
				Strategy: &v1alpha1.UpdateStrategy{
					Type:           v1alpha1.CanaryUpdateStrategyType,
					MaxUnavailable: func() *int32 { v := int32(2); return &v }(),
				},
			},
			expectedPatch: []patchValue{},
		},
		{
			name: "None specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:   "v1.0.0",
				NodeNames: []string{"node1"},
			},
			expectedPatch: []patchValue{
				{
					Op:    "add",
					Path:  "/spec/concurrency",
					Value: 1,
				},
				{
					Op:    "add",
					Path:  "/spec/timeoutSeconds",
					Value: func() *uint32 { v := uint32(300); return &v }(),
				},
				{
					Op:   "add",
					Path: "/spec/strategy",
					Value: &v1alpha1.UpdateStrategy{
						Type:           v1alpha1.RollingUpdateStrategyType,
						MaxUnavailable: func() *int32 { v := int32(1); return &v }(),
					},
				},
			},
		},
		{
			name: "Concurrency specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
				Strategy: &v1alpha1.UpdateStrategy{
					Type:           v1alpha1.CanaryUpdateStrategyType,
					MaxUnavailable: func() *int32 { v := int32(2); return &v }(),
				},
			},
			expectedPatch: []patchValue{
				{
					Op:    "add",
					Path:  "/spec/concurrency",
					Value: 1,
				},
			},
		},
		{
			name: "TimeoutSeconds specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:     "v1.0.0",
				NodeNames:   []string{"node1"},
				Concurrency: 2,
				Strategy: &v1alpha1.UpdateStrategy{
					Type:           v1alpha1.CanaryUpdateStrategyType,
					MaxUnavailable: func() *int32 { v := int32(2); return &v }(),
				},
			},
			expectedPatch: []patchValue{
				{
					Op:    "add",
					Path:  "/spec/timeoutSeconds",
					Value: func() *uint32 { v := uint32(300); return &v }(),
				},
			},
		},
		{
			name: "Strategy type missing",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				Concurrency:    2,
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
				Strategy: &v1alpha1.UpdateStrategy{
					MaxUnavailable: func() *int32 { v := int32(2); return &v }(),
				},
			},
			expectedPatch: []patchValue{
				{
					Op:    "add",
					Path:  "/spec/strategy/type",
					Value: v1alpha1.RollingUpdateStrategyType,
				},
			},
		},
		{
			name: "Strategy maxUnavailable missing",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				Concurrency:    2,
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
				Strategy: &v1alpha1.UpdateStrategy{
					Type: v1alpha1.CanaryUpdateStrategyType,
				},
			},
			expectedPatch: []patchValue{
				{
					Op:    "add",
					Path:  "/spec/strategy/maxUnavailable",
					Value: func() *int32 { v := int32(1); return &v }(),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patch := generateNodeUpgradeJobPatch(tc.spec)
			assert.Equal(tc.expectedPatch, patch)
		})
	}
}

func TestValidateNodeUpgradeJobAllowsOptionalAndValidImage(t *testing.T) {
	cases := []struct {
		name    string
		upgrade *v1alpha1.NodeUpgradeJob
	}{
		{
			name: "empty image is allowed",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					NodeNames: []string{"node1"},
				},
			},
		},
		{
			name: "valid image repo is allowed",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0.0",
					Image:     "kubeedge/installation-package:v1.23.1",
					NodeNames: []string{"node1"},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateNodeUpgradeJob(tt.upgrade); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestGenerateNodeUpgradeJobPatchReturnsEmptyWhenDefaultsSpecified(t *testing.T) {
	timeoutSeconds := uint32(600)
	maxUnavailable := int32(1)
	patch := generateNodeUpgradeJobPatch(v1alpha1.NodeUpgradeJobSpec{
		Version:        "v1.0.0",
		NodeNames:      []string{"node1"},
		Concurrency:    2,
		TimeoutSeconds: &timeoutSeconds,
		Strategy: &v1alpha1.UpdateStrategy{
			Type:           v1alpha1.RollingUpdateStrategyType,
			MaxUnavailable: &maxUnavailable,
		},
	})

	if len(patch) != 0 {
		t.Fatalf("expected empty patch, got %#v", patch)
	}
}
