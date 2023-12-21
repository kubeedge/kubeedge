package main

import (
	"errors"
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/device"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/config"
	"github.com/kubeedge/mapper-framework/pkg/grpcserver"
	"github.com/kubeedge/mapper-framework/pkg/httpserver"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

func main() {
	var err error
	var c *config.Config

	klog.InitFlags(nil)
	defer klog.Flush()

	if c, err = config.Parse(); err != nil {
		klog.Fatal(err)
		os.Exit(1)
	}
	klog.Infof("config: %+v", c)

	// start grpc server
	grpcServer := grpcserver.NewServer(
		grpcserver.Config{
			SockPath: c.GrpcServer.SocketPath,
			Protocol: common.ProtocolCustomized,
		},
		device.NewDevPanel(),
	)

	panel := device.NewDevPanel()
	err = panel.DevInit(c)
	if err != nil && !errors.Is(err, parse.ErrEmptyData) {
		klog.Fatal(err)
	}
	klog.Infoln("devInit finished")

	go panel.DevStart()

	httpServer := httpserver.NewRestServer(panel)
	go httpServer.StartServer()

	defer grpcServer.Stop()
	if err = grpcServer.Start(); err != nil {
		klog.Fatal(err)
	}
}
