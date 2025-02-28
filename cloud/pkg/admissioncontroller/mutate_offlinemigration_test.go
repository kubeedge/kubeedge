package admissioncontroller

import (
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMutateOfflineMigration(t *testing.T) {
	testCases := []struct {
		name                       string
		inputTolerations           []corev1.Toleration
		expectedTolerations        int
		checkUnreachableToleration bool
	}{
		{
			name:                       "Pod with No Tolerations",
			inputTolerations:           []corev1.Toleration{},
			expectedTolerations:        1,
			checkUnreachableToleration: true,
		},
		{
			name: "Pod with Existing Tolerations",
			inputTolerations: []corev1.Toleration{
				{
					Key:      "test-key",
					Operator: corev1.TolerationOpEqual,
					Value:    "test-value",
				},
			},
			expectedTolerations:        2,
			checkUnreachableToleration: true,
		},
		{
			name: "Pod with Existing NodeUnreachable Toleration",
			inputTolerations: []corev1.Toleration{
				{
					Key:      "test-key",
					Operator: corev1.TolerationOpEqual,
					Value:    "test-value",
				},
				{
					Key:      corev1.TaintNodeUnreachable,
					Operator: corev1.TolerationOpExists,
				},
			},
			expectedTolerations:        2,
			checkUnreachableToleration: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a sample pod with input tolerations
			pod := corev1.Pod{
				Spec: corev1.PodSpec{
					Tolerations: tc.inputTolerations,
				},
			}

			// Marshal the pod
			rawPod, err := json.Marshal(pod)
			if err != nil {
				t.Fatalf("Failed to marshal pod: %v", err)
			}

			// Create admission review
			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: rawPod,
					},
				},
			}

			// Call the function
			response := mutateOfflineMigration(review)

			// Verify basic response
			if !response.Allowed {
				t.Errorf("Expected admission to be allowed, got false")
			}

			// Always expect a patch (based on implementation)
			if len(response.Patch) == 0 {
				t.Errorf("Expected patch to be generated, but got empty patch")
			}

			// Unmarshal the patch
			var patchValues []patchMapValue
			err = json.Unmarshal(response.Patch, &patchValues)
			if err != nil {
				t.Fatalf("Failed to unmarshal patch: %v", err)
			}

			// Verify patch structure
			if len(patchValues) != 1 {
				t.Errorf("Expected 1 patch operation, got %d", len(patchValues))
			}

			if patchValues[0].Op != "replace" {
				t.Errorf("Expected patch operation to be 'replace', got %s", patchValues[0].Op)
			}

			if patchValues[0].Path != "/spec/tolerations" {
				t.Errorf("Expected patch path to be '/spec/tolerations', got %s", patchValues[0].Path)
			}

			// Unmarshal the Value into a slice of Tolerations
			var tolerations []corev1.Toleration
			tolerationsJSON, err := json.Marshal(patchValues[0].Value)
			if err != nil {
				t.Fatalf("Failed to marshal tolerations: %v", err)
			}

			err = json.Unmarshal(tolerationsJSON, &tolerations)
			if err != nil {
				t.Fatalf("Failed to unmarshal tolerations: %v", err)
			}

			// Verify number of tolerations
			if len(tolerations) != tc.expectedTolerations {
				t.Errorf("Expected %d tolerations, got %d",
					tc.expectedTolerations,
					len(tolerations))
			}

			// Check for NodeUnreachable toleration if required
			if tc.checkUnreachableToleration {
				foundUnreachableToleration := false
				for _, toleration := range tolerations {
					if toleration.Key == corev1.TaintNodeUnreachable &&
						toleration.Operator == corev1.TolerationOpExists {
						foundUnreachableToleration = true
						break
					}
				}
				if !foundUnreachableToleration {
					t.Errorf("Expected NodeUnreachable toleration not found")
				}
			}
		})
	}
}

// Test error handling
func TestMutateOfflineMigrationErrorHandling(t *testing.T) {
	// Create an invalid admission review with non-pod object
	invalidReview := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte("invalid json"),
			},
		},
	}

	// Call the function
	response := mutateOfflineMigration(invalidReview)

	// Verify error response
	if response.Allowed {
		t.Errorf("Expected admission to be not allowed for invalid input")
	}

	if response.Result == nil {
		t.Errorf("Expected error result for invalid input")
	}
}
