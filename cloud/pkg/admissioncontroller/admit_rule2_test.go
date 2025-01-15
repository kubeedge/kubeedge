package rulesv1

import (
	"testing"
	"fmt"
)

func TestValidateSourceRuleEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		ruleEndpoint   *RuleEndpoint
		sourceResource map[string]string
		expectedError  error
	}{
		{
			name: "Valid REST RuleEndpoint with path",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
					RuleEndpointType: RuleEndpointTypeRest,
				},
			},
			sourceResource: map[string]string{
				"path": "/api/v1/resource",
			},
			expectedError: nil,
		},
		{
			name: "Missing path in sourceResource for REST RuleEndpoint",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
					RuleEndpointType: RuleEndpointTypeRest,
				},
			},
			sourceResource: map[string]string{},
			expectedError:  fmt.Errorf("\"path\" property missed in sourceResource when ruleEndpoint is \"rest\""),
		},
		{
			name: "Non-REST RuleEndpoint",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
					RuleEndpointType: "non-rest",
				},
			},
			sourceResource: map[string]string{},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSourceRuleEndpoint(tt.ruleEndpoint, tt.sourceResource)
			if err != nil && tt.expectedError == nil {
				t.Errorf("Unexpected error: %v", err)
			} else if err == nil && tt.expectedError != nil {
				t.Errorf("Expected error: %v, got nil", tt.expectedError)
			} else if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("Expected error: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}