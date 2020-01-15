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
package flags

import (
	"fmt"
	"os"
	"strconv"

	flag "github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/pkg/apis/cloudcore/v1alpha1"
)

type minConfigValue int
type defaultConfigValue int

const (
	MinConfigFalse minConfigValue = 0
	MinConfigTrue  minConfigValue = 1

	DefaultConfigFalse defaultConfigValue = 0
	DefaultConfigTrue  defaultConfigValue = 1
)

func (m *minConfigValue) IsBoolFlag() bool {
	return true
}
func (d *defaultConfigValue) IsBoolFlag() bool {
	return true
}

func (m *minConfigValue) Get() interface{} {
	return minConfigValue(*m)
}

func (d *defaultConfigValue) Get() interface{} {
	return defaultConfigValue(*d)
}

func (m *minConfigValue) Set(s string) error {
	boolVal, err := strconv.ParseBool(s)
	if boolVal {
		*m = MinConfigTrue
	} else {
		*m = MinConfigFalse
	}
	return err
}

func (d *defaultConfigValue) Set(s string) error {
	boolVal, err := strconv.ParseBool(s)
	if boolVal {
		*d = DefaultConfigTrue
	} else {
		*d = DefaultConfigFalse
	}
	return err
}

func (m *minConfigValue) String() string {
	return fmt.Sprintf("%v", bool(*m == MinConfigTrue))
}

func (d *defaultConfigValue) String() string {
	return fmt.Sprintf("%v", bool(*d == DefaultConfigTrue))
}

// The type of the flag as required by the pflag.Value interface
func (m *minConfigValue) Type() string {
	return "minconfig"
}

// The type of the flag as required by the pflag.Value interface
func (d *defaultConfigValue) Type() string {
	return "defaultconfig"
}

func MinConfigVar(p *minConfigValue, name string, value minConfigValue, usage string) {
	*p = value
	flag.Var(p, name, usage)
	// "--minconfig" will be treated as "--minconfig=true"
	flag.Lookup(name).NoOptDefVal = "true"
}

func DefaultConfigVar(p *defaultConfigValue, name string, value defaultConfigValue, usage string) {
	*p = value
	flag.Var(p, name, usage)
	// "--defaultconfig" will be treated as "--defaultconfig=true"
	flag.Lookup(name).NoOptDefVal = "true"
}

func minConfig(name string, value minConfigValue, usage string) *minConfigValue {
	p := new(minConfigValue)
	MinConfigVar(p, name, value, usage)
	return p
}

func defaultConfig(name string, value defaultConfigValue, usage string) *defaultConfigValue {
	p := new(defaultConfigValue)
	DefaultConfigVar(p, name, value, usage)
	return p
}

const minConfigFlagName = "minconfig"
const defaultConfigFlagName = "defaultconfig"

var (
	minConfigFlag     = minConfig(minConfigFlagName, MinConfigFalse, "Print min configuration for reference, users can refer to it to create their own configuration files, it is suitable for beginners.")
	defaultConfigFlag = defaultConfig(defaultConfigFlagName, DefaultConfigFalse, "Print default configuration for reference, users can refer to it to create their own configuration files, it is suitable for advanced users.")
)

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *flag.FlagSet) {
	fs.AddFlag(flag.Lookup(minConfigFlagName))
	fs.AddFlag(flag.Lookup(defaultConfigFlagName))
}

// PrintMinConfigAndExitIfRequested will check if the -minconfig flag was passed
// and, if so, print the min config and exit.
func PrintMinConfigAndExitIfRequested() {
	if *minConfigFlag == MinConfigTrue {
		config := v1alpha1.NewMinCloudCoreConfig()
		data, err := yaml.Marshal(config)
		if err != nil {
			fmt.Printf("Marshal cloudcore min config to yaml error %v\n", err)
			os.Exit(1)
		}
		fmt.Println("# With --minconfig , you can easily used this configurations as reference.")
		fmt.Println("# It's useful to users who are new to KubeEdge, and you can modify/create your own configs accordingly. ")
		fmt.Println("# This configuration is suitable for beginners.")
		fmt.Printf("\n%v\n\n", string(data))
		os.Exit(0)
	}
}

// PrintDefaultConfigAndExitIfRequested will check if the -defaultconfig flag was passed
// and, if so, print the default config and exit.
func PrintDefaultConfigAndExitIfRequested() {
	if *defaultConfigFlag == DefaultConfigTrue {
		config := v1alpha1.NewDefaultCloudCoreConfig()
		data, err := yaml.Marshal(config)
		if err != nil {
			fmt.Printf("Marshal cloudcore default config to yaml error %v\n", err)
			os.Exit(1)
		}
		fmt.Println("# With --defaultconfig flag, users can easily get a default full config file as reference, with all fields (and field descriptions) included and default values set. ")
		fmt.Println("# Users can modify/create their own configs accordingly as reference. ")
		fmt.Println("# Because it is a full configuration, it is more suitable for advanced users.")
		fmt.Printf("\n%v\n\n", string(data))
		os.Exit(0)
	}
}
