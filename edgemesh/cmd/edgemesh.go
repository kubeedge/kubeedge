package main

import (
	"flag"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	_ "github.com/kubeedge/kubeedge/edgemesh/pkg/panel"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	// TODO need parse edgemesh config file before Register @kadisi
	//pkg.Register()

	//Start server
	server.StartTCP()
}
