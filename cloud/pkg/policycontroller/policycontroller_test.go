package policycontroller

import (
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/pkg/features"
)

func TestRegister(t *testing.T) {
	if err := features.DefaultMutableFeatureGate.SetFromMap(map[string]bool{string(features.RequireAuthorization): true}); err != nil {
		t.Errorf("Failed to set feature gate: %v", err)
	}
	tests := []struct {
		name       string
		controller *policyController
		ctrEnable  bool
		ctrName    string
	}{
		{
			name:       "Register Policy controller",
			controller: &policyController{},
			ctrName:    "policycontroller",
			ctrEnable:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc = &policyController{}
			if !reflect.DeepEqual(tt.ctrName, pc.Name()) {
				t.Errorf("TestCase %q got %v, want %v", tt.name, pc.Name(), tt.ctrName)
			}
			if !reflect.DeepEqual(tt.ctrEnable, pc.Enable()) {
				t.Errorf("TestCase %q got %v, want %v", tt.name, pc.Enable(), tt.ctrEnable)
			}
		})
	}
}
