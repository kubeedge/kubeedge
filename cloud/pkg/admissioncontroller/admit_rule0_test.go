package main

import (
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	rulesv1 "your-package-path/rules/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

func TestAdmitRule(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		review         admissionv1.AdmissionReview
		expectedResult *admissionv1.AdmissionResponse
		expectError    bool
	}{
		{
			name: "Successful Create Operation",
			review: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{"apiVersion":"rules/v1","kind":"Rule","metadata":{"name":"test-rule"},"spec":{"someField":"someValue"}}`),
					},
				},
			},
			expectedResult: &admissionv1.AdmissionResponse{
				Allowed: true,
			},
			expectError: false,
		},
		{
			name: "Invalid JSON in Create Operation",
			review: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`invalid json`),
					},
				},
			},
			expectedResult: nil,
			expectError:    true,
		},
		{
			name: "Unsupported Operation",
			review: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
				},
			},
			expectedResult: &admissionv1.AdmissionResponse{
				Allowed: true,
			},
			expectError: false,
		},
	}

	// Initialize the deserializer
	codecs := serializer.NewCodecFactory(runtime.NewScheme())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			response := admitRule(tt.review)

			// Check for errors
			if tt.expectError {
				if response == nil || response.Allowed {
					t.Errorf("Expected an error, but got a successful response")
				}
			} else {
				if response == nil {
					t.Errorf("Expected a response, but got nil")
				} else if response.Allowed != tt.expectedResult.Allowed {
					t.Errorf("Expected Allowed=%v, but got Allowed=%v", tt.expectedResult.Allowed, response.Allowed)
				}
			}
		})
	}
}

// Helper function to convert an error to an AdmissionResponse
func toAdmissionResponse(err error) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}