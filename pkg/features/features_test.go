/*
Copyright 2025 The Kubernetes Authors.

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

package features

import (
	"testing"

	"k8s.io/component-base/featuregate"
)

func TestDefaultFeatureGates(t *testing.T) {
	// First verify initialization
	if DefaultMutableFeatureGate == nil {
		t.Error("DefaultMutableFeatureGate should not be nil")
	}
	if DefaultFeatureGate == nil {
		t.Error("DefaultFeatureGate should not be nil")
	}
	if DefaultFeatureGate != DefaultMutableFeatureGate {
		t.Error("DefaultFeatureGate should be the same as DefaultMutableFeatureGate")
	}

	testCases := []struct {
		feature     featuregate.Feature
		expectedOff bool
	}{
		{
			feature:     RequireAuthorization,
			expectedOff: true,
		},
		{
			feature:     ModuleRestart,
			expectedOff: true,
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.feature), func(t *testing.T) {
			// Check that feature is disabled by default
			if DefaultFeatureGate.Enabled(tc.feature) {
				t.Errorf("Feature %s should be disabled by default", tc.feature)
			}

			// Verify the feature can be enabled
			featureGate := featuregate.NewFeatureGate()
			err := featureGate.Add(defaultFeatureGates)
			if err != nil {
				t.Fatalf("Failed to add feature gates: %v", err)
			}

			// Set the feature state using SetFromMap
			err = featureGate.SetFromMap(map[string]bool{string(tc.feature): true})
			if err != nil {
				t.Fatalf("Failed to set feature gate: %v", err)
			}

			if !featureGate.Enabled(tc.feature) {
				t.Errorf("Feature %s should be enabled after setting", tc.feature)
			}
		})
	}
}

func TestFeatureGateSpecifications(t *testing.T) {
	testCases := []struct {
		feature      featuregate.Feature
		expectedSpec featuregate.FeatureSpec
	}{
		{
			feature: RequireAuthorization,
			expectedSpec: featuregate.FeatureSpec{
				Default:    false,
				PreRelease: featuregate.Alpha,
			},
		},
		{
			feature: ModuleRestart,
			expectedSpec: featuregate.FeatureSpec{
				Default:    false,
				PreRelease: featuregate.Alpha,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.feature), func(t *testing.T) {
			spec, exists := defaultFeatureGates[tc.feature]
			if !exists {
				t.Errorf("Feature %s not found in defaultFeatureGates", tc.feature)
				return
			}

			if spec.Default != tc.expectedSpec.Default {
				t.Errorf("Unexpected default value for %s. Expected %v, got %v",
					tc.feature, tc.expectedSpec.Default, spec.Default)
			}

			if spec.PreRelease != tc.expectedSpec.PreRelease {
				t.Errorf("Unexpected pre-release status for")
			}
		})
	}
}
