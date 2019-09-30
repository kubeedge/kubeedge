package main

import (
	"strings"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/examples/chat/config"
)

func init() {
	klog.InitFlags(nil)
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
