package main

import (
	"errors"

	"k8s.io/klog/v2"

	"github.com/kubeedge/modbus/device"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/config"
	"github.com/kubeedge/mapper-framework/pkg/grpcclient"
	"github.com/kubeedge/mapper-framework/pkg/grpcserver"
	"github.com/kubeedge/mapper-framework/pkg/httpserver"
)

func main() {
	var err error
	var c *config.Config

	klog.InitFlags(nil)
	defer klog.Flush()

	if c, err = config.Parse(); err != nil {
		klog.Fatal(err)
	}
	klog.Infof("config: %+v", c)

	klog.Infoln("Mapper will register to edgecore")
	deviceList, deviceModelList, err := grpcclient.RegisterMapper(true)
	if err != nil {
		klog.Fatal(err)
	}
	klog.Infoln("Mapper register finished")

	panel := device.NewDevPanel()
	err = panel.DevInit(deviceList, deviceModelList)
	if err != nil && !errors.Is(err, device.ErrEmptyData) {
		klog.Fatal(err)
	}
	klog.Infoln("devInit finished")
	go panel.DevStart()

	// start http server
	httpServer := httpserver.NewRestServer(panel, c.Common.HTTPPort)
	go httpServer.StartServer()

	// start grpc server
	grpcServer := grpcserver.NewServer(
		grpcserver.Config{
			SockPath: c.GrpcServer.SocketPath,
			Protocol: common.ProtocolCustomized,
		},
		panel,
	)
	defer grpcServer.Stop()
	if err = grpcServer.Start(); err != nil {
		klog.Fatal(err)
	}

}
