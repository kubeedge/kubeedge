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

	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

type EdgeCoreOptions struct {
	ConfigFile string
}

func NewEdgeCoreOptions() *EdgeCoreOptions {
	return &EdgeCoreOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "edgecore.yaml"),
	}
}

func (o *EdgeCoreOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("general")
	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (o *EdgeCoreOptions) Validate() []error {
	var errs []error
	if len(o.ConfigFile) == 0 {
		errs = append(errs, field.Required(field.NewPath("ConfigFile"), ""))
	}
	return errs
}

func (o *EdgeCoreOptions) Config() (*config.EdgeCoreConfig, error) {
	cfg := config.NewDefaultEdgeCoreConfig()
	if err := cfg.Parse(o.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
