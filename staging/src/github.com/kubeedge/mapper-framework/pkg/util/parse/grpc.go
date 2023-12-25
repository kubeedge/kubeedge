package parse

import (
	"encoding/json"
	"errors"

	"k8s.io/klog/v2"

	dmiapi "github.com/kubeedge/kubeedge/pkg/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

type TwinResultResponse struct {
	PropertyName string `json:"property_name"`
	Payload      []byte `json:"payload"`
}

func getProtocolNameFromGrpc(device *dmiapi.Device) (string, error) {
	return device.Spec.Protocol.ProtocolName, nil
}

func getPushMethodFromGrpc(visitor *dmiapi.DeviceProperty) (string, error) {
	if visitor.PushMethod != nil && visitor.PushMethod.Http != nil {
		return common.PushMethodHTTP, nil
	}
	if visitor.PushMethod != nil && visitor.PushMethod.Mqtt != nil {
		return common.PushMethodMQTT, nil
	}
	return "", errors.New("can not parse publish method")
}

func getDBMethodFromGrpc(visitor *dmiapi.DeviceProperty) (string, error) {
	if visitor.PushMethod.DBMethod.Influxdb2 != nil {
		return "influx", nil
	} else if visitor.PushMethod.DBMethod.Redis != nil {
		return "redis", nil
	} else if visitor.PushMethod.DBMethod.Tdengine != nil {
		return "tdengine", nil
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
	if len(device.Spec.Properties) == 0 {
		return nil
	}
	res := make([]common.Twin, 0, len(device.Spec.Properties))
	for _, property := range device.Spec.Properties {
		cur := common.Twin{
			PropertyName: property.Name,
			ObservedDesired: common.TwinProperty{
				Value: property.Desired.Value,
				Metadata: common.Metadata{
					Timestamp: property.Desired.Metadata["timestamp"],
					Type:      property.Desired.Metadata["type"],
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
		var pushMethod []byte
		var pushMethodName string
		if pptv.PushMethod != nil && pptv.PushMethod.DBMethod != nil {
			dbMethodName, err = getDBMethodFromGrpc(pptv)
			if err != nil {
				klog.Errorf("get DBMethod err: %+v", err)
				return nil
			}
			switch dbMethodName {
			case "influx":
				clientconfig, err := json.Marshal(pptv.PushMethod.DBMethod.Influxdb2.Influxdb2ClientConfig)
				if err != nil {
					klog.Errorf("influx client config err: %+v", err)
					return nil
				}
				dataconfig, err := json.Marshal(pptv.PushMethod.DBMethod.Influxdb2.Influxdb2DataConfig)
				if err != nil {
					klog.Errorf("influx data config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					Influxdb2ClientConfig: clientconfig,
					Influxdb2DataConfig:   dataconfig,
				}
			case "redis":
				clientConfig, err := json.Marshal(pptv.PushMethod.DBMethod.Redis.RedisClientConfig)
				if err != nil {
					klog.Errorf("redis config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					RedisClientConfig: clientConfig,
				}
			case "tdengine":
				clientConfig, err := json.Marshal(pptv.PushMethod.DBMethod.Tdengine.TdEngineClientConfig)
				if err != nil {
					klog.Errorf("tdengine config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					TDEngineClientConfig: clientConfig,
				}
			}
		}

		// get pushMethod filed by grpc device instance
		pushMethodName, err = getPushMethodFromGrpc(pptv)
		if err != nil {
			klog.Errorf("err: %+v", err)
			return nil
		}
		switch pushMethodName {
		case common.PushMethodHTTP:
			pushMethod, err = json.Marshal(pptv.PushMethod.Http)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		case common.PushMethodMQTT:
			pushMethod, err = json.Marshal(pptv.PushMethod.Mqtt)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		}

		// get the final Properties
		cur := common.DeviceProperty{
			Name:          pptv.GetName(),
			PropertyName:  pptv.GetName(),
			ModelName:     device.Spec.DeviceModelReference,
			CollectCycle:  pptv.GetCollectCycle(),
			ReportCycle:   pptv.GetReportCycle(),
			ReportToCloud: pptv.GetReportToCloud(),
			Protocol:      protocolName,
			Visitors:      visitorConfig,
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
			DataType:    property.GetType(),
			AccessMode:  property.GetAccessMode(),
			Minimum:     property.GetMinimum(),
			Maximum:     property.GetMaximum(),
			Unit:        property.GetUnit(),
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
		Model:        device.GetSpec().GetDeviceModelReference(),
		Twins:        buildTwinsFromGrpc(device),
		Properties:   buildPropertiesFromGrpc(device),
	}
	// copy Properties to twin
	propertiesMap := make(map[string]common.DeviceProperty)
	for i := range instance.Properties {
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
	for i := range instance.Twins {
		if v, ok := propertiesMap[instance.Twins[i].PropertyName]; ok {
			instance.Twins[i].Property = &v
		}
	}
	klog.V(2).Infof("final instance data from grpc = %v", instance)
	return instance, nil
}
