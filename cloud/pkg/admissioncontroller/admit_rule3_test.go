package rulesv1

import (
	"testing"
)

func TestValidateTargetRuleEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		ruleEndpoint   *RuleEndpoint
		targetResource map[string]string
		expectedError  string
	}{
		{
			name: "Valid REST RuleEndpoint with resource",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
					RuleEndpointType: RuleEndpointTypeRest,
				},
			},
			targetResource: map[string]string{
				"resource": "some-resource",
			},
			expectedError: "",
		},
		{
			name: "Invalid REST RuleEndpoint without resource",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
					RuleEndpointType: RuleEndpointTypeRest,
				},
			},
			targetResource: map[string]string{},
			expectedError:  "\"resource\" property missed in targetResource when ruleEndpoint is \"rest\"",
		},
		{
			name: "Valid RuleEndpoint with unknown type",
			ruleEndpoint: &RuleEndpoint{
				Spec: RuleEndpointSpec{
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