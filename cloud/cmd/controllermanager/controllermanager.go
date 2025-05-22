package main

import (
	"os"

	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/cmd/controllermanager/app"
)

func main() {
	logs.InitLogs()
	klog.EnableContextualLogging(true)
	defer logs.FlushLogs()

	ctx := apiserver.SetupSignalContext()

	if err := app.NewControllerManagerCommand(ctx).Execute(); err != nil {
		os.Exit(1)
	}
}
