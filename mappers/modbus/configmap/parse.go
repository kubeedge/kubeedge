package configmap

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"k8s.io/klog"
)

func Parse(path string,
	devices map[string]*globals.ModbusDev,
	dms map[string]mappercommon.DeviceModel,
	protocols map[string]mappercommon.Protocol) error {
	jsonFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var deviceProfile mappercommon.DeviceProfile
	err = json.Unmarshal(jsonFile, &deviceProfile)
	if err != nil {
		return err
	}

	klog.Error("profile len:", len(deviceProfile.DeviceInstances))
	for i := 0; i < len(deviceProfile.DeviceInstances); i++ {
		instance := deviceProfile.DeviceInstances[i]
		j := 0
		for j = 0; j < len(deviceProfile.Protocols); j++ {
			if instance.ProtocolName == deviceProfile.Protocols[j].Name {
				instance.PProtocol = deviceProfile.Protocols[j]
				break
			}
		}
		// Protocol not found
		if j == len(deviceProfile.Protocols) {
			err = errors.New("Protocol not found")
			return err
		}

		if instance.PProtocol.Protocol != "modbus" {
			continue
		}

		for k := 0; k < len(instance.PropertyVisitors); k++ {
			modelName := instance.PropertyVisitors[k].ModelName
			propertyName := instance.PropertyVisitors[k].PropertyName
			l := 0
			for l = 0; l < len(deviceProfile.DeviceModels); l++ {
				if modelName == deviceProfile.DeviceModels[l].Name {
					m := 0
					for m = 0; m < len(deviceProfile.DeviceModels[l].Properties); m++ {
						if propertyName == deviceProfile.DeviceModels[l].Properties[m].Name {
							instance.PropertyVisitors[k].PProperty = deviceProfile.DeviceModels[l].Properties[m]
							break
						}
					}

					if m == len(deviceProfile.DeviceModels[l].Properties) {
						err = errors.New("Property not found")
						return err
					}
					break
				}
			}

			if l == len(deviceProfile.DeviceModels) {
				err = errors.New("Device model not found")
				return err
			}
		}

		for k := 0; k < len(instance.Twins); k++ {
			name := instance.Twins[k].PropertyName
			l := 0
			for l = 0; l < len(instance.PropertyVisitors); l++ {
				if name == instance.PropertyVisitors[l].PropertyName {
					instance.Twins[k].PVisitor = &instance.PropertyVisitors[l]
					break
				}
			}

			if l == len(instance.PropertyVisitors) {
				return errors.New("PropertyVisitor not found")
			}
		}
		for k := 0; k < len(instance.Datas.Properties); k++ {
			name := instance.Datas.Properties[k].PropertyName
			l := 0
			for l = 0; l < len(instance.PropertyVisitors); l++ {
				if name == instance.PropertyVisitors[l].PropertyName {
					instance.Datas.Properties[k].PVisitor = &instance.PropertyVisitors[l]
					break
				}
			}

			if l == len(instance.PropertyVisitors) {
				return errors.New("PropertyVisitor not found")
			}
		}

		devices[instance.ID] = new(globals.ModbusDev)
		devices[instance.ID].Instance = instance
		klog.Error("Instance id:", instance.ID)
	}

	for i := 0; i < len(deviceProfile.DeviceModels); i++ {
		dms[deviceProfile.DeviceModels[i].Name] = deviceProfile.DeviceModels[i]
	}

	for i := 0; i < len(deviceProfile.Protocols); i++ {
		protocols[deviceProfile.Protocols[i].Name] = deviceProfile.Protocols[i]
	}
	return nil
}
