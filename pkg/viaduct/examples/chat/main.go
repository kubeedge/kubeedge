package main

import (
	"flag"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/viaduct/examples/chat/config"
)

func init() {
	klog.InitFlags(nil)
	// Opt into the new klog behavior so that -stderrthreshold is honored even
	// when -logtostderr=true (the default).
	// Ref: kubernetes/klog#212, kubernetes/klog#432
	_ = flag.CommandLine.Set("legacy_stderr_threshold_behavior", "false")
}

func main() {
	cfg := config.InitConfig()

	var err error
	if strings.Compare(cfg.CmdType, "server") == 0 {
		err = StartServer(cfg)
	} else {
		err = StartClient(cfg)
	}
	if err != nil {
		klog.Errorf("start %s failed, error: %+v", cfg.CmdType, err)
	}
}
