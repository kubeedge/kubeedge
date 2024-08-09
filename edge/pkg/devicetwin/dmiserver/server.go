/*
Copyright 2023 The KubeEdge Authors.

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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	deviceconst "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/common/constants"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	deviceconfig "github.com/kubeedge/kubeedge/edge/pkg/devicetwin/config"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/util"
)

const (
	Limit = 1000
	Burst = 100
)

type server struct {
	pb.UnimplementedDeviceManagerServiceServer
	limiter  *rate.Limiter
	dmiCache *DMICache
}

type DMICache struct {
	MapperMu        *sync.Mutex
	DeviceMu        *sync.Mutex
	DeviceModelMu   *sync.Mutex
	MapperList      map[string]*pb.MapperInfo
	DeviceModelList map[string]*v1beta1.DeviceModel
	DeviceList      map[string]*v1beta1.Device
}

func (s *server) MapperRegister(_ context.Context, in *pb.MapperRegisterRequest) (*pb.MapperRegisterResponse, error) {
	if !s.limiter.Allow() {
		return nil, fmt.Errorf("fail to register mapper because of too many request: %s", in.Mapper.Name)
	}

	if in.Mapper.Protocol == "" {
		klog.Errorf("fail to register mapper %s because the protocol is nil", in.Mapper.Name)
		return nil, fmt.Errorf("fail to register mapper %s because the protocol is nil", in.Mapper.Name)
	}

	klog.V(4).Infof("receive mapper register: %+v", in.Mapper)
	err := saveMapper(in.Mapper)
	if err != nil {
		klog.Errorf("fail to save mapper %s to db with err: %v", in.Mapper.Name, err)
		return nil, err
	}
	s.dmiCache.MapperMu.Lock()
	s.dmiCache.MapperList[in.Mapper.Name] = in.Mapper
	s.dmiCache.MapperMu.Unlock()

	if !in.WithData {
		return &pb.MapperRegisterResponse{}, nil
	}

	var deviceList []*pb.Device
	var deviceModelList []*pb.DeviceModel
	s.dmiCache.DeviceMu.Lock()
	for _, device := range s.dmiCache.DeviceList {
		if device.Spec.Protocol.ProtocolName == in.Mapper.Protocol {
			dev, err := dtcommon.ConvertDevice(device)
			if err != nil {
				klog.Errorf("fail to convert device %s with err: %v", device.Name, err)
				continue
			}
			modelID := util.GetResourceID(device.Namespace, device.Spec.DeviceModelRef.Name)

			s.dmiCache.DeviceModelMu.Lock()
			model, ok := s.dmiCache.DeviceModelList[modelID]
			s.dmiCache.DeviceModelMu.Unlock()
			if !ok {
				klog.Errorf("fail to get device model %s in deviceModelList", device.Spec.DeviceModelRef.Name)
				continue
			}
			dm, err := dtcommon.ConvertDeviceModel(model)
			if err != nil {
				klog.Errorf("fail to convert device model %s with err: %v", device.Spec.DeviceModelRef.Name, err)
				continue
			}
			deviceList = append(deviceList, dev)
			deviceModelList = append(deviceModelList, dm)
		}
	}
	s.dmiCache.DeviceMu.Unlock()

	dmiclient.DMIClientsImp.CreateDMIClient(in.Mapper.Protocol, string(in.Mapper.Address))

	return &pb.MapperRegisterResponse{
		DeviceList: deviceList,
		ModelList:  deviceModelList,
	}, nil
}

func (s *server) ReportDeviceStatus(_ context.Context, in *pb.ReportDeviceStatusRequest) (*pb.
	ReportDeviceStatusResponse, error) {
	if !s.limiter.Allow() {
		return nil, fmt.Errorf("fail to report device status because of too many request: %s", in.DeviceName)
	}

	if in != nil && in.ReportedDevice != nil && in.ReportedDevice.Twins != nil {
		for _, twin := range in.ReportedDevice.Twins {
			msg, err := CreateMessageTwinUpdate(twin)
			if err != nil {
				klog.Errorf("fail to create message data for property %s of device %s with err: %v", twin.PropertyName, in.DeviceName, err)
				return nil, err
			}
			handleDeviceTwin(in, msg)
		}
	} else {
		return &pb.ReportDeviceStatusResponse{}, errors.New("ReportDeviceStatusRequest does not have twin data")
	}

	return &pb.ReportDeviceStatusResponse{}, nil
}

func (s *server) ReportDeviceStates(_ context.Context, in *pb.ReportDeviceStatesRequest) (*pb.
	ReportDeviceStatesResponse, error) {
	if !s.limiter.Allow() {
		return nil, fmt.Errorf("fail to report device states because of too many request: %s", in.DeviceName)
	}

	if in != nil && in.State != "" && in.DeviceName != "" {
		msg, err := CreateMessageStateUpdate(in)
		if err != nil {
			klog.Errorf("fail to create state message data of device %s with err: %v", in.DeviceName, err)
			return nil, err
		}
		handleDeviceState(in, msg)
	} else {
		return &pb.ReportDeviceStatesResponse{}, fmt.Errorf("ReportDeviceStatesRequest is invalid data")
	}

	return &pb.ReportDeviceStatesResponse{}, nil
}

func handleDeviceTwin(in *pb.ReportDeviceStatusRequest, payload []byte) {
	deviceID := util.GetResourceID(in.DeviceNamespace, in.DeviceName)
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.TwinETUpdateSuffix
	target := modules.TwinGroup
	resource := base64.URLEncoding.EncodeToString([]byte(topic))
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	message := beehiveModel.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
		resource, messagepkg.OperationResponse).FillBody(string(payload))

	beehiveContext.SendToGroup(target, *message)
}

func handleDeviceState(in *pb.ReportDeviceStatesRequest, payload []byte) {
	deviceID := util.GetResourceID(in.DeviceNamespace, in.DeviceName)
	topic := dtcommon.DeviceETPrefix + deviceID + dtcommon.DeviceETStateUpdateSuffix
	target := modules.TwinGroup
	resource := base64.URLEncoding.EncodeToString([]byte(topic))
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	message := beehiveModel.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
		resource, messagepkg.OperationResponse).FillBody(string(payload))

	beehiveContext.SendToGroup(target, *message)
}

// CreateMessageTwinUpdate create twin update message.
func CreateMessageTwinUpdate(twin *pb.Twin) ([]byte, error) {
	var updateMsg DeviceTwinUpdate

	updateMsg.BaseMessage.Timestamp = getTimestamp()
	updateMsg.Twin = map[string]*types.MsgTwin{}
	updateMsg.Twin[twin.PropertyName] = &types.MsgTwin{}
	updateMsg.Twin[twin.PropertyName].Expected = &types.TwinValue{Value: &twin.ObservedDesired.Value}
	updateMsg.Twin[twin.PropertyName].Actual = &types.TwinValue{Value: &twin.Reported.Value}

	msg, err := json.Marshal(updateMsg)
	return msg, err
}

// CreateMessageStateUpdate create state update message.
func CreateMessageStateUpdate(in *pb.ReportDeviceStatesRequest) ([]byte, error) {
	var stateMsg DeviceStateUpdate
	stateMsg.BaseMessage.Timestamp = getTimestamp()
	stateMsg.State = in.State
	msg, err := json.Marshal(stateMsg)
	return msg, err
}

func StartDMIServer(cache *DMICache) {
	var DMISockPath string
	if deviceconfig.Get().DeviceTwin.DMISockPath != "" {
		DMISockPath = deviceconfig.Get().DeviceTwin.DMISockPath
	} else {
		DMISockPath = SockPath
	}
	err := initSock(DMISockPath)
	if err != nil {
		klog.Fatalf("failed to remove uds socket with err: %v", err)
		return
	}

	lis, err := net.Listen(deviceconst.UnixNetworkType, DMISockPath)
	if err != nil {
		klog.Errorf("failed to start DMI Server with err: %v", err)
		return
	}

	limiter := rate.NewLimiter(rate.Every(Limit*time.Millisecond), Burst)

	s := grpc.NewServer()
	pb.RegisterDeviceManagerServiceServer(s, &server{
		limiter:  limiter,
		dmiCache: cache,
	})
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		klog.Errorf("failed to start DMI Server with err: %v", err)
		return
	}
	klog.Infoln("success to start DMI Server")
}

func saveMapper(mapper *pb.MapperInfo) error {
	content, err := json.Marshal(mapper)
	if err != nil {
		klog.Errorf("marshal mapper info failed, %s: %v", mapper.Name, err)
		return err
	}
	resource := fmt.Sprintf("%s%s%s%s%s%s%s%s%s", "node", constants.ResourceSep, "nodeID",
		constants.ResourceSep, "namespace", constants.ResourceSep, deviceconst.ResourceTypeDeviceMapper, constants.ResourceSep, mapper.Name)
	meta := &dao.Meta{
		Key:   resource,
		Type:  deviceconst.ResourceTypeDeviceMapper,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		klog.Errorf("save meta failed, %s: %v", mapper.Name, err)
		return err
	}
	klog.Infof("success to save mapper info of %s to db", mapper.Name)
	return nil
}
