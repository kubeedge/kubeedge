/*
Copyright 2018 The Kubernetes Authors.

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

package globalflag

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
)

// AddGlobalFlags explicitly registers flags that libraries (klog, verflag, etc.) register
// against the global flagsets from "flag" and "k8s.io/klog".
// We do this in order to prevent unwanted flags from leaking into the component's flagset.
func AddGlobalFlags(fs *pflag.FlagSet, name string) {
	addKlogFlags(fs)
	logs.AddFlags(fs)

	fs.BoolP("help", "h", false, fmt.Sprintf("help for %s", name))
}

// addKlogFlags adds flags from k8s.io/klog
func addKlogFlags(fs *pflag.FlagSet) {
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = normalize(fl.Name)
		fs.AddGoFlag(fl)
	})
}

// normalize replaces underscores with hyphens
// we should always use hyphens instead of underscores when registering component flags
func normalize(s string) string {
	return strings.Replace(s, "_", "-", -1)
}

// Register adds a flag to local that targets the Value associated with the Flag named globalName in flag.CommandLine.
func Register(local *pflag.FlagSet, globalName string) {
	if f := flag.CommandLine.Lookup(globalName); f != nil {
		pflagFlag := pflag.PFlagFromGoFlag(f)
		pflagFlag.Name = normalize(pflagFlag.Name)
		local.AddFlag(pflagFlag)
	} else {
		panic(fmt.Sprintf("failed to find flag in global flagset (flag): %s", globalName))
	}
}
