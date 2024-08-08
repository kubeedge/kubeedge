/*
Copyright 2020 The KubeEdge Authors.

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

package flag

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type ConfigValue int

const (
	ConfigFalse ConfigValue = 0
	ConfigTrue  ConfigValue = 1
)

func (m *ConfigValue) IsBoolFlag() bool {
	return true
}

func (m *ConfigValue) Get() interface{} {
	return ConfigValue(*m)
}

func (m *ConfigValue) Set(s string) error {
	boolVal, err := strconv.ParseBool(s)
	if boolVal {
		*m = ConfigTrue
	} else {
		*m = ConfigFalse
	}
	return err
}

func (m *ConfigValue) String() string {
	return fmt.Sprintf("%v", bool(*m == ConfigTrue))
}

// The type of the flag as required by the pflag.Value interface
func (m *ConfigValue) Type() string {
	return "config"
}

func ConfigVar(p *ConfigValue, name string, value ConfigValue, usage string) {
	*p = value
	pflag.Var(p, name, usage)
	pflag.Lookup(name).NoOptDefVal = "true"
}

func Config(name string, value ConfigValue, usage string) *ConfigValue {
	p := new(ConfigValue)
	ConfigVar(p, name, value, usage)
	return p
}

const minConfigFlagName = "minconfig"
const defaultConfigFlagName = "defaultconfig"

var (
	minConfigFlag     = Config(minConfigFlagName, ConfigFalse, "Print min configuration for reference, users can refer to it to create their own configuration files, it is suitable for beginners.")
	defaultConfigFlag = Config(defaultConfigFlagName, ConfigFalse, "Print default configuration for reference, users can refer to it to create their own configuration files, it is suitable for advanced users.")
)

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *pflag.FlagSet) {
	fs.AddFlag(pflag.Lookup(minConfigFlagName))
	fs.AddFlag(pflag.Lookup(defaultConfigFlagName))
}

// PrintMinConfigAndExitIfRequested will check if the -minconfig flag was passed
// and, if so, print the min config and exit.
func PrintMinConfigAndExitIfRequested(config interface{}) {
	if *minConfigFlag == ConfigTrue {
		data, err := yaml.Marshal(config)
		if err != nil {
			fmt.Printf("Marshal min config to yaml error %v\n", err)
			os.Exit(1)
		}
		fmt.Println("# With --minconfig , you can easily used this configurations as reference.")
		fmt.Println("# It's useful to users who are new to KubeEdge, and you can modify/create your own configs accordingly. ")
		fmt.Println("# This configuration is suitable for beginners.")
		fmt.Printf("\n%v\n\n", string(data))
		os.Exit(0)
	}
}

// PrintDefaultConfigAndExitIfRequested will check if the --defaultconfig flag was passed
// and, if so, print the default config and exit.
func PrintDefaultConfigAndExitIfRequested(config interface{}) {
	if *defaultConfigFlag == ConfigTrue {
		data, err := yaml.Marshal(config)
		if err != nil {
			fmt.Printf("Marshal default config to yaml error %v\n", err)
			os.Exit(1)
		}
		fmt.Println("# With --defaultconfig flag, users can easily get a default full config file as reference, with all fields (and field descriptions) included and default values set. ")
		fmt.Println("# Users can modify/create their own configs accordingly as reference. ")
		fmt.Println("# Because it is a full configuration, it is more suitable for advanced users.")
		fmt.Printf("\n%v\n\n", string(data))
		os.Exit(0)
	}
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
