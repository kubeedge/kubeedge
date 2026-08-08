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

package dmiserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmicache"
)

func newTestServer() *server {
	return &server{limiter: rate.NewLimiter(rate.Inf, 1), dmiCache: dmicache.NewDMICache()}
}

func newThrottledServer() *server {
	return &server{limiter: rate.NewLimiter(0, 0), dmiCache: dmicache.NewDMICache()}
}

func TestCreateMessageTwinUpdate_Basic(t *testing.T) {
	twin := &pb.Twin{
		PropertyName:    "temperature",
		ObservedDesired: &pb.TwinProperty{Value: "25"},
		Reported:        &pb.TwinProperty{Value: "24"},
	}
	payload, err := CreateMessageTwinUpdate(twin)
	require.NoError(t, err)
	var msg DeviceTwinUpdate
	require.NoError(t, json.Unmarshal(payload, &msg))
	require.Contains(t, msg.Twin, "temperature")
	assert.Equal(t, "25", *msg.Twin["temperature"].Expected.Value)
	assert.Equal(t, "24", *msg.Twin["temperature"].Actual.Value)
}

func TestCreateMessageTwinUpdate_EmptyPropertyName(t *testing.T) {
	twin := &pb.Twin{PropertyName: "", ObservedDesired: &pb.TwinProperty{Value: "1"}, Reported: &pb.TwinProperty{Value: "0"}}
	payload, err := CreateMessageTwinUpdate(twin)
	require.NoError(t, err)
	var msg DeviceTwinUpdate
	require.NoError(t, json.Unmarshal(payload, &msg))
	_, ok := msg.Twin[""]
	assert.True(t, ok)
}

func TestCreateMessageTwinUpdate_TimestampSet(t *testing.T) {
	twin := &pb.Twin{PropertyName: "speed", ObservedDesired: &pb.TwinProperty{Value: "100"}, Reported: &pb.TwinProperty{Value: "90"}}
	payload, err := CreateMessageTwinUpdate(twin)
	require.NoError(t, err)
	var msg DeviceTwinUpdate
	require.NoError(t, json.Unmarshal(payload, &msg))
	assert.Greater(t, msg.BaseMessage.Timestamp, int64(0))
}

func TestCreateMessageStateUpdate_Online(t *testing.T) {
	req := &pb.ReportDeviceStatesRequest{DeviceName: "robot", DeviceNamespace: "default", State: "online"}
	payload, err := CreateMessageStateUpdate(req)
	require.NoError(t, err)
	var msg DeviceStateUpdate
	require.NoError(t, json.Unmarshal(payload, &msg))
	assert.Equal(t, "online", msg.State)
	assert.Greater(t, msg.BaseMessage.Timestamp, int64(0))
}

func TestCreateMessageStateUpdate_Offline(t *testing.T) {
	req := &pb.ReportDeviceStatesRequest{DeviceName: "sensor", State: "offline"}
	payload, err := CreateMessageStateUpdate(req)
	require.NoError(t, err)
	var msg DeviceStateUpdate
	require.NoError(t, json.Unmarshal(payload, &msg))
	assert.Equal(t, "offline", msg.State)
}

func TestMapperRegister_RateLimitExceeded(t *testing.T) {
	s := newThrottledServer()
	_, err := s.MapperRegister(context.Background(), &pb.MapperRegisterRequest{Mapper: &pb.MapperInfo{Name: "m", Protocol: "modbus"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many request")
}

func TestMapperRegister_EmptyProtocol(t *testing.T) {
	s := newTestServer()
	_, err := s.MapperRegister(context.Background(), &pb.MapperRegisterRequest{Mapper: &pb.MapperInfo{Name: "m", Protocol: ""}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "protocol is nil")
}

func TestReportDeviceStatus_RateLimitExceeded(t *testing.T) {
	s := newThrottledServer()
	req := &pb.ReportDeviceStatusRequest{
		DeviceName: "d1", DeviceNamespace: "default",
		ReportedDevice: &pb.DeviceStatus{Twins: []*pb.Twin{{PropertyName: "t", ObservedDesired: &pb.TwinProperty{Value: "1"}, Reported: &pb.TwinProperty{Value: "1"}}}},
	}
	_, err := s.ReportDeviceStatus(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many request")
}

func TestReportDeviceStatus_NilRequest(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStatus(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have twin data")
}

func TestReportDeviceStatus_NilReportedDevice(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStatus(context.Background(), &pb.ReportDeviceStatusRequest{DeviceName: "d2", ReportedDevice: nil})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have twin data")
}

func TestReportDeviceStatus_NilTwins(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStatus(context.Background(), &pb.ReportDeviceStatusRequest{DeviceName: "d3", ReportedDevice: &pb.DeviceStatus{Twins: nil}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not have twin data")
}

func TestReportDeviceStates_RateLimitExceeded(t *testing.T) {
	s := newThrottledServer()
	_, err := s.ReportDeviceStates(context.Background(), &pb.ReportDeviceStatesRequest{DeviceName: "d4", State: "online"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many request")
}

func TestReportDeviceStates_EmptyState(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStates(context.Background(), &pb.ReportDeviceStatesRequest{DeviceName: "d5", State: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data")
}

func TestReportDeviceStates_EmptyDeviceName(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStates(context.Background(), &pb.ReportDeviceStatesRequest{DeviceName: "", State: "online"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data")
}

func TestReportDeviceStates_NilRequest(t *testing.T) {
	s := newTestServer()
	_, err := s.ReportDeviceStates(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data")
}

func TestGetTimestamp_Positive(t *testing.T) {
	assert.Greater(t, getTimestamp(), int64(0))
}

func TestGetTimestamp_Increasing(t *testing.T) {
	assert.GreaterOrEqual(t, getTimestamp(), getTimestamp()-1)
}

func TestDeviceTwinUpdate_JSONRoundTrip(t *testing.T) {
	val := "42"
	orig := DeviceTwinUpdate{
		BaseMessage: types.BaseMessage{Timestamp: 1234567890},
		Twin:        map[string]*types.MsgTwin{"speed": {Expected: &types.TwinValue{Value: &val}}},
	}
	data, err := json.Marshal(orig)
	require.NoError(t, err)
	var decoded DeviceTwinUpdate
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, orig.BaseMessage.Timestamp, decoded.BaseMessage.Timestamp)
	assert.Equal(t, val, *decoded.Twin["speed"].Expected.Value)
}

func TestDeviceStateUpdate_JSONRoundTrip(t *testing.T) {
	orig := DeviceStateUpdate{BaseMessage: types.BaseMessage{Timestamp: 9999}, State: "unhealthy"}
	data, err := json.Marshal(orig)
	require.NoError(t, err)
	var decoded DeviceStateUpdate
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "unhealthy", decoded.State)
	assert.Equal(t, int64(9999), decoded.BaseMessage.Timestamp)
}

func TestMapperRegister_ErrorCode(t *testing.T) {
	s := newThrottledServer()
	_, err := s.MapperRegister(context.Background(), &pb.MapperRegisterRequest{Mapper: &pb.MapperInfo{Name: "m", Protocol: "mqtt"}})
	require.Error(t, err)
	assert.NotEqual(t, codes.OK, status.Code(err))
}
