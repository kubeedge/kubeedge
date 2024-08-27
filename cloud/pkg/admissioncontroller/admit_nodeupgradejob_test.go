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
			expectedError:   "version must begin with prefix 'v'",
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
			expectedError:   "version is not a semver compatible version",
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
			expectedError:   "both NodeNames and LabelSelctor are NOT specified",
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
			expectedError:   "both NodeNames and LabelSelctor are specified",
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
			name:      "Invalid Update - Spec Change",
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
			expectedAllowed: false,
			expectedError:   "spec fields are not allowed to update once it's created",
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
			expectedErr: "version must begin with prefix 'v'",
		},
		{
			name: "Invalid version (not semver compatible)",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version:   "v1.0",
					NodeNames: []string{"node1"},
				},
			},
			expectedErr: "version is not a semver compatible version",
		},
		{
			name: "Missing both NodeNames and LabelSelector",
			upgrade: &v1alpha1.NodeUpgradeJob{
				Spec: v1alpha1.NodeUpgradeJobSpec{
					Version: "v1.0.0",
				},
			},
			expectedErr: "both NodeNames and LabelSelctor are NOT specified",
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
			expectedErr: "both NodeNames and LabelSelctor are specified",
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
	assert.Len(patch, 2)
	assert.Equal("replace", patch[0]["op"])
	assert.Equal("/spec/concurrency", patch[0]["path"])
	assert.Equal(float64(1), patch[0]["value"])
	assert.Equal("replace", patch[1]["op"])
	assert.Equal("/spec/timeoutSeconds", patch[1]["path"])
	assert.Equal(float64(300), patch[1]["value"])
}

func TestGenerateNodeUpgradeJobPatch(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		spec          v1alpha1.NodeUpgradeJobSpec
		expectedPatch []patchValue
	}{
		{
			name: "Concurrency and TimeoutSeconds both specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				Concurrency:    2,
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
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
					Op:    "replace",
					Path:  "/spec/concurrency",
					Value: 1,
				},
				{
					Op:    "replace",
					Path:  "/spec/timeoutSeconds",
					Value: func() *uint32 { v := uint32(300); return &v }(),
				},
			},
		},
		{
			name: "Concurrency specified",
			spec: v1alpha1.NodeUpgradeJobSpec{
				Version:        "v1.0.0",
				NodeNames:      []string{"node1"},
				TimeoutSeconds: func() *uint32 { v := uint32(600); return &v }(),
			},
			expectedPatch: []patchValue{
				{
					Op:    "replace",
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
			},
			expectedPatch: []patchValue{
				{
					Op:    "replace",
					Path:  "/spec/timeoutSeconds",
					Value: func() *uint32 { v := uint32(300); return &v }(),
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
