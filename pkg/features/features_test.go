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

func TestRequireAuthorizationRegistered(t *testing.T) {
	spec, ok := defaultFeatureGates[RequireAuthorization]
	if !ok {
		t.Fatal("RequireAuthorization not in defaultFeatureGates")
	}
	if spec.Default != false {
		t.Errorf("RequireAuthorization default: got %v, want false", spec.Default)
	}
	if spec.PreRelease != featuregate.Alpha {
		t.Errorf("RequireAuthorization prerelease: got %v, want Alpha", spec.PreRelease)
	}
}

func TestModuleRestartRegistered(t *testing.T) {
	spec, ok := defaultFeatureGates[ModuleRestart]
	if !ok {
		t.Fatal("ModuleRestart not in defaultFeatureGates")
	}
	if spec.Default != false {
		t.Errorf("ModuleRestart default: got %v, want false", spec.Default)
	}
	if spec.PreRelease != featuregate.Alpha {
		t.Errorf("ModuleRestart prerelease: got %v, want Alpha", spec.PreRelease)
	}
}

func TestDisableNodeTaskV1alpha2Registered(t *testing.T) {
	spec, ok := defaultFeatureGates[DisableNodeTaskV1alpha2]
	if !ok {
		t.Fatal("DisableNodeTaskV1alpha2 not in defaultFeatureGates")
	}
	if spec.Default != false {
		t.Errorf("DisableNodeTaskV1alpha2 default: got %v, want false", spec.Default)
	}
	if spec.PreRelease != featuregate.Alpha {
		t.Errorf("DisableNodeTaskV1alpha2 prerelease: got %v, want Alpha", spec.PreRelease)
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
