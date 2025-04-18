package parse

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/fatih/structs"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/mapper-generator/pkg/common"
	dmiapi "github.com/kubeedge/mapper-generator/pkg/temp"
)

type TwinResultResponse struct {
	PropertyName string `json:"property_name"`
	Payload      []byte `json:"payload"`
}

func getProtocolNameFromGrpc(device *dmiapi.Device) (string, error) {
	if device.Spec.Protocol.Modbus != nil {
		return constants.Modbus, nil
	}
	if device.Spec.Protocol.Opcua != nil {
		return constants.OPCUA, nil
	}
	if device.Spec.Protocol.Bluetooth != nil {
		return constants.Bluetooth, nil
	}
	if device.Spec.Protocol.CustomizedProtocol != nil {
		return constants.CustomizedProtocol, nil
	}
	return "", errors.New("can not parse device protocol")
}

func getPushMethodFromGrpc(visitor *dmiapi.DevicePropertyVisitor) (string, error) {
	// TODO add more push method
	if visitor.PushMethod.Http != nil {
		return "http", nil
	}
	if visitor.PushMethod.Mqtt != nil {
		return "mqtt", nil
	}
	if visitor.PushMethod.CustomizedProtocol != nil {
		return "customizedPushMethod", nil
	}
	return "", errors.New("can not parse publish method")
}

func BuildProtocolFromGrpc(device *dmiapi.Device) (common.Protocol, error) {
	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return common.Protocol{}, err
	}
	var protocolCommonConfig []byte

	if device.Spec.Protocol.Common.CustomizedValues != nil {
		commonConfig := make(map[string]interface{})
		recvAdapter := make(map[string]interface{})
		for k, v := range device.Spec.Protocol.Common.CustomizedValues.Data {
			value, err := common.DecodeAnyValue(v)
			if err != nil {
				continue
			}
			recvAdapter[k] = value
		}
		if device.Spec.Protocol.Common.Com != nil {
			commonConfig["com"] = structs.Map(device.Spec.Protocol.Common.Com)
		}
		if device.Spec.Protocol.Common.Tcp != nil {
			commonConfig["tcp"] = structs.Map(device.Spec.Protocol.Common.Tcp)
		}
		commonConfig["commType"] = device.Spec.Protocol.Common.CommType
		commonConfig["reconnTimeout"] = device.Spec.Protocol.Common.ReconnTimeout
		commonConfig["reconnRetryTimes"] = device.Spec.Protocol.Common.ReconnRetryTimes
		commonConfig["collectTimeout"] = device.Spec.Protocol.Common.CollectTimeout
		commonConfig["collectRetryTimes"] = device.Spec.Protocol.Common.CollectRetryTimes
		commonConfig["collectType"] = device.Spec.Protocol.Common.CollectType
		commonConfig["customizedValues"] = recvAdapter
		protocolCommonConfig, err = json.Marshal(commonConfig)
		if err != nil {
			return common.Protocol{}, err
		}
	} else {
		protocolCommonConfig, err = json.Marshal(device.Spec.Protocol.Common)
		if err != nil {
			return common.Protocol{}, err
		}
	}

	var protocolConfig []byte
	switch protocolName {
	case constants.Modbus:
		protocolConfig, err = json.Marshal(device.Spec.Protocol.Modbus)
		if err != nil {
			return common.Protocol{}, err
		}
	case constants.OPCUA:
		protocolConfig, err = json.Marshal(device.Spec.Protocol.Opcua)
		if err != nil {
			return common.Protocol{}, err
		}
	case constants.Bluetooth:
		protocolConfig, err = json.Marshal(device.Spec.Protocol.Bluetooth)
		if err != nil {
			return common.Protocol{}, err
		}
	case constants.CustomizedProtocol:
		customizedProtocol := make(map[string]interface{})
		customizedProtocol["protocolName"] = device.Spec.Protocol.CustomizedProtocol.ProtocolName
		if device.Spec.Protocol.CustomizedProtocol.ConfigData != nil {
			recvAdapter := make(map[string]interface{})
			for k, v := range device.Spec.Protocol.CustomizedProtocol.ConfigData.Data {
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
			return common.Protocol{}, err
		}
	}
	return common.Protocol{
		Name:                 protocolName + "-" + device.Name,
		Protocol:             protocolName,
		ProtocolConfigs:      protocolConfig,
		ProtocolCommonConfig: protocolCommonConfig,
	}, nil
}

