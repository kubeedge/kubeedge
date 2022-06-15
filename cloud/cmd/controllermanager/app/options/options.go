package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

type ControllerManagerOptions struct {
	UseServerSideApply bool
}

func NewControllerManagerOptions() *ControllerManagerOptions {
	return &ControllerManagerOptions{}
}

func (o *ControllerManagerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("ControllerManager")
	fs.BoolVar(&o.UseServerSideApply, "use-server-side-apply", o.UseServerSideApply, "If use server-side apply when updating templates")
	return
}
