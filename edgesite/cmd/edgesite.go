package main

import (
	"flag"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edgesite/cmd/app"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	command := app.NewEdgeSiteCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
