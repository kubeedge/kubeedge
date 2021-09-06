package features

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

var (
	// DefaultMutableFeatureGate is a mutable version of DefaultFeatureGate.
	DefaultMutableFeatureGate featuregate.MutableFeatureGate = featuregate.NewFeatureGate()

	// DefaultFeatureGate is a shared global FeatureGate.
	// Top-level commands/options setup that needs to modify this feature gate should use DefaultMutableFeatureGate.
	DefaultFeatureGate featuregate.FeatureGate = DefaultMutableFeatureGate
)

func init() {
	runtime.Must(DefaultMutableFeatureGate.Add(defaultFeatureGates))
}

const (
	// ProcessStatusSync supports synchronization between process message channel statuses
	ProcessStatusSync featuregate.Feature = "ProcessStatusSync"
)

// defaultFeatureGates consists of all known Kubeedge-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Kubeedge binaries.
var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	ProcessStatusSync: {Default: false, PreRelease: featuregate.Alpha},
}
