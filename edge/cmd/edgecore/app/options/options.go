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
	"fmt"
	"path"

	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
)

type EdgeCoreOptions struct {
	ConfigFile string
}

var edgeCoreOptions *EdgeCoreOptions
var edgeCoreConfig *v1alpha2.EdgeCoreConfig

func GetEdgeCoreOptions() *EdgeCoreOptions {
	return edgeCoreOptions
}

func GetEdgeCoreConfig() *v1alpha2.EdgeCoreConfig {
	return edgeCoreConfig
}

func NewEdgeCoreOptions() *EdgeCoreOptions {
	edgeCoreOptions = &EdgeCoreOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "edgecore.yaml"),
	}
	return edgeCoreOptions
}

func (o *EdgeCoreOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (o *EdgeCoreOptions) Validate() []error {
	var errs []error
	if !validation.FileIsExist(o.ConfigFile) {
		errs = append(errs, field.Required(field.NewPath("config"),
			fmt.Sprintf("config file %v not exist. For the configuration file format, please refer to --minconfig and --defaultconfig command", o.ConfigFile)))
	}
	return errs
}

func (o *EdgeCoreOptions) Config() (*v1alpha2.EdgeCoreConfig, error) {
	edgeCoreConfig = v1alpha2.NewDefaultEdgeCoreConfig()
	if err := edgeCoreConfig.Parse(o.ConfigFile); err != nil {
		return nil, err
	}

	return edgeCoreConfig, nil
}
