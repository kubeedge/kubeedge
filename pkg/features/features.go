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
	// ModuleRestart supports automatic restarting for modules.
	// If a module exits when running because of uncaught or external errors, BeeHive will try to keep the module running by restarting it.
	// If moduleRestart enabled, modules will be kept running forever. The interval between starting a module increases whenever it exits,
	// with maximum of 30s.
	// alpha: v1.17
	// owner: @micplus
	ModuleRestart featuregate.Feature = "moduleRestart"
	// SupportCSINode enables CSI (Container Storage Interface) support in edgecore.
	// When set to true (default), CSI drivers are used for storage operations.
	// Disabling this (set to false) will turn off all CSI-related functionality.
	// This is useful in environments that do not require CSI or for troubleshooting.
	// alpha: v1.21
	SupportCSINode featuregate.Feature = "supportCSINode"
)

// defaultFeatureGates consists of all known Kubeedge-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Kubeedge binaries.
var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	RequireAuthorization: {Default: false, PreRelease: featuregate.Alpha},
	ModuleRestart:        {Default: false, PreRelease: featuregate.Alpha},
	SupportCSINode:       {Default: true, PreRelease: featuregate.Alpha},
}
