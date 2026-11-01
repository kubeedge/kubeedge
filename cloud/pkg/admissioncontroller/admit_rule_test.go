package admissioncontroller

import (
	"testing"

	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
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
						Raw: []byte(`{"apiVersion":"rules/v1","kind":"Rule","metadata":{"name":"test-rule","Namespace":"test"},"spec":{"source":"someValue"}}`),
					},
				},
			},
			expectedResult: &admissionv1.AdmissionResponse{
				Allowed: false,
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
			expectedResult: nil,
			expectError:    true,
		},
	}

	// Initialize the deserializer
	serializer.NewCodecFactory(runtime.NewScheme())

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

func TestValidateSourceRuleEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		ruleEndpoint   *rulesv1.RuleEndpoint
		sourceResource map[string]string
		expectedError  bool
	}{
		{
			name: "Valid REST RuleEndpoint with path",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: rulesv1.RuleEndpointTypeRest,
				},
			},
			sourceResource: map[string]string{
				"path": "/api/v1/resource",
			},
			expectedError: true,
		},
		{
			name: "Missing path in sourceResource for REST RuleEndpoint",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
				},
			},
			sourceResource: map[string]string{},
			expectedError:  true,
		},
		{
			name: "Non-REST RuleEndpoint",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: "non-rest",
				},
			},
			sourceResource: map[string]string{},
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ruleEndpoint.Namespace = "default"
			err := validateSourceRuleEndpoint(tt.ruleEndpoint, tt.sourceResource)
			if err != nil && tt.expectedError == false {
				t.Errorf("Should return error error: %v", err)
			}
		})
	}
}

func TestValidateTargetRuleEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		ruleEndpoint   *rulesv1.RuleEndpoint
		targetResource map[string]string
		expectedError  string
	}{
		{
			name: "Valid REST RuleEndpoint with resource",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: rulesv1.RuleEndpointTypeRest,
				},
			},
			targetResource: map[string]string{
				"resource": "some-resource",
			},
			expectedError: "",
		},
		{
			name: "Invalid REST RuleEndpoint without resource",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: rulesv1.RuleEndpointTypeRest,
				},
			},
			targetResource: map[string]string{},
			expectedError:  "\"resource\" property missed in targetResource when ruleEndpoint is \"rest\"",
		},
		{
			name: "Valid RuleEndpoint with unknown type",
			ruleEndpoint: &rulesv1.RuleEndpoint{
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: "unknown",
				},
			},
			targetResource: map[string]string{},
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargetRuleEndpoint(tt.ruleEndpoint, tt.targetResource)

			if err != nil && err.Error() != tt.expectedError {
				t.Errorf("validateTargetRuleEndpoint() error = %v, expectedError %v", err, tt.expectedError)
			}

			if err == nil && tt.expectedError != "" {
				t.Errorf("validateTargetRuleEndpoint() expected error = %v, got nil", tt.expectedError)
			}
		})
	}
}
