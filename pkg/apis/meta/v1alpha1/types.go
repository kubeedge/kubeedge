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

package v1alpha1

type ModuleName string

const (
	// Available modules for CloudCore
	ModuleNameEdgeController   ModuleName = "edgecontroller"
	ModuleNameDeviceController ModuleName = "devicecontroller"
	ModuleNameCloudHub         ModuleName = "cloudhub"

	// Available modules for EdgeCore
	ModuleNameEventBus    ModuleName = "eventbus"
	ModuleNameServiceBus  ModuleName = "servicebus"
	ModuleNameWebsocket   ModuleName = "websocket"
	ModuleNameMetaManager ModuleName = "metaManager"
	ModuleNameEdged       ModuleName = "edged"
	ModuleNameTwin        ModuleName = "twin"
	ModuleNameDBTest      ModuleName = "dbTest"
	ModuleNameEdgeMesh    ModuleName = "edgemesh"
)

type Modules struct {
	//Enabled defineds the enabled modules
	Enabled []ModuleName `json:"enabled,omitempty"`
}

type TypeMeta struct {
	// Kind is a string value representing the REST resource this object represents.
	// +optional
	Kind string `json:"kind,omitempty"`

	// APIVersion defines the versioned schema of this representation of an object.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}
