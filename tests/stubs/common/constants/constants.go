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

package constants

const (

	// Modules
	CloudHub       = "cloudhub"
	EdgeHub        = "edgehub"
	ControllerStub = "controllerstub"
	HandlerStub    = "handlerstub"

	// Group
	ControllerGroup = "controller"
	HubGroup        = "hub"
	MetaGroup       = "meta"

	ResourceSliceLength       = 5
	ResourceSliceLengthQuery  = 4
	ResourceNodeIndex         = 0
	ResourceNodeIDIndex       = 1
	ResourceNamespaceIndex    = 2
	ResourceResourceTypeIndex = 3
	ResourceResourceNameIndex = 4
	ResourceNode              = "node"

	// Group
	GroupResource    = "resource"
	NamespaceDefault = "default"

	// Pod status

	PodResource = "/pods"

	PodPending   = "Pending"
	PodRunning   = "Running"
	PodSucceeded = "Succeeded"
	PodFailed    = "Failed"
	PodUnknown   = "Unknown"
)
