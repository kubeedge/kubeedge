/*
Copyright 2025 The KubeEdge Authors.

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmicache"
	"golang.org/x/time/rate"
)

// init initialises the beehive channel context once per test binary so that
// beehiveContext.SendToGroup does not nil-panic when called from Report methods.
func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
}

// registerTwinGroup adds modules.TwinGroup to the channel context so that
// SendToGroup delivers messages to a Receive-able channel in tests.
func registerTwinGroup() {
	module := &common.ModuleInfo{
		ModuleName: modules.TwinGroup,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(module)
	beehiveContext.AddModuleGroup(module.ModuleName, module.ModuleName)
}

// receiveWithTimeout performs a bounded Receive on the given module so that a
// regression fails promptly instead of hanging until the package timeout.
// It calls t.Fatal if no message is delivered within the timeout, making the
// test fail rather than silently pass.
func receiveWithTimeout(t *testing.T, moduleName string) {
	t.Helper()
	type result struct {
		msg beehiveModel.Message
		err error
	}
	ch := make(chan result, 1)
	go func() {
		msg, err := beehiveContext.Receive(moduleName)
		ch <- result{msg, err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("Receive(%s) returned error: %v", moduleName, res.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("no message delivered to %s within timeout — SendToGroup may have dropped it", moduleName)
	}
}

// --- Invalid-request tests ---

func TestReportDeviceStatesInvalidRequest(t *testing.T) {
	s := &server{
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	_, err := s.ReportDeviceStates(context.Background(), &pb.ReportDeviceStatesRequest{})
	assert.Error(t, err, "expected error for empty ReportDeviceStatesRequest")
	assert.Contains(t, err.Error(), "invalid", "error message should describe invalid data")
}

func TestReportDeviceStatusInvalidRequest(t *testing.T) {
	s := &server{
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	_, err := s.ReportDeviceStatus(context.Background(), &pb.ReportDeviceStatusRequest{})
	assert.Error(t, err, "expected error when ReportedDevice is nil")
	assert.Contains(t, err.Error(), "twin", "error message should mention missing twin data")
}

// --- Success tests with message delivery verification ---

func TestReportDeviceStatesSuccess(t *testing.T) {
	registerTwinGroup()

	s := &server{
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	req := &pb.ReportDeviceStatesRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
		State:           "online",
	}

	resp, err := s.ReportDeviceStates(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify that a message was actually delivered to TwinGroup (not silently dropped).
	receiveWithTimeout(t, modules.TwinGroup)
}

func TestReportDeviceStatusSuccess(t *testing.T) {
	registerTwinGroup()

	s := &server{
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	req := &pb.ReportDeviceStatusRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
		ReportedDevice: &pb.DeviceStatus{
			Twins: []*pb.Twin{
				{
					PropertyName: "temperature",
					ObservedDesired: &pb.TwinProperty{
						Value: "25",
					},
					Reported: &pb.TwinProperty{
						Value: "24",
					},
				},
			},
		},
	}

	resp, err := s.ReportDeviceStatus(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify that one message per twin was delivered to TwinGroup.
	receiveWithTimeout(t, modules.TwinGroup)
}

// --- Rate-limit tests ---

func TestReportDeviceStatesRateLimited(t *testing.T) {
	s := &server{
		limiter: rate.NewLimiter(rate.Every(time.Hour), 0),
	}

	req := &pb.ReportDeviceStatesRequest{
		DeviceName:      "device1",
		DeviceNamespace: "default",
		State:           "online",
	}

	_, err := s.ReportDeviceStates(context.Background(), req)
	assert.Error(t, err, "expected rate-limit error")
	assert.Contains(t, err.Error(), "too many request")
}

func TestReportDeviceStatusRateLimited(t *testing.T) {
	s := &server{
		limiter: rate.NewLimiter(rate.Every(time.Hour), 0),
	}

	req := &pb.ReportDeviceStatusRequest{
		DeviceName: "device1",
	}

	_, err := s.ReportDeviceStatus(context.Background(), req)
	assert.Error(t, err, "expected rate-limit error")
	assert.Contains(t, err.Error(), "too many request")
}

// --- MapperRegister tests ---

func TestMapperRegisterNilProtocol(t *testing.T) {
	s := &server{
		limiter:  rate.NewLimiter(rate.Inf, 1),
		dmiCache: dmicache.NewDMICache(),
	}

	req := &pb.MapperRegisterRequest{
		Mapper: &pb.MapperInfo{
			Name:     "mapper1",
			Protocol: "",
		},
	}

	_, err := s.MapperRegister(context.Background(), req)
	assert.Error(t, err, "expected error for nil protocol")
	assert.Contains(t, err.Error(), "protocol", "error message should mention missing protocol")
}

func TestMapperRegisterRateLimit(t *testing.T) {
	s := &server{
		limiter:  rate.NewLimiter(rate.Every(time.Hour), 0),
		dmiCache: dmicache.NewDMICache(),
	}

	req := &pb.MapperRegisterRequest{
		Mapper: &pb.MapperInfo{
			Name:     "mapper1",
			Protocol: "modbus",
		},
	}

	_, err := s.MapperRegister(context.Background(), req)
	assert.Error(t, err, "expected rate-limit error for MapperRegister")
	assert.Contains(t, err.Error(), "too many request")
}
