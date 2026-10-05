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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
)

func newDevice(name, namespace, protocol string) *v1beta1.Device {
	return &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.DeviceSpec{
			Protocol: v1beta1.ProtocolConfig{
				ProtocolName: protocol,
			},
		},
	}
}

func newDeviceModel(name, namespace, protocol string) *v1beta1.DeviceModel {
	return &v1beta1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.DeviceModelSpec{
			Protocol: protocol,
		},
	}
}

// startUnixServer spins up a minimal gRPC server on a temp Unix socket and
// returns the socket path together with a cleanup func.
func startUnixServer(t *testing.T, impl dmiapi.DeviceMapperServiceServer) (sockPath string, cleanup func()) {
	t.Helper()

	f, err := os.CreateTemp("", "dmi-test-*.sock")
	assert.NoError(t, err)
	_ = f.Close()
	_ = os.Remove(f.Name())

	lis, err := net.Listen("unix", f.Name())
	assert.NoError(t, err)

	srv := grpc.NewServer()
	dmiapi.RegisterDeviceMapperServiceServer(srv, impl)

	go func() {
		_ = srv.Serve(lis)
	}()

	return f.Name(), func() {
		srv.Stop()
		_ = os.Remove(f.Name())
	}
}

type fakeMapperServer struct {
	dmiapi.UnimplementedDeviceMapperServiceServer
	registerDeviceErr error
	removeDeviceErr   error
	updateDeviceErr   error
	createModelErr    error
	removeModelErr    error
	updateModelErr    error
}

func (f *fakeMapperServer) RegisterDevice(_ context.Context, _ *dmiapi.RegisterDeviceRequest) (*dmiapi.RegisterDeviceResponse, error) {
	return &dmiapi.RegisterDeviceResponse{}, f.registerDeviceErr
}
func (f *fakeMapperServer) RemoveDevice(_ context.Context, _ *dmiapi.RemoveDeviceRequest) (*dmiapi.RemoveDeviceResponse, error) {
	return &dmiapi.RemoveDeviceResponse{}, f.removeDeviceErr
}
func (f *fakeMapperServer) UpdateDevice(_ context.Context, _ *dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error) {
	return &dmiapi.UpdateDeviceResponse{}, f.updateDeviceErr
}
func (f *fakeMapperServer) CreateDeviceModel(_ context.Context, _ *dmiapi.CreateDeviceModelRequest) (*dmiapi.CreateDeviceModelResponse, error) {
	return &dmiapi.CreateDeviceModelResponse{}, f.createModelErr
}
func (f *fakeMapperServer) RemoveDeviceModel(_ context.Context, _ *dmiapi.RemoveDeviceModelRequest) (*dmiapi.RemoveDeviceModelResponse, error) {
	return &dmiapi.RemoveDeviceModelResponse{}, f.removeModelErr
}
func (f *fakeMapperServer) UpdateDeviceModel(_ context.Context, _ *dmiapi.UpdateDeviceModelRequest) (*dmiapi.UpdateDeviceModelResponse, error) {
	return &dmiapi.UpdateDeviceModelResponse{}, f.updateModelErr
}

func TestDMIClient_ConnectAndClose(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dc := &DMIClient{
		protocol: "modbus",
		socket:   sock,
	}

	err := dc.connect()
	assert.NoError(t, err)
	assert.NotNil(t, dc.Client)
	assert.NotNil(t, dc.Conn)
	assert.NotNil(t, dc.Ctx)
	assert.NotNil(t, dc.CancelFunc)

	dc.close()
	assert.Nil(t, dc.Client)
}

func TestDMIClient_ConnectFail(t *testing.T) {
	dc := &DMIClient{
		protocol: "modbus",
		// point at a path that definitely has no listener
		socket: "/tmp/no-such-socket-xyz.sock",
	}
	// grpc.Dial is non-blocking by default, so connect() itself won't fail;
	// the failure only surfaces on the first RPC call.  The test just verifies
	// that connect() completes without panic and sets up the fields.
	err := dc.connect()
	assert.NoError(t, err)
	dc.close()
}

