package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/avast/retry-go"
	"k8s.io/klog/v2"

	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

func (s *Server) RegisterDevice(_ context.Context, request *dmiapi.RegisterDeviceRequest) (*dmiapi.RegisterDeviceResponse, error) {
	klog.V(3).Info("RegisterDevice")
	device := request.GetDevice()
	if device == nil {
		return nil, errors.New("device is nil")
	}
	if _, err := s.devPanel.GetDevice(device.Name); err == nil {
		// The device has been registered
		return &dmiapi.RegisterDeviceResponse{DeviceName: device.Name}, nil
	}

	var model common.DeviceModel
	var err error
	err = retry.Do(
		func() error {
			model, err = s.devPanel.GetModel(device.Spec.DeviceModelReference)
			return err
		},
		retry.Delay(1*time.Second),
		retry.Attempts(3),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return nil, fmt.Errorf("deviceModel %s not found, err: %s", device.Spec.DeviceModelReference, err)
	}
	protocol, err := parse.BuildProtocolFromGrpc(device)
	if err != nil {
		return nil, fmt.Errorf("parse device %s protocol failed, err: %s", device.Name, err)
	}
	klog.Infof("model: %+v", model)
	deviceInstance, err := parse.GetDeviceFromGrpc(device, &model)

	if err != nil {
		return nil, fmt.Errorf("parse device %s instance failed, err: %s", device.Name, err)
	}

	deviceInstance.PProtocol = protocol
	s.devPanel.UpdateDev(&model, deviceInstance)

	return &dmiapi.RegisterDeviceResponse{DeviceName: device.Name}, nil
}

func (s *Server) RemoveDevice(_ context.Context, request *dmiapi.RemoveDeviceRequest) (*dmiapi.RemoveDeviceResponse, error) {
	if request.GetDeviceName() == "" {
		return nil, errors.New("device name is nil")
	}

	return &dmiapi.RemoveDeviceResponse{}, s.devPanel.RemoveDevice(request.GetDeviceName())
}

func (s *Server) UpdateDevice(_ context.Context, request *dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error) {
	klog.V(3).Info("UpdateDevice")
	device := request.GetDevice()
	if device == nil {
		return nil, errors.New("device is nil")
	}

	model, err := s.devPanel.GetModel(device.Spec.DeviceModelReference)
	if err != nil {
		return nil, fmt.Errorf("deviceModel %s not found, err: %s", device.Spec.DeviceModelReference, err)
	}
	protocol, err := parse.BuildProtocolFromGrpc(device)
	if err != nil {
		return nil, fmt.Errorf("parse device %s protocol failed, err: %s", device.Name, err)
	}

	klog.V(3).Infof("model: %+v", model)
	deviceInstance, err := parse.GetDeviceFromGrpc(device, &model)
	if err != nil {
		return nil, fmt.Errorf("parse device %s instance failed, err: %s", device.Name, err)
	}
	deviceInstance.PProtocol = protocol

	s.devPanel.UpdateDev(&model, deviceInstance)

	return &dmiapi.UpdateDeviceResponse{}, nil
}

func (s *Server) CreateDeviceModel(_ context.Context, request *dmiapi.CreateDeviceModelRequest) (*dmiapi.CreateDeviceModelResponse, error) {
	deviceModel := request.GetModel()
	klog.Infof("start create deviceModel: %v", deviceModel.Name)
	if deviceModel == nil {
		return nil, errors.New("deviceModel is nil")
	}

	model := parse.GetDeviceModelFromGrpc(deviceModel)

	s.devPanel.UpdateModel(&model)

	klog.Infof("create deviceModel done: %v", deviceModel.Name)

	return &dmiapi.CreateDeviceModelResponse{DeviceModelName: deviceModel.Name}, nil
}

func (s *Server) UpdateDeviceModel(_ context.Context, request *dmiapi.UpdateDeviceModelRequest) (*dmiapi.UpdateDeviceModelResponse, error) {
	deviceModel := request.GetModel()
	if deviceModel == nil {
		return nil, errors.New("deviceModel is nil")
	}
	if _, err := s.devPanel.GetModel(deviceModel.Name); err != nil {
		return nil, fmt.Errorf("update deviceModel %s failed, not existed", deviceModel.Name)
	}

	model := parse.GetDeviceModelFromGrpc(deviceModel)

	s.devPanel.UpdateModel(&model)

	return &dmiapi.UpdateDeviceModelResponse{}, nil
}

func (s *Server) RemoveDeviceModel(_ context.Context, request *dmiapi.RemoveDeviceModelRequest) (*dmiapi.RemoveDeviceModelResponse, error) {
	s.devPanel.RemoveModel(request.ModelName)

	return &dmiapi.RemoveDeviceModelResponse{}, nil
}

func (s *Server) GetDevice(_ context.Context, request *dmiapi.GetDeviceRequest) (*dmiapi.GetDeviceResponse, error) {
	if request.GetDeviceName() == "" {
		return nil, errors.New("device name is nil")
	}

	device, err := s.devPanel.GetDevice(request.GetDeviceName())
	if err != nil {
		return nil, err
	}
	res := &dmiapi.GetDeviceResponse{
		Device: &dmiapi.Device{
			Status: &dmiapi.DeviceStatus{},
		},
	}
	deviceValue := reflect.ValueOf(device)
	twinsValue := deviceValue.FieldByName("Instance").FieldByName("Twins")
	if !twinsValue.IsValid() {
		return nil, fmt.Errorf("twins field not found")
	}
	twins, err := parse.ConvTwinsToGrpc(twinsValue.Interface().([]common.Twin))
	if err != nil {
		return nil, err
	}
	res.Device.Status.Twins = twins
	//res.Device.Status.State = common.DEVSTOK
	return res, nil
}
