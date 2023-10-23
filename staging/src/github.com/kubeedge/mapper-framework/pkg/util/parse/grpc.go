package parse

import (
	"encoding/json"
	"errors"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/pkg/common"
	dmiapi "github.com/kubeedge/Template/pkg/dmi-api"
)

type TwinResultResponse struct {
	PropertyName string `json:"property_name"`
	Payload      []byte `json:"payload"`
}

func getProtocolNameFromGrpc(device *dmiapi.Device) (string, error) {

	return device.Spec.Protocol.ProtocolName, nil
}

func getPushMethodFromGrpc(visitor *dmiapi.DeviceProperty) (string, error) {
	// TODO add more push method
	if visitor.PushMethod.Http != nil {
		return "http", nil
	}
	if visitor.PushMethod.Mqtt != nil {
		return "mqtt", nil
	}
	return "", errors.New("can not parse publish method")
}

func getDBMethodFromGrpc(visitor *dmiapi.DeviceProperty) (string, error) {
	// TODO add more dbMethod
	if visitor.PushMethod.DBMethod.Influxdb2 != nil {
		return "influx", nil
	}
	return "", errors.New("can not parse dbMethod")
}

func BuildProtocolFromGrpc(device *dmiapi.Device) (common.ProtocolConfig, error) {
	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return common.ProtocolConfig{}, err
	}

	var protocolConfig []byte

	customizedProtocol := make(map[string]interface{})
	customizedProtocol["protocolName"] = protocolName
	if device.Spec.Protocol.ConfigData != nil {
		recvAdapter := make(map[string]interface{})
		for k, v := range device.Spec.Protocol.ConfigData.Data {
			value, err := common.DecodeAnyValue(v)
			if err != nil {
				continue
			}
			recvAdapter[k] = value
		}
		customizedProtocol["configData"] = recvAdapter
	}
	protocolConfig, err = json.Marshal(customizedProtocol)
	if err != nil {
		return common.ProtocolConfig{}, err
	}

	return common.ProtocolConfig{
		ProtocolName: protocolName,
		ConfigData:   protocolConfig,
	}, nil
}

func buildTwinsFromGrpc(device *dmiapi.Device) []common.Twin {
	if len(device.Status.Twins) == 0 {
		return nil
	}
	res := make([]common.Twin, 0, len(device.Status.Twins))
	for _, twin := range device.Status.Twins {
		cur := common.Twin{
			PropertyName: twin.PropertyName,

			ObservedDesired: common.TwinProperty{
				Value: twin.ObservedDesired.Value,
				Metadata: common.Metadata{
					Timestamp: twin.ObservedDesired.Metadata["timestamp"],
					Type:      twin.ObservedDesired.Metadata["type"],
				},
			},
			Reported: common.TwinProperty{
				Value: twin.Reported.Value,
				Metadata: common.Metadata{
					Timestamp: twin.ObservedDesired.Metadata["timestamp"],
					Type:      twin.ObservedDesired.Metadata["type"],
				},
			},
		}
		res = append(res, cur)
	}
	return res
}

