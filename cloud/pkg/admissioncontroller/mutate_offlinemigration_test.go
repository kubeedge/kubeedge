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

package admissioncontroller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGeneratePatch(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		name         string
		tolerations  []corev1.Toleration
		expectedLen  int
		expectedOp   string
		expectedPath string
	}{
		{
			name:         "Empty tolerations",
			tolerations:  []corev1.Toleration{},
			expectedLen:  1, // Just the added NodeUnreachable
			expectedOp:   "replace",
			expectedPath: "/spec/tolerations",
		},
		{
			name: "Tolerations without NodeUnreachable",
			tolerations: []corev1.Toleration{
				{Key: "key1", Operator: corev1.TolerationOpExists},
			},
			expectedLen:  2, // 1 existing + 1 added
			expectedOp:   "replace",
			expectedPath: "/spec/tolerations",
		},
		{
			name: "Tolerations with existing NodeUnreachable",
			tolerations: []corev1.Toleration{
				{Key: corev1.TaintNodeUnreachable, Operator: corev1.TolerationOpExists},
				{Key: "key1", Operator: corev1.TolerationOpExists},
			},
			expectedLen:  2, // 1 filtered out + 1 existing + 1 added = 2
			expectedOp:   "replace",
			expectedPath: "/spec/tolerations",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patch := generatePatch(tc.tolerations)
			assert.Len(patch, 1)
			assert.Equal(tc.expectedOp, patch[0].Op)
			assert.Equal(tc.expectedPath, patch[0].Path)
			assert.Len(patch[0].Value, tc.expectedLen)
		})
	}
}

func TestMutateOfflineMigration(t *testing.T) {
	assert := assert.New(t)

	t.Run("Valid Pod with tolerations", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Tolerations: []corev1.Toleration{
					{Key: "key1", Operator: corev1.TolerationOpExists},
				},
			},
		}
		raw, err := json.Marshal(pod)
		assert.NoError(err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{Raw: raw},
			},
		}

		response := mutateOfflineMigration(review)
		assert.True(response.Allowed)
		assert.NotNil(response.Patch)
		assert.Equal(admissionv1.PatchTypeJSONPatch, *response.PatchType)

		var patch []map[string]interface{}
		err = json.Unmarshal(response.Patch, &patch)
		assert.NoError(err)
		assert.Len(patch, 1)
		assert.Equal("replace", patch[0]["op"])
		assert.Equal("/spec/tolerations", patch[0]["path"])
	})

	t.Run("Invalid JSON payload", func(t *testing.T) {
		invalidReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{Raw: []byte("invalid json")},
			},
		}
		
		errResponse := mutateOfflineMigration(invalidReview)
		assert.False(errResponse.Allowed)
		assert.NotNil(errResponse.Result)
		assert.Contains(errResponse.Result.Message, "invalid character")
	})
}

func TestServeOfflineMigration(t *testing.T) {
	assert := assert.New(t)

	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Tolerations: []corev1.Toleration{},
		},
	}
	raw, _ := json.Marshal(pod)

	review := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:    "test-uid",
			Object: runtime.RawExtension{Raw: raw},
		},
	}
	reviewRaw, _ := json.Marshal(review)

	req := httptest.NewRequest(http.MethodPost, "/mutate-offline-migration", bytes.NewReader(reviewRaw))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	serveOfflineMigration(w, req)

	assert.Equal(http.StatusOK, w.Code)

	var respReview admissionv1.AdmissionReview
	err := json.Unmarshal(w.Body.Bytes(), &respReview)
	assert.NoError(err)
	assert.True(respReview.Response.Allowed)
	assert.Equal("test-uid", string(respReview.Response.UID))
}
