/*
Copyright 2019 The Kubeedge Authors.

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

package util

//MQTTInstTool embedes Common struct and It implements ToolsInstaller interface
type MQTTInstTool struct {
	Common
}

//InstallTools sets the OS interface, it simply installs the said version
func (m *MQTTInstTool) InstallTools() error {
	m.SetOSInterface(GetOSInterface())
	err := m.InstallMQTT()
	if err != nil {
		return err
	}
	return nil
}

//TearDown shoud uninstall MQTT, but it is not required either for cloud or edge node.
//It is defined so that MQTTInstTool implements ToolsInstaller interface
func (m *MQTTInstTool) TearDown() error {
	return nil
}
