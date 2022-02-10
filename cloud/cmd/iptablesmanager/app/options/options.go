/*
Copyright 2021 The KubeEdge Authors.

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

package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

// IptablesManagerOptions config
type IptablesManagerOptions struct {
	KubeConfig  string
	ForwardPort int
}

// NewIptablesManagerOptions returns options object
func NewIptablesManagerOptions() *IptablesManagerOptions {
	return &IptablesManagerOptions{
		KubeConfig:  "",
		ForwardPort: 10003,
	}
}

// Flags return flag sets
func (o *IptablesManagerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("IptablesManager")
	fs.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig, "The KubeConfig path. Flags override values in this file.")
	fs.IntVar(&o.ForwardPort, "forwardport", o.ForwardPort, "The forward port, default is the stream port, 10003.")
	return
}