func buildTwinsFromGrpc(device *dmiapi.Device) []common.Twin {
	if len(device.Status.Twins) == 0 {
		return nil
	}
	res := make([]common.Twin, 0, len(device.Status.Twins))
	for _, twin := range device.Status.Twins {
		var visitor *dmiapi.DevicePropertyVisitor
		for _, v := range device.Spec.PropertyVisitors {
			if twin.PropertyName == v.PropertyName {
				visitor = v
				break
			}
		}

		protocolName, err := getProtocolNameFromGrpc(device)
		if err != nil {
			klog.Errorf("fail to get protocol name from grpc for device %s with err: %+v", device.Name, err)
			return nil
		}

		var visitorConfig []byte
		switch protocolName {
		case constants.Modbus:
			visitorConfig, err = json.Marshal(visitor.Modbus)
			if err != nil {
				return nil
			}
		case constants.OPCUA:
			visitorConfig, err = json.Marshal(visitor.Opcua)
			if err != nil {
				return nil
			}
		case constants.Bluetooth:
			visitorConfig, err = json.Marshal(visitor.Bluetooth)
			if err != nil {
				return nil
			}
		case constants.CustomizedProtocol:
			customizedProtocol := make(map[string]interface{})
			customizedProtocol["protocolName"] = visitor.CustomizedProtocol.ProtocolName
			if visitor.CustomizedProtocol.ConfigData != nil {
				recvAdapter := make(map[string]interface{})
				for k, v := range visitor.CustomizedProtocol.ConfigData.Data {
					value, err := common.DecodeAnyValue(v)
					if err != nil {
						continue
					}
					recvAdapter[k] = value
				}
				customizedProtocol["configData"] = recvAdapter
			}
			visitorConfig, err = json.Marshal(customizedProtocol)
			if err != nil {
				return nil
			}
		}

		cur := common.Twin{
			PropertyName: twin.PropertyName,
			PVisitor: &common.PropertyVisitor{
				Name:         twin.PropertyName,
				PropertyName: twin.PropertyName,
				ModelName:    device.Spec.DeviceModelReference,
				CollectCycle: visitor.CollectCycle,
				ReportCycle:  visitor.ReportCycle,
				PProperty: common.Property{
					Name:         twin.PropertyName,
					DataType:     "",
					Description:  "",
					AccessMode:   "",
					DefaultValue: nil,
					Minimum:      0,
					Maximum:      0,
					Unit:         "",
				},
				Protocol:      protocolName,
				VisitorConfig: visitorConfig,
			},
			Desired: common.DesiredData{
				Value: twin.Desired.Value,
				Metadatas: common.Metadata{
					Timestamp: twin.Desired.Metadata["timestamp"],
					Type:      twin.Desired.Metadata["type"],
				},
			},
			Reported: common.ReportedData{
				Value: twin.Reported.Value,
				Metadatas: common.Metadata{
					Timestamp: twin.Desired.Metadata["timestamp"],
					Type:      twin.Desired.Metadata["type"],
				},
			},
		}
		res = append(res, cur)
	}
	return res
}

func buildDataFromGrpc(device *dmiapi.Device) common.Data {
	res := common.Data{}
	if len(device.Spec.PropertyVisitors) > 0 {
		res.Properties = make([]common.DataProperty, 0, len(device.Spec.PropertyVisitors))
		for _, property := range device.Spec.PropertyVisitors {
			cur := common.DataProperty{
				Metadatas:    common.DataMetadata{},
				PropertyName: property.PropertyName,
				PVisitor:     nil,
			}
			if property.CustomizedValues != nil && property.CustomizedValues.Data != nil {
				timestamp, ok := property.CustomizedValues.Data["timestamp"]
				if ok {
					t, _ := strconv.ParseInt(string(timestamp.GetValue()), 10, 64)
					cur.Metadatas.Timestamp = t
				}
				tpe, ok := property.CustomizedValues.Data["type"]
				if ok {
					cur.Metadatas.Type = string(tpe.GetValue())
				}
				res.Properties = append(res.Properties, cur)
			}
		}
	}
	return res
}

