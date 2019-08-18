package main

import (
	"flag"
	"github.com/kubeedge/kubeedge/edgemesh/pkg"
	"github.com/spf13/pflag"
	"k8s.io/klog"

	_ "github.com/kubeedge/kubeedge/edgemesh/pkg/panel"
	"github.com/kubeedge/kubeedge/edgemesh/pkg/server"
)

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	pkg.Register()

	//Start server
	server.StartTCP()
}
