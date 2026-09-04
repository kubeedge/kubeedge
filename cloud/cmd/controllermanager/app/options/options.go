package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

// ControllerManagerOptions holds the configuration for the controller manager.
type ControllerManagerOptions struct {
	UseServerSideApply     bool
	HealthProbeBindAddress string
	FeatureGates           []string
}

// NewControllerManagerOptions creates a new ControllerManagerOptions with
// default configuration values.
func NewControllerManagerOptions() *ControllerManagerOptions {
	return &ControllerManagerOptions{}
}

func (o *ControllerManagerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("ControllerManager")
	fs.BoolVar(&o.UseServerSideApply, "use-server-side-apply", o.UseServerSideApply, "If use server-side apply when updating templates.")
	fs.StringVar(&o.HealthProbeBindAddress, "health-probe-bind-address", ":9001", "The TCP address that the controller should bind to for serving health probes.")
	fs.StringArrayVar(&o.FeatureGates, "feature-gates", o.FeatureGates, "Used to enable some features.")
	return
}
