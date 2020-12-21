/*
Copyright 2019 The KubeEdge Authors.

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

package configuration

import (
	"encoding/json"
	"io/ioutil"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

//ReadFromConfigMap is used to load the information from the configmaps that are provided from the cloud
func ReadFromConfigMap(deviceProfile *types.DeviceProfile) error {
	jsonFile, err := ioutil.ReadFile(ConfigMapPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonFile, deviceProfile)
	if err != nil {
		return err
	}
	return nil
}
