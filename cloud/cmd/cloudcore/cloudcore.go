package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/cmd/cloudcore/app"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	command := app.NewCloudCoreCommand()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
