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

const (
	MinConfigFalse minConfigValue = 0
	MinConfigTrue  minConfigValue = 1
	// TODO support json output (default yaml ) @kadisi
)

func (m *minConfigValue) IsBoolFlag() bool {
	return true
}

func (m *minConfigValue) Get() interface{} {
	return minConfigValue(*m)
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

func (m *minConfigValue) String() string {
	return fmt.Sprintf("%v", bool(*m == MinConfigTrue))
}

// The type of the flag as required by the pflag.Value interface
func (m *minConfigValue) Type() string {
	return "minconfig"
}

func MinConfigVar(p *minConfigValue, name string, value minConfigValue, usage string) {
	*p = value
	flag.Var(p, name, usage)
	// "--minconfig" will be treated as "--minconfig=true"
	flag.Lookup(name).NoOptDefVal = "true"
}

func minConfig(name string, value minConfigValue, usage string) *minConfigValue {
	p := new(minConfigValue)
	MinConfigVar(p, name, value, usage)
	return p
}

const minConfigFlagName = "minconfig"

var (
	minConfigFlag = minConfig(minConfigFlagName, MinConfigFalse, "Print min configuration for reference, users can refer to it to create their own configuration files")
)

// AddFlags registers this package's flags on arbitrary FlagSets, such that they point to the
// same value as the global flags.
func AddFlags(fs *flag.FlagSet) {
	fs.AddFlag(flag.Lookup(minConfigFlagName))
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
		fmt.Printf("\n%v\n\n", string(data))
		fmt.Println(`# With --minconfig , you can easily used this configurations as reference. 
# It's useful to users who are new to KubeEdge, and you can modify/create your own configs accordingly. 
# This configuration is suitable for beginners.`)
		os.Exit(0)
	}
}
