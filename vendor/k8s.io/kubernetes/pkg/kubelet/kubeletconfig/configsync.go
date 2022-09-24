/*
Copyright 2017 The Kubernetes Authors.

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

package kubeletconfig

const (
	// KubeletConfigChangedEventReason identifies an event as a change of Kubelet configuration
	KubeletConfigChangedEventReason = "KubeletConfigChanged"
	// LocalEventMessage is sent when the Kubelet restarts to use local config
	LocalEventMessage = "Kubelet restarting to use local config"
	// RemoteEventMessageFmt is sent when the Kubelet restarts to use a remote config
	RemoteEventMessageFmt = "Kubelet restarting to use %s, UID: %s, ResourceVersion: %s, KubeletConfigKey: %s"
)
