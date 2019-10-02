/*
Copyright 2019 The KubeEdge Authors.

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
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/cloudcore/config"
)

// TODO set cloudcore config
type CloudCoreOptions struct {
	ConfigFile string
}

func NewCloudCoreOptions() *CloudCoreOptions {
	return &CloudCoreOptions{}
}

func (o *CloudCoreOptions) Flags() (fss cliflag.NamedFlagSets) {
	// TODO set CloudCoreOptions field
	//fs := fss.FlagSet("general")
	return
}

func (o *CloudCoreOptions) Validate() []error {
	var errs []error
	/*
		if len(o.ConfigFile) == 0 {
		errs = append(errs, field.Required(field.NewPath("ConfigFile"), ""))
		}
	*/

	return errs
}

func (o *CloudCoreOptions) Config() (*config.CloudCoreConfig, error) {
	cfg := config.NewDefaultCloudCoreConfig()
	if err := cfg.Parse(o.ConfigFile); err != nil {
		klog.Errorf("Parse config %s error %v", o.ConfigFile, err)
		return nil, err
	}
	return cfg, nil
}