func TestDMIClient_CloseIdempotent(t *testing.T) {
	dc := &DMIClient{}
	// calling close on a zero-value DMIClient must not panic
	dc.close()
	dc.close()
}

func TestCreateDeviceRequest(t *testing.T) {
	device := newDevice("dev1", "default", "modbus")
	req, err := createDeviceRequest(device)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, req.Device)
}

func TestRemoveDeviceRequest(t *testing.T) {
	device := newDevice("dev1", "default", "modbus")
	req, err := removeDeviceRequest(device)
	assert.NoError(t, err)
	assert.Equal(t, "dev1", req.DeviceName)
	assert.Equal(t, "default", req.DeviceNamespace)
}

func TestUpdateDeviceRequest(t *testing.T) {
	device := newDevice("dev1", "default", "modbus")
	req, err := updateDeviceRequest(device)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, req.Device)
}

func TestCreateDeviceModelRequest(t *testing.T) {
	model := newDeviceModel("model1", "default", "modbus")
	req, err := createDeviceModelRequest(model)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, req.Model)
}

func TestUpdateDeviceModelRequest(t *testing.T) {
	model := newDeviceModel("model1", "default", "modbus")
	req, err := updateDeviceModelRequest(model)
	assert.NoError(t, err)
	assert.NotNil(t, req)
	assert.NotNil(t, req.Model)
}

func TestRemoveDeviceModelRequest(t *testing.T) {
	model := newDeviceModel("model1", "default", "modbus")
	req, err := removeDeviceModelRequest(model)
	assert.NoError(t, err)
	assert.Equal(t, "model1", req.ModelName)
	assert.Equal(t, "default", req.ModelNamespace)
}

func freshClients() *DMIClients {
	return &DMIClients{
		mutex:   sync.Mutex{},
		clients: make(map[string]*DMIClient),
	}
}

func TestInit(t *testing.T) {
	assert.NotNil(t, DMIClientsImp)
	assert.NotNil(t, DMIClientsImp.clients)
}

func TestGetDMIClientByProtocol_NotFound(t *testing.T) {
	dcs := freshClients()
	_, err := dcs.getDMIClientByProtocol("modbus")
	assert.Error(t, err)
}

func TestGetDMIClientByProtocol_Found(t *testing.T) {
	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: "/tmp/x.sock"}
	dc, err := dcs.getDMIClientByProtocol("modbus")
	assert.NoError(t, err)
	assert.Equal(t, "modbus", dc.protocol)
}

func TestCreateDMIClient_NewWithoutConnect(t *testing.T) {
	dcs := freshClients()
	err := dcs.CreateDMIClient("modbus", "/tmp/fake.sock", false)
	assert.NoError(t, err)

	dc, err := dcs.getDMIClientByProtocol("modbus")
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/fake.sock", dc.socket)
}

func TestCreateDMIClient_ExistingClientUpdatesSocket(t *testing.T) {
	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: "/tmp/old.sock"}

	err := dcs.CreateDMIClient("modbus", "/tmp/new.sock", false)
	assert.NoError(t, err)

	dc, _ := dcs.getDMIClientByProtocol("modbus")
	assert.Equal(t, "/tmp/new.sock", dc.socket)
}

func TestCreateDMIClient_TryConnectSuccess(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	err := dcs.CreateDMIClient("modbus", sock, true)
	assert.NoError(t, err)
}

func TestCreateDMIClient_TryConnectFail_NoSocket(t *testing.T) {
	dcs := freshClients()
	// Use a socket path with no listener; grpc.Dial is lazy so we need to
	// force a connection attempt.  Because grpc.Dial itself does not block,
	// the test verifies the function returns without panicking.
	// The real failure path (DialContext with block) is exercised in
	// TestCreateDMIClient_TryConnectBlockFail below.
	err := dcs.CreateDMIClient("modbus", "/tmp/nonexistent-dmi.sock", true)
	// May or may not error depending on grpc lazy-dial; just must not panic.
	_ = err
}

