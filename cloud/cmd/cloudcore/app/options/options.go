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
	"path"

	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/cloudcore/apis/config"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type CloudCoreOptions struct {
	ConfigFile string
}

func NewDefaultCloudCoreOptions() *CloudCoreOptions {
	return &CloudCoreOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "cloudcore.yaml"),
	}
}

func (c *CloudCoreOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&c.ConfigFile, "config", c.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (c *CloudCoreOptions) Validate() []error {
	var errs []error
	if len(c.ConfigFile) == 0 {
		errs = append(errs, field.Required(field.NewPath("ConfigFile"), ""))
	}

	return errs
}

func (c *CloudCoreOptions) Config() (*config.CloudCoreConfig, error) {
	cfg := config.NewDefaultCloudCoreConfig()
	if err := cfg.Parse(c.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
