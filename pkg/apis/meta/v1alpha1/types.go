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
type GroupName string

// Available modules for CloudCore
const (
	ModuleNameEdgeController   ModuleName = "edgecontroller"
	ModuleNameDeviceController ModuleName = "devicecontroller"
	ModuleNameCloudHub         ModuleName = "cloudhub"
)

// Available modules for EdgeCore
const (
	ModuleNameEventBus   ModuleName = "eventbus"
	ModuleNameServiceBus ModuleName = "servicebus"
	// TODO @kadisi change websocket to edgehub
	ModuleNameEdgeHub     ModuleName = "websocket"
	ModuleNameMetaManager ModuleName = "metaManager"
	ModuleNameEdged       ModuleName = "edged"
	ModuleNameTwin        ModuleName = "twin"
	ModuleNameDBTest      ModuleName = "dbTest"
	ModuleNameEdgeMesh    ModuleName = "edgemesh"
)

// Available modules group
const (
	GroupNameHub            GroupName = "hub"
	GroupNameEdgeController GroupName = "edgecontroller"
	GroupNameBus            GroupName = "bus"
	GroupNameTwin           GroupName = "twin"
	GroupNameMeta           GroupName = "meta"
	GroupNameEdged          GroupName = "edged"
	GroupNameUser           GroupName = "user"
	GroupNameMesh           GroupName = "mesh"
)
