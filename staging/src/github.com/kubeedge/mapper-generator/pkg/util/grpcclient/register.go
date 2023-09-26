package grpcclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/config"
	dmiapi "github.com/kubeedge/mapper-generator/pkg/temp"
)

// RegisterMapper if withData is true, edgecore will send device and model list.
func RegisterMapper(cfg *config.Config, withData bool) ([]*dmiapi.Device, []*dmiapi.DeviceModel, error) {
	// connect grpc server
	conn, err := grpc.Dial(cfg.Common.EdgeCoreSock,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(
			func(ctx context.Context, s string) (net.Conn, error) {
				unixAddress, err := net.ResolveUnixAddr("unix", cfg.Common.EdgeCoreSock)
				if err != nil {
					return nil, err
				}
				return net.DialUnix("unix", nil, unixAddress)
			},
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	// init Greeter client
	c := dmiapi.NewDeviceManagerServiceClient(conn)

	// init ctx，set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := c.MapperRegister(ctx, &dmiapi.MapperRegisterRequest{
		WithData: withData,
		Mapper: &dmiapi.MapperInfo{
			Name:       cfg.Common.Name,
			Version:    cfg.Common.Version,
			ApiVersion: cfg.Common.APIVersion,
			Protocol:   cfg.Common.Protocol,
			Address:    []byte(cfg.GrpcServer.SocketPath),
			State:      common.DEVSTOK,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return resp.DeviceList, resp.ModelList, err
}