func buildPropertyVisitorsFromGrpc(device *dmiapi.Device) []common.PropertyVisitor {
	if len(device.Spec.PropertyVisitors) == 0 {
		return nil
	}
	protocolName, err := getProtocolNameFromGrpc(device)
	if err != nil {
		return nil
	}
	res := make([]common.PropertyVisitor, 0, len(device.Spec.PropertyVisitors))
	for _, pptv := range device.Spec.PropertyVisitors {
		var visitorConfig []byte
		switch protocolName {
		case constants.Modbus:
			visitorConfig, err = json.Marshal(pptv.Modbus)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		case constants.OPCUA:
			visitorConfig, err = json.Marshal(pptv.Opcua)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		case constants.Bluetooth:
			visitorConfig, err = json.Marshal(pptv.Bluetooth)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		case constants.CustomizedProtocol:
			recvAdapter := make(map[string]interface{})
			for k, v := range pptv.CustomizedProtocol.ConfigData.Data {
				value, err := common.DecodeAnyValue(v)
				if err != nil {
					continue
				}
				recvAdapter[k] = value
			}
			customizedProtocol := make(map[string]interface{})
			customizedProtocol["protocolName"] = pptv.CustomizedProtocol.ProtocolName
			customizedProtocol["configData"] = recvAdapter
			visitorConfig, err = json.Marshal(customizedProtocol)
			if err != nil {
				klog.Errorf("err: %+v", err)
				return nil
			}
		}

		if pptv.PushMethod == nil {
			cur := common.PropertyVisitor{
				Name:          pptv.PropertyName,
				PropertyName:  pptv.PropertyName,
				ModelName:     device.Spec.DeviceModelReference,
				CollectCycle:  pptv.GetCollectCycle(),
				ReportCycle:   pptv.GetReportCycle(),
				Protocol:      protocolName,
				VisitorConfig: visitorConfig,
			}
			res = append(res, cur)
			continue
		}
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
		case "customizedPushMethod":
			//TODO add customized push method parse
			return nil
		}
		cur := common.PropertyVisitor{
			Name:          pptv.PropertyName,
			PropertyName:  pptv.PropertyName,
			ModelName:     device.Spec.DeviceModelReference,
			CollectCycle:  pptv.GetCollectCycle(),
			ReportCycle:   pptv.GetReportCycle(),
			Protocol:      protocolName,
			VisitorConfig: visitorConfig,
			PushMethod: common.PushMethodConfig{
				MethodName:   pushMethodName,
				MethodConfig: pushMethod,
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
	properties := make([]common.Property, 0, len(model.Spec.Properties))
	for _, property := range model.Spec.Properties {
		p := common.Property{
			Name:        property.GetName(),
			Description: property.GetDescription(),
		}
		if property.Type.GetString_() != nil {
			p.DataType = "string"
			p.AccessMode = property.Type.String_.GetAccessMode()
			p.DefaultValue = property.Type.String_.GetDefaultValue()
		} else if property.Type.GetBytes() != nil {
			p.DataType = "bytes"
			p.AccessMode = property.Type.Bytes.GetAccessMode()
		} else if property.Type.GetBoolean() != nil {
			p.DataType = "boolean"
			p.AccessMode = property.Type.Boolean.GetAccessMode()
			p.DefaultValue = property.Type.Boolean.GetDefaultValue()
		} else if property.Type.GetInt() != nil {
			p.DataType = "int"
			p.AccessMode = property.Type.Int.GetAccessMode()
			p.DefaultValue = property.Type.Int.GetDefaultValue()
			p.Minimum = property.Type.Int.Minimum
			p.Maximum = property.Type.Int.Maximum
			p.Unit = property.Type.Int.Unit
		} else if property.Type.GetDouble() != nil {
			p.DataType = "double"
			p.AccessMode = property.Type.Double.GetAccessMode()
			p.DefaultValue = property.Type.Double.GetDefaultValue()
			p.Minimum = int64(property.Type.Double.Minimum)
			p.Maximum = int64(property.Type.Double.Maximum)
			p.Unit = property.Type.Double.Unit
		} else if property.Type.GetFloat() != nil {
			p.DataType = "float"
			p.AccessMode = property.Type.Float.GetAccessMode()
			p.DefaultValue = property.Type.Float.GetDefaultValue()
			p.Minimum = int64(property.Type.Float.Minimum)
			p.Maximum = int64(property.Type.Float.Maximum)
			p.Unit = property.Type.Float.Unit
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
		ID:               device.GetName(),
		Name:             device.GetName(),
		ProtocolName:     protocolName + "-" + device.GetName(),
		Model:            device.Spec.DeviceModelReference,
		Twins:            buildTwinsFromGrpc(device),
		Datas:            buildDataFromGrpc(device),
		PropertyVisitors: buildPropertyVisitorsFromGrpc(device),
	}
	propertyVisitorMap := make(map[string]common.PropertyVisitor)
	for i := 0; i < len(instance.PropertyVisitors); i++ {
		if commonModel == nil {
			klog.Errorf("commonModel == nil")
			continue
		}

		for _, property := range commonModel.Properties {
			if property.Name == instance.PropertyVisitors[i].PropertyName {
				instance.PropertyVisitors[i].PProperty = property
				break
			}
		}
		propertyVisitorMap[instance.PropertyVisitors[i].PProperty.Name] = instance.PropertyVisitors[i]
	}
	for i := 0; i < len(instance.Twins); i++ {
		if v, ok := propertyVisitorMap[instance.Twins[i].PropertyName]; ok {
			instance.Twins[i].PVisitor = &v
		}
	}
	for i := 0; i < len(instance.Datas.Properties); i++ {
		if v, ok := propertyVisitorMap[instance.Datas.Properties[i].PropertyName]; ok {
			instance.Datas.Properties[i].PVisitor = &v
		}
	}
	return instance, nil
}
