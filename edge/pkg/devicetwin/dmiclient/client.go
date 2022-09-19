/*
Copyright 2022 The KubeEdge Authors.

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

package dmiclient

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"

	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1alpha1"
)

type DMIClient struct {
	protocol   string
	socket     string
	Client     dmiapi.DeviceMapperServiceClient
	Ctx        context.Context
	Conn       *grpc.ClientConn
	CancelFunc context.CancelFunc
}

type DMIClients struct {
	mutex   sync.Mutex
	clients map[string]*DMIClient
}

var DMIClientsImp *DMIClients

func init() {
	DMIClientsImp = &DMIClients{
		mutex:   sync.Mutex{},
		clients: make(map[string]*DMIClient),
	}
}

func (dc *DMIClient) connect() error {
	dialer := func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(deviceconst.UnixNetworkType, addr)
	}

	conn, err := grpc.Dial(dc.socket, grpc.WithInsecure(), grpc.WithDialer(dialer))
	if err != nil {
		klog.Errorf("did not connect: %v\n", err)
		return err
	}

	c := dmiapi.NewMapperClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	dc.Client = c
	dc.Ctx = ctx
	dc.Conn = conn
	dc.CancelFunc = cancel

	return nil
}

func (dc *DMIClient) close() {
	dc.Conn.Close()
	dc.CancelFunc()
}

func createDeviceRequest(device *v1alpha2.Device) (*dmiapi.RegisterDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.RegisterDeviceRequest{
		Device: d,
	}, nil
}

func removeDeviceRequest(deviceName string) (*dmiapi.RemoveDeviceRequest, error) {
	return &dmiapi.RemoveDeviceRequest{
		DeviceName: deviceName,
	}, nil
}

func updateDeviceRequest(device *v1alpha2.Device) (*dmiapi.UpdateDeviceRequest, error) {
	d, err := dtcommon.ConvertDevice(device)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceRequest{
		Device: d,
	}, nil
}

func createDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.CreateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.CreateDeviceModelRequest{
		Model: m,
	}, nil
}

func updateDeviceModelRequest(model *v1alpha2.DeviceModel) (*dmiapi.UpdateDeviceModelRequest, error) {
	m, err := dtcommon.ConvertDeviceModel(model)
	if err != nil {
		return nil, err
	}

	return &dmiapi.UpdateDeviceModelRequest{
		Model: m,
	}, nil
}

func removeDeviceModelRequest(deviceModelName string) (*dmiapi.RemoveDeviceModelRequest, error) {
	return &dmiapi.RemoveDeviceModelRequest{
		ModelName: deviceModelName,
	}, nil
}

func (dcs *DMIClients) getDMIClientByProtocol(protocol string) (*DMIClient, error) {
	dcs.mutex.Lock()
	defer dcs.mutex.Unlock()
	dc, ok := dcs.clients[protocol]
	if !ok {
		return nil, fmt.Errorf("fail to get dmi client of protocol %s", protocol)
	}
	return dc, nil
}

func (dcs *DMIClients) CreateDMIClient(protocol, sockPath string) {
	dc, err := dcs.getDMIClientByProtocol(protocol)
	if err == nil {
		dcs.mutex.Lock()
		dc.protocol = protocol
		dc.socket = sockPath
		dcs.mutex.Unlock()
		return
	}

	dcs.mutex.Lock()
	dcs.clients[protocol] = &DMIClient{
		protocol: protocol,
		socket:   sockPath,
	}
	dcs.mutex.Unlock()
}

func (dcs *DMIClients) getDMIClientConn(protocol string) (*DMIClient, error) {
	dc, err := dcs.getDMIClientByProtocol(protocol)
	if err != nil {
		return nil, err
	}

	err = dc.connect()
	if err != nil {
		return nil, err
	}
	return dc, nil
}

func (dcs *DMIClients) RegisterDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	cdr, err := createDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to create createDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.RegisterDevice(dc.Ctx, cdr)
	if err != nil {
		return err
	}
	return nil
}

func (dcs *DMIClients) RemoveDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	rdr, err := removeDeviceRequest(device.Name)
	if err != nil {
		return fmt.Errorf("fail to generate RemoveDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.RemoveDevice(dc.Ctx, rdr)
	if err != nil {
		return err
	}
	return nil
}

func (dcs *DMIClients) UpdateDevice(device *v1alpha2.Device) error {
	protocol, err := dtcommon.GetProtocolNameOfDevice(device)
	if err != nil {
		return err
	}

	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	udr, err := updateDeviceRequest(device)
	if err != nil {
		return fmt.Errorf("fail to generate UpdateDeviceRequest for device %s with err: %v", device.Name, err)
	}
	_, err = dc.Client.UpdateDevice(dc.Ctx, udr)
	if err != nil {
		return err
	}
	return nil
}

func (dcs *DMIClients) CreateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	cdmr, err := createDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create RegisterDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.CreateDeviceModel(dc.Ctx, cdmr)
	if err != nil {
		return err
	}
	return nil
}

func (dcs *DMIClients) RemoveDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	rdmr, err := removeDeviceModelRequest(model.Name)
	if err != nil {
		return fmt.Errorf("fail to create RemoveDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.RemoveDeviceModel(dc.Ctx, rdmr)
	if err != nil {
		return err
	}
	return nil
}

func (dcs *DMIClients) UpdateDeviceModel(model *v1alpha2.DeviceModel) error {
	protocol := model.Spec.Protocol
	dc, err := dcs.getDMIClientConn(protocol)
	if err != nil {
		return err
	}

	defer dc.close()

	udmr, err := updateDeviceModelRequest(model)
	if err != nil {
		return fmt.Errorf("fail to create UpdateDeviceModelRequest for device model %s with err: %v", model.Name, err)
	}
	_, err = dc.Client.UpdateDeviceModel(dc.Ctx, udmr)
	if err != nil {
		return err
	}
	return nil
}
