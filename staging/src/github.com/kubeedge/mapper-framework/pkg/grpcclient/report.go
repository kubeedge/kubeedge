package grpcclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/config"
)

// ReportDeviceStatus report device status to edgecore
func ReportDeviceStatus(request *dmiapi.ReportDeviceStatusRequest) error {
	cfg := config.Cfg()

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
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	// init Greeter client
	c := dmiapi.NewDeviceManagerServiceClient(conn)

	// init context，set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.ReportDeviceStatus(ctx, request)
	return err
}

func ReportDeviceStates(request *dmiapi.ReportDeviceStatesRequest) error {
	cfg := config.Cfg()

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
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	// init Greeter client
	c := dmiapi.NewDeviceManagerServiceClient(conn)

	// init context，set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.ReportDeviceStates(ctx, request)
	return err
}
