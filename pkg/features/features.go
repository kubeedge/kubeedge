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
	// RequireAuthorization supports application access authorization from edge sides.
	// It will determine whether app can acquire meta data from kube-apiserver (if node is online) or from local host db (when node is offline)
	// without authorization. When node is offline and this value set to true, meta data can't be retrieved from meta server
	// because authorization offline is not achieved as of now.
	// alpha: v1.12
	// owner: @vincentgoat
	RequireAuthorization featuregate.Feature = "requireAuthorization"
)

// defaultFeatureGates consists of all known Kubeedge-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Kubeedge binaries.
var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	RequireAuthorization: {Default: false, PreRelease: featuregate.Alpha},
}
