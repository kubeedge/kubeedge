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

package app

import (
	"flag"
	"fmt"
	"time"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd"
)

// Run executes the keadm command
func Run() error {
	flagSet := flag.NewFlagSet("keadm", flag.ExitOnError)
	cmd := cmd.NewKubeedgeCommand()
	flags := cmd.Flags()
	klog.InitFlags(flagSet)
	err := flagSet.Set("v", "0")
	if err != nil {
		return fmt.Errorf("error setting klog flags: %v", err)
	}

	currentDateTime := time.Now().Format("20060102_1504")
	logFileName := fmt.Sprintf("./keadm_%s.log", currentDateTime)

	err = flagSet.Set("log_file", logFileName)
	if err != nil {
		return fmt.Errorf("error setting log file: %v", err)
	}
	err = flagSet.Set("logtostderr", "false") 
	if err != nil {
		return fmt.Errorf("error setting logtostderr: %v", err)
	}
	err = flagSet.Set("stderrthreshold", "FATAL") 
	if err != nil {
		return fmt.Errorf("error setting stderrthreshold: %v", err)
	}

	flagSet.Visit(func(fl *flag.Flag) {
		fl.Name = util.Normalize(fl.Name)
		flags.AddGoFlag(fl)
	})

	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddFlagSet(flags)


	defer klog.Flush()
	return cmd.Execute()
}
