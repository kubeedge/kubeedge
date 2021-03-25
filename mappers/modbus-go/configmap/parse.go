/*
Copyright 2020 The KubeEdge Authors.

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

package configmap

import (
	"encoding/json"
	"io/ioutil"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus-go/globals"
)

const (
	OPCUA              = "opcua"
	Modbus             = "modbus"
	Bluetooth          = "bluetooth"
	CustomizedProtocol = "customized-protocol"
)

func getProtocolName(config v1alpha2.ProtocolConfig) string {
	if config.OpcUA != nil {
		return OPCUA
	} else if config.Modbus != nil {
		return Modbus
	} else if config.Bluetooth != nil {
		return Bluetooth
	} else if config.CustomizedProtocol != nil {
		return CustomizedProtocol
	}
	return ""
}

// Parse parse the configmap.
func Parse(path string,
	devices map[string]*globals.ModbusDev,
	dms map[string]mappercommon.DeviceModel,
	protocols map[string]mappercommon.Protocol) error {
	var deviceProfile mappercommon.DeviceProfile

	jsonFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(jsonFile, &deviceProfile); err != nil {
		return err
	}

	klog.Errorf("device profile is %v", deviceProfile)

	for i := 0; i < len(deviceProfile.DeviceInstances); i++ {
		instance := deviceProfile.DeviceInstances[i]

		devices[instance.Namespace+"/"+instance.Name] = new(globals.ModbusDev)
		devices[instance.Namespace+"/"+instance.Name].Instance = instance
		klog.V(4).Info("Instance: ", instance.Name, instance)
	}

	for i := 0; i < len(deviceProfile.DeviceModels); i++ {
		dms[deviceProfile.DeviceModels[i].Name] = deviceProfile.DeviceModels[i]
	}

	for i := 0; i < len(deviceProfile.Protocols); i++ {
		protocols[deviceProfile.Protocols[i].Name] = deviceProfile.Protocols[i]
	}
	return nil
}
