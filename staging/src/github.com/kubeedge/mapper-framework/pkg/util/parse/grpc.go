package parse

import (
	"encoding/json"

	"k8s.io/klog/v2"

	dmiapi "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

type TwinResultResponse struct {
	PropertyName string `json:"property_name"`
	Payload      []byte `json:"payload"`
}

func getProtocolNameFromGrpc(device *dmiapi.Device) (string, error) {
	return device.Spec.Protocol.ProtocolName, nil
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

		// get the whole pushmethod filed by grpc device instance
		var dbMethodName string
		var dbconfig common.DBConfig
		var pushMethod []byte
		var pushMethodName string
		if pptv.PushMethod != nil && pptv.PushMethod.DbMethod != nil {
			//parse dbmethod filed
			switch {
			case pptv.PushMethod.DbMethod.Influxdb2 != nil:
				dbMethodName = "influx"
				clientconfig, err := json.Marshal(pptv.PushMethod.DbMethod.Influxdb2.Influxdb2ClientConfig)
				if err != nil {
					klog.Errorf("influx client config err: %+v", err)
					return nil
				}
				dataconfig, err := json.Marshal(pptv.PushMethod.DbMethod.Influxdb2.Influxdb2DataConfig)
				if err != nil {
					klog.Errorf("influx data config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					Influxdb2ClientConfig: clientconfig,
					Influxdb2DataConfig:   dataconfig,
				}
			case pptv.PushMethod.DbMethod.Redis != nil:
				dbMethodName = "redis"
				clientConfig, err := json.Marshal(pptv.PushMethod.DbMethod.Redis.RedisClientConfig)
				if err != nil {
					klog.Errorf("redis config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					RedisClientConfig: clientConfig,
				}
			case pptv.PushMethod.DbMethod.Tdengine != nil:
				dbMethodName = "tdengine"
				clientConfig, err := json.Marshal(pptv.PushMethod.DbMethod.Tdengine.TdEngineClientConfig)
				if err != nil {
					klog.Errorf("tdengine config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					TDEngineClientConfig: clientConfig,
				}
			case pptv.PushMethod.DbMethod.Mysql != nil:
				dbMethodName = "mysql"
				clientConfig, err := json.Marshal(pptv.PushMethod.DbMethod.Mysql.MysqlClientConfig)
				if err != nil {
					klog.Errorf("mysql config err: %+v", err)
					return nil
				}
				dbconfig = common.DBConfig{
					MySQLClientConfig: clientConfig,
				}
			default:
				klog.Errorf("get DBMethod err: Unsupported database type")
			}
		}
		if pptv.PushMethod != nil {
			//parse pushmethod filed
			switch {
			case pptv.PushMethod.Http != nil:
				pushMethodName = common.PushMethodHTTP
				pushMethod, err = json.Marshal(pptv.PushMethod.Http)
				if err != nil {
					klog.Errorf("err: %+v", err)
					return nil
				}
			case pptv.PushMethod.Mqtt != nil:
				pushMethodName = common.PushMethodMQTT
				pushMethod, err = json.Marshal(pptv.PushMethod.Mqtt)
				if err != nil {
					klog.Errorf("err: %+v", err)
					return nil
				}
			case pptv.PushMethod.Otel != nil:
				dbMethodName = common.PushMethodOTEL
				pushMethod, err = json.Marshal(pptv.PushMethod.Otel)
				if err != nil {
					klog.Errorf("err: %+v", err)
					return nil
				}
			default:
				klog.Errorf("get PushMethod err: Unsupported pushmethod type")
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

// buildMethodsFromGrpc parse device method from grpc
func buildMethodsFromGrpc(device *dmiapi.Device) []common.DeviceMethod {
	if len(device.Spec.Methods) == 0 {
		return nil
	}
	res := make([]common.DeviceMethod, 0, len(device.Spec.Properties))
	klog.V(3).Info("Start converting devicemethod information from grpc")
	for _, method := range device.Spec.Methods {
		// Convert device method field
		cur := common.DeviceMethod{
			Name:          method.GetName(),
			Description:   method.GetDescription(),
			PropertyNames: method.GetPropertyNames(),
		}
		res = append(res, cur)
	}

	return res
}

func GetDeviceModelFromGrpc(model *dmiapi.DeviceModel) common.DeviceModel {
	cur := common.DeviceModel{
		ID:        GetResourceID(model.GetNamespace(), model.GetName()),
		Name:      model.GetName(),
		Namespace: model.GetNamespace(),
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

func GetDeviceFromGrpc(device *dmiapi.Device, commonModel *common.DeviceModel) (*common.DeviceInstance, error) {
	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return nil, err
	}
	instance := &common.DeviceInstance{
		ID:           GetResourceID(device.GetNamespace(), device.GetName()),
		Name:         device.GetName(),
		Namespace:    device.GetNamespace(),
		ProtocolName: protocolName + "-" + device.GetName(),
		Model:        device.GetSpec().GetDeviceModelReference(),
		Twins:        buildTwinsFromGrpc(device),
		Properties:   buildPropertiesFromGrpc(device),
		Methods:      buildMethodsFromGrpc(device),
		Status: common.DeviceStatus{
			ReportToCloud: device.Status.GetReportToCloud(),
			ReportCycle:   device.Status.GetReportCycle(),
		},
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

// GetResourceID return resource ID
func GetResourceID(namespace, name string) string {
	return namespace + "/" + name
}
