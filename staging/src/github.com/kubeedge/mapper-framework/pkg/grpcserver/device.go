package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/avast/retry-go"
	"k8s.io/klog/v2"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
	"github.com/kubeedge/mapper-framework/pkg/util/parse"
)

func (s *Server) RegisterDevice(_ context.Context, request *dmiapi.RegisterDeviceRequest) (*dmiapi.RegisterDeviceResponse, error) {
	klog.V(3).Info("RegisterDevice")
	device := request.GetDevice()
	if device == nil {
		return nil, errors.New("device is nil")
	}
	deviceID := parse.GetResourceID(device.Namespace, device.Name)
	if _, err := s.devPanel.GetDevice(deviceID); err == nil {
		// The device has been registered
		return &dmiapi.RegisterDeviceResponse{DeviceName: device.Name}, nil
	}

	var model common.DeviceModel
	var err error
	modelID := parse.GetResourceID(device.Namespace, device.Spec.DeviceModelReference)
	err = retry.Do(
		func() error {
			model, err = s.devPanel.GetModel(modelID)
			return err
		},
		retry.Delay(1*time.Second),
		retry.Attempts(3),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return nil, fmt.Errorf("deviceModel %s in %s namespace not found, err: %s", device.Spec.DeviceModelReference, device.Namespace, err)
	}
	protocol, err := parse.BuildProtocolFromGrpc(device)
	if err != nil {
		return nil, fmt.Errorf("parse device %s protocol in %s namespace failed, err: %s", device.Name, device.Namespace, err)
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
	deviceID := parse.GetResourceID(request.GetDeviceNamespace(), request.GetDeviceName())

	return &dmiapi.RemoveDeviceResponse{}, s.devPanel.RemoveDevice(deviceID)
}

func (s *Server) UpdateDevice(_ context.Context, request *dmiapi.UpdateDeviceRequest) (*dmiapi.UpdateDeviceResponse, error) {
	klog.V(3).Info("UpdateDevice")
	device := request.GetDevice()
	if device == nil {
		return nil, errors.New("device is nil")
	}

	modelID := parse.GetResourceID(device.GetNamespace(), device.Spec.DeviceModelReference)
	model, err := s.devPanel.GetModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("deviceModel %s in %s namespace not found, err: %s", device.Spec.DeviceModelReference, device.GetNamespace(), err)
	}
	klog.V(3).Infof("model: %+v", model)
	protocol, err := parse.BuildProtocolFromGrpc(device)
	if err != nil {
		return nil, fmt.Errorf("parse device %s protocol failed, err: %s", device.Name, err)
	}
	deviceInstance, err := parse.GetDeviceFromGrpc(device, &model)
	if err != nil {
		return nil, fmt.Errorf("parse device %s instance in %s namespace failed, err: %s", device.Name, device.Namespace, err)
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
	modelID := parse.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	if _, err := s.devPanel.GetModel(modelID); err != nil {
		return nil, fmt.Errorf("update deviceModel %s failed, not existed", deviceModel.Name)
	}

	model := parse.GetDeviceModelFromGrpc(deviceModel)

	s.devPanel.UpdateModel(&model)

	return &dmiapi.UpdateDeviceModelResponse{}, nil
}

func (s *Server) RemoveDeviceModel(_ context.Context, request *dmiapi.RemoveDeviceModelRequest) (*dmiapi.RemoveDeviceModelResponse, error) {
	modelID := parse.GetResourceID(request.ModelNamespace, request.ModelName)
	s.devPanel.RemoveModel(modelID)
	return &dmiapi.RemoveDeviceModelResponse{}, nil
}

func (s *Server) GetDevice(_ context.Context, request *dmiapi.GetDeviceRequest) (*dmiapi.GetDeviceResponse, error) {
	if request.GetDeviceName() == "" {
		return nil, errors.New("device name is nil")
	}
	deviceID := parse.GetResourceID(request.GetDeviceNamespace(), request.GetDeviceName())
	device, err := s.devPanel.GetDevice(deviceID)
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