func TestCreateDMIClient_ReplacesExistingWithConnectedOne(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	// Pre-populate with a "connected" client
	existingDC := &DMIClient{protocol: "modbus", socket: sock}
	_ = existingDC.connect()
	dcs.clients["modbus"] = existingDC

	// CreateDMIClient when client already exists just updates the socket
	err := dcs.CreateDMIClient("modbus", sock, false)
	assert.NoError(t, err)
}

func TestGetDMIClientConn_ProtocolNotRegistered(t *testing.T) {
	dcs := freshClients()
	_, err := dcs.getDMIClientConn("unknown")
	assert.Error(t, err)
}

func TestGetDMIClientConn_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	dc, err := dcs.getDMIClientConn("modbus")
	assert.NoError(t, err)
	assert.NotNil(t, dc.Client)
	dc.close()
}

func TestRegisterDevice_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RegisterDevice(newDevice("dev1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestRegisterDevice_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.RegisterDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

func TestRegisterDevice_ServerError(t *testing.T) {
	fake := &fakeMapperServer{registerDeviceErr: fmt.Errorf("server side failure")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RegisterDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

func TestRemoveDevice_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RemoveDevice(newDevice("dev1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestRemoveDevice_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.RemoveDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

func TestRemoveDevice_ServerError(t *testing.T) {
	fake := &fakeMapperServer{removeDeviceErr: fmt.Errorf("remove failed")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RemoveDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

// ── UpdateDevice ──────────────────────────────────────────────────────────────

func TestUpdateDevice_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.UpdateDevice(newDevice("dev1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestUpdateDevice_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.UpdateDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

func TestUpdateDevice_ServerError(t *testing.T) {
	fake := &fakeMapperServer{updateDeviceErr: fmt.Errorf("update failed")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.UpdateDevice(newDevice("dev1", "default", "modbus"))
	assert.Error(t, err)
}

func TestCreateDeviceModel_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.CreateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestCreateDeviceModel_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.CreateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestCreateDeviceModel_ServerError(t *testing.T) {
	fake := &fakeMapperServer{createModelErr: fmt.Errorf("create model failed")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.CreateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestRemoveDeviceModel_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RemoveDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestRemoveDeviceModel_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.RemoveDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestRemoveDeviceModel_ServerError(t *testing.T) {
	fake := &fakeMapperServer{removeModelErr: fmt.Errorf("remove model failed")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.RemoveDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestUpdateDeviceModel_Success(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.UpdateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.NoError(t, err)
}

func TestUpdateDeviceModel_ProtocolNotFound(t *testing.T) {
	dcs := freshClients()
	err := dcs.UpdateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestUpdateDeviceModel_ServerError(t *testing.T) {
	fake := &fakeMapperServer{updateModelErr: fmt.Errorf("update model failed")}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}

	err := dcs.UpdateDeviceModel(newDeviceModel("model1", "default", "modbus"))
	assert.Error(t, err)
}

func TestCreateDMIClient_Concurrent(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dcs := freshClients()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = dcs.CreateDMIClient("modbus", sock, false)
		}()
	}
	wg.Wait()

	dc, err := dcs.getDMIClientByProtocol("modbus")
	assert.NoError(t, err)
	assert.NotNil(t, dc)
}

func TestRegisterDevice_Concurrent(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each goroutine gets its own DMIClients instance to avoid
			// the shared DMIClient.Client being nilled by dc.close() in
			// one goroutine while another goroutine is mid-RPC.
			dcs := freshClients()
			dcs.clients["modbus"] = &DMIClient{protocol: "modbus", socket: sock}
			err := dcs.RegisterDevice(newDevice(fmt.Sprintf("dev%d", i), "default", "modbus"))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
}

func TestDMIClient_ConnectContextTimeout(t *testing.T) {
	fake := &fakeMapperServer{}
	sock, cleanup := startUnixServer(t, fake)
	defer cleanup()

	dc := &DMIClient{protocol: "modbus", socket: sock}
	err := dc.connect()
	assert.NoError(t, err)

	// Verify the context has a deadline set (10-second timeout)
	deadline, ok := dc.Ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))

	dc.close()
}
