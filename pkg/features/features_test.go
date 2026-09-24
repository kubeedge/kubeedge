package features

import (
	"testing"

	"k8s.io/component-base/featuregate"
)

func TestDefaultFeatureGateIsMutableFeatureGate(t *testing.T) {
	if DefaultFeatureGate != DefaultMutableFeatureGate {
		t.Error("DefaultFeatureGate must be same instance as DefaultMutableFeatureGate")
	}
}

func TestDefaultMutableFeatureGateNotNil(t *testing.T) {
	if DefaultMutableFeatureGate == nil {
		t.Fatal("DefaultMutableFeatureGate must not be nil")
	}
}

func TestDefaultFeatureGates(t *testing.T) {
	tests := []struct {
		feature    featuregate.Feature
		defaultVal bool
		preRelease interface{}
	}{
		{RequireAuthorization, false, featuregate.Alpha},
		{ModuleRestart, false, featuregate.Alpha},
		{DisableNodeTaskV1alpha2, false, featuregate.Alpha},
	}

	for _, tc := range tests {
		t.Run(string(tc.feature), func(t *testing.T) {
			spec, ok := defaultFeatureGates[tc.feature]
			if !ok {
				t.Fatalf("%s not in defaultFeatureGates", tc.feature)
			}
			if spec.Default != tc.defaultVal {
				t.Errorf("default: got %v, want %v", spec.Default, tc.defaultVal)
			}
			if spec.PreRelease != tc.preRelease {
				t.Errorf("prerelease: got %v, want %v", spec.PreRelease, tc.preRelease)
			}
		})
	}
}

func TestInitRegistersAllFeatures(t *testing.T) {
	features := []featuregate.Feature{
		RequireAuthorization,
		ModuleRestart,
		DisableNodeTaskV1alpha2,
	}
	for _, f := range features {
		if DefaultMutableFeatureGate.Enabled(f) {
			t.Errorf("feature %v should be disabled by default", f)
		}
		_ = DefaultFeatureGate.Enabled(f)
	}
}
