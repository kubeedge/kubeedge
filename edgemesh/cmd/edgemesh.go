package main

import (
	"flag"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	// TODO need parse edgemesh config file before Register @kadisi
	//pkg.Register()
}
