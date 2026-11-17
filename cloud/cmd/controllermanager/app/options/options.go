package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

type ControllerManagerOptions struct {
	UseServerSideApply     bool
	HealthProbeBindAddress string
	WebhookPort            int
	WebhookCertDir         string
	FeatureGates           []string
}

func NewControllerManagerOptions() *ControllerManagerOptions {
	return &ControllerManagerOptions{}
}

func (o *ControllerManagerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("ControllerManager")
	fs.BoolVar(&o.UseServerSideApply, "use-server-side-apply", o.UseServerSideApply, "If use server-side apply when updating templates.")
	fs.StringVar(&o.HealthProbeBindAddress, "health-probe-bind-address", ":9001", "The TCP address that the controller should bind to for serving health probes.")
	fs.IntVar(&o.WebhookPort, "webhook-port", -1, "The port that the webhook server binds to. Set to -1 to disable the webhook server.")
	fs.StringVar(&o.WebhookCertDir, "webhook-cert-dir", "", "The directory containing the webhook TLS certificate and key.")
	fs.StringArrayVar(&o.FeatureGates, "feature-gates", o.FeatureGates, "Used to enable some features.")
	return
}
