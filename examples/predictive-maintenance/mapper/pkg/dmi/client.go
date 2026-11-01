/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dmi

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"

	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/config"
	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/driver"
)

// Client wraps the DMI gRPC connection.
type Client struct {
	cfg    config.DMIConfig
	conn   *grpc.ClientConn
	client pb.DeviceManagerServiceClient
}

// NewClient connects to EdgeCore DMI socket.
func NewClient(cfg config.DMIConfig) (*Client, error) {
	conn, err := grpc.NewClient(
		"unix://"+cfg.SocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", cfg.SocketPath)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dmi connect failed: %w", err)
	}

	klog.Infof("DMI connected: %s", cfg.SocketPath)
	return &Client{cfg: cfg, conn: conn, client: pb.NewDeviceManagerServiceClient(conn)}, nil
}

// Register announces this mapper to EdgeCore.
func (c *Client) Register(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req := &pb.MapperRegisterRequest{
		WithData: true,
		Mapper: &pb.MapperInfo{
			Name:     c.cfg.MapperName,
			Protocol: c.cfg.Protocol,
			Address:  []byte(c.cfg.SocketPath),
		},
	}
	resp, err := c.client.MapperRegister(ctx, req)
	if err != nil {
		return fmt.Errorf("register failed: %w", err)
	}
	klog.Infof("registered: %d devices, %d models", len(resp.GetDeviceList()), len(resp.GetModelList()))
	return nil
}

// ReportStatus sends reading to EdgeCore.
func (c *Client) ReportStatus(ctx context.Context, r driver.SensorReading) error {
	vib := fmt.Sprintf("%.4f", r.Vibration)
	tmp := fmt.Sprintf("%.2f", r.Temperature)
	ano := "false"
	if r.IsAnomaly {
		ano = "true"
	}

	req := &pb.ReportDeviceStatusRequest{
		DeviceName:      c.cfg.DeviceName,
		DeviceNamespace: c.cfg.DeviceNamespace,
		ReportedDevice: &pb.DeviceStatus{
			Twins: []*pb.Twin{
				{PropertyName: "vibration", Reported: &pb.TwinProperty{Value: vib}, ObservedDesired: &pb.TwinProperty{Value: vib}},
				{PropertyName: "temperature", Reported: &pb.TwinProperty{Value: tmp}, ObservedDesired: &pb.TwinProperty{Value: tmp}},
				{PropertyName: "anomaly-detected", Reported: &pb.TwinProperty{Value: ano}, ObservedDesired: &pb.TwinProperty{Value: ano}},
			},
		},
	}

	rCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := c.client.ReportDeviceStatus(rCtx, req); err != nil {
		return fmt.Errorf("report failed: %w", err)
	}
	klog.V(3).Infof("reported vib=%s temp=%s anomaly=%s", vib, tmp, ano)
	return nil
}

// Close releases gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
