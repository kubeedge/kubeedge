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

package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/cmd/server/app/options"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"

	"github.com/kubeedge/kubeedge/edgesite/cmd/edgesite-server/app"
)

func main() {
	// flag.CommandLine.Parse(os.Args[1:])
	proxy := &app.Proxy{}
	o := options.NewProxyRunOptions()
	command := app.NewProxyCommand(proxy, o)
	flags := command.Flags()
	flags.AddFlagSet(o.Flags())
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(local)
	err := local.Set("v", "4")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting klog flags: %v", err)
	}
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = util.Normalize(fl.Name)
		flags.AddGoFlag(fl)
	})
	if err := command.Execute(); err != nil {
		klog.Errorf("error: %v\n", err)
		klog.Flush()
		os.Exit(1)
	}
}
