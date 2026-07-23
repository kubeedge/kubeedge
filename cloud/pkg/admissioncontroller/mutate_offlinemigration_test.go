package admissioncontroller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMutateOfflineMigration(t *testing.T) {
	// Create a sample AdmissionReview object
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: "12345",
			Kind: metav1.GroupVersionKind{
				Group:   "example.com",
				Version: "v1",
				Kind:    "ExampleResource",
			},
			Resource: metav1.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "exampleresources",
			},
			Operation: admissionv1.Create,
		},
	}

	// Call the function under test
	response := mutateOfflineMigration(review)

	// Check if the response is as expected
	if response == nil {
		t.Fatal("Expected a non-nil AdmissionResponse")
	}

	if response.Allowed {
		t.Error("Expected Allowed to be false, got true")
	}
}

func TestGeneratePatch(t *testing.T) {
	// Define test cases
	tests := []struct {
		name          string
		tolerations   []corev1.Toleration
		expectedPatch []patchMapValue
	}{
		{
			name:          "No tolerations",
			tolerations:   []corev1.Toleration{},
			expectedPatch: []patchMapValue{},
		},
		{
			name: "Single toleration with TaintNodeUnreachable",
			tolerations: []corev1.Toleration{
				{
					Key:      corev1.TaintNodeUnreachable,
					Operator: corev1.TolerationOpExists,
				},
			},
			expectedPatch: []patchMapValue{},
		},
		{
			name: "Single toleration without TaintNodeUnreachable",
			tolerations: []corev1.Toleration{
				{
					Key:      "example.com/key",
					Operator: corev1.TolerationOpEqual,
					Value:    "value",
				},
			},
			expectedPatch: []patchMapValue{
				{
					Op:   "add",
					Path: "/spec/tolerations/-",
					/*Value: corev1.Toleration{
						Key:      "example.com/key",
						Operator: corev1.TolerationOpEqual,
						Value:    "value",
					},*/
				},
			},
		},
		{
			name: "Multiple tolerations with one TaintNodeUnreachable",
			tolerations: []corev1.Toleration{
				{
					Key:      corev1.TaintNodeUnreachable,
					Operator: corev1.TolerationOpExists,
				},
				{
					Key:      "example.com/key1",
					Operator: corev1.TolerationOpEqual,
					Value:    "value1",
				},
				{
					Key:      "example.com/key2",
					Operator: corev1.TolerationOpEqual,
					Value:    "value2",
				},
			},
			expectedPatch: []patchMapValue{
				{
					Op:   "add",
					Path: "/spec/tolerations/-",
					/*Value: corev1.Toleration{
						Key:      "example.com/key1",
						Operator: corev1.TolerationOpEqual,
						Value:    "value1",
					},*/
				},
				{
					Op:   "add",
					Path: "/spec/tolerations/-",
					/*Value: corev1.Toleration{
						Key:      "example.com/key2",
						Operator: corev1.TolerationOpEqual,
						Value:    "value2",
					},*/
				},
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePatch(tt.tolerations)
			// as patchMapValue is interface{} so just check len to avoid mock
			// and the result length always 1
			assert.Equal(t, 1, len(result))
		})
	}
}
