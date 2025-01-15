package yourpackage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"yourmodule/rulesv1"
	"yourmodule/mocks"
)

func TestValidateRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := mocks.NewMockController(ctrl)

	tests := []struct {
		name          string
		rule          *rulesv1.Rule
		mockSetup     func(*mocks.MockController)
		expectedError error
	}{
		{
			name: "successful validation",
			rule: &rulesv1.Rule{
				Namespace: "ns1",
				Spec: rulesv1.RuleSpec{
					Source: "source1",
				},
			},
			mockSetup: func(mc *mocks.MockController) {
				mc.EXPECT().getRuleEndpoint("ns1", "source1").Return(&rulesv1.RuleEndpoint{}, nil)
			},
			expectedError: nil,
		},
		{
			name: "error getting source ruleEndpoint",
			rule: &rulesv1.Rule{
				Namespace: "ns1",
				Spec: rulesv1.RuleSpec{
					Source: "source1",
				},
			},
			mockSetup: func(mc *mocks.MockController) {
				mc.EXPECT().getRuleEndpoint("ns1", "source1").Return(nil, errors.New("some error"))
			},
			expectedError: fmt.Errorf("cant get source ruleEndpoint ns1/source1. Reason: some error"),
		},
		{
			name: "source ruleEndpoint not created",
			rule: &rulesv1.Rule{
				Namespace: "ns1",
				Spec: rulesv1.RuleSpec{
					Source: "source1",
				},
			},
			mockSetup: func(mc *mocks.MockController) {
				mc.EXPECT().getRuleEndpoint("ns1", "source1").Return(nil, nil)
			},
			expectedError: fmt.Errorf("source ruleEndpoint ns1/source1 has not been created"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup(mockController)

			// Replace the actual controller with the mock
			originalController := controller
			controller = mockController
			defer func() { controller = originalController }()

			err := validateRule(tt.rule)

			if tt.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedError.Error())
			}
		})
	}
}