func buildPropertiesFromGrpc(device *dmiapi.Device) []common.DeviceProperty {
	if len(device.Spec.Properties) == 0 {
		return nil
	}
	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return nil
	}
	res := make([]common.DeviceProperty, 0, len(device.Spec.Properties))
	klog.V(3).Infof("In buildPropertiesFromGrpc, PropertyVisitors = %v", device.Spec.Properties)
	for _, pptv := range device.Spec.Properties {

		// get visitorConfig filed by grpc device instance
		var visitorConfig []byte
		recvAdapter := make(map[string]interface{})
		for k, v := range pptv.Visitors.ConfigData.Data {
			value, err := common.DecodeAnyValue(v)
			if err != nil {
				continue
			}
			recvAdapter[k] = value
		}
		customizedProtocol := make(map[string]interface{})
		customizedProtocol["protocolName"] = pptv.Visitors.ProtocolName
		customizedProtocol["configData"] = recvAdapter
		visitorConfig, err = json.Marshal(customizedProtocol)
		if err != nil {
			klog.Errorf("err: %+v", err)
			return nil
		}

		// get dbMethod filed by grpc device instance
		var dbMethodName string
		var dbconfig common.DBConfig
		if pptv.PushMethod.DBMethod != nil {
			dbMethodName, err = getDBMethodFromGrpc(pptv)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
			switch dbMethodName {
			case "influx":
				clientconfig, err := json.Marshal(pptv.PushMethod.DBMethod.Influxdb2.Influxdb2ClientConfig)
				if err != nil {
					klog.Errorf("err: %+v", err)
					return nil
				}
				dataconfig, err := json.Marshal(pptv.PushMethod.DBMethod.Influxdb2.Influxdb2DataConfig)
				if err != nil {
					klog.Errorf("err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					Influxdb2ClientConfig: clientconfig,
					Influxdb2DataConfig:   dataconfig,
				}
			}
		}

		// get pushMethod filed by grpc device instance
		pushMethodName, err := getPushMethodFromGrpc(pptv)
		if err != nil {
			klog.Errorf("err: %+v", err)
			return nil
		}
		var pushMethod []byte
		switch pushMethodName {
		case "http":
			pushMethod, err = json.Marshal(pptv.PushMethod.Http)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		case "mqtt":
			pushMethod, err = json.Marshal(pptv.PushMethod.Mqtt)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		}

		// get the final Properties
		cur := common.DeviceProperty{
			Name:         pptv.GetName(),
			PropertyName: pptv.GetName(),
			ModelName:    device.Spec.DeviceModelReference,
			CollectCycle: pptv.GetCollectCycle(),
			ReportCycle:  pptv.GetReportCycle(),
			ReportToCloud: pptv.GetReportToCloud(),
			Protocol:     protocolName,
			Visitors:     visitorConfig,
			PushMethod: common.PushMethodConfig{
				MethodName:   pushMethodName,
				MethodConfig: pushMethod,
				DBMethod: common.DBMethodConfig{
					DBMethodName: dbMethodName,
					DBConfig:     dbconfig,
				},
			},
		}
		res = append(res, cur)

	}
	return res
}

func ParseDeviceModelFromGrpc(model *dmiapi.DeviceModel) common.DeviceModel {
	cur := common.DeviceModel{
		Name: model.GetName(),
	}
	if model.GetSpec() == nil || len(model.GetSpec().GetProperties()) == 0 {
		return cur
	}
	properties := make([]common.ModelProperty, 0, len(model.Spec.Properties))
	for _, property := range model.Spec.Properties {
		p := common.ModelProperty{
			Name:        property.GetName(),
			Description: property.GetDescription(),
			DataType:    property.Type,
			AccessMode:  property.AccessMode,
			Minimum:     property.Minimum,
			Maximum:     property.Maximum,
			Unit:        property.Unit,
		}
		properties = append(properties, p)
	}
	cur.Properties = properties
	return cur
}

func ParseDeviceFromGrpc(device *dmiapi.Device, commonModel *common.DeviceModel) (*common.DeviceInstance, error) {

	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return nil, err
	}
	instance := &common.DeviceInstance{
		ID:           device.GetName(),
		Name:         device.GetName(),
		ProtocolName: protocolName + "-" + device.GetName(),
		Model:        device.Spec.DeviceModelReference,
		Twins:        buildTwinsFromGrpc(device),
		Properties:   buildPropertiesFromGrpc(device),
	}
	// copy Properties to twin
	propertiesMap := make(map[string]common.DeviceProperty)
	for i := 0; i < len(instance.Properties); i++ {
		if commonModel == nil {
			klog.Errorf("commonModel == nil")
			continue
		}

		// parse the content of the modelproperty field into instance
		for _, property := range commonModel.Properties {
			if property.Name == instance.Properties[i].PropertyName {
				instance.Properties[i].PProperty = property
				break
			}
		}
		propertiesMap[instance.Properties[i].PProperty.Name] = instance.Properties[i]
	}
	for i := 0; i < len(instance.Twins); i++ {
		if v, ok := propertiesMap[instance.Twins[i].PropertyName]; ok {
			instance.Twins[i].Property = &v
		}
	}
	klog.V(2).Infof("final instance data from grpc = %v", instance)
	return instance, nil
}
