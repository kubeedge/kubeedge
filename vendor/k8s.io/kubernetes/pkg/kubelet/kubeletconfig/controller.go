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

import (
	"k8s.io/client-go/tools/cache"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"
	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/checkpoint/store"
	"k8s.io/kubernetes/pkg/kubelet/kubeletconfig/status"
)

// TransformFunc edits the KubeletConfiguration in-place, and returns an
// error if any of the transformations failed.
type TransformFunc func(kc *kubeletconfig.KubeletConfiguration) error

// Controller manages syncing dynamic Kubelet configurations
// For more information, see the proposal: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/dynamic-kubelet-configuration.md
type Controller struct {
	// transform applies an arbitrary transformation to config after loading, and before validation.
	// This can be used, for example, to include config from flags before the controller's validation step.
	// If transform returns an error, loadConfig will fail, and an InternalError will be reported.
	// Be wary if using this function as an extension point, in most cases the controller should
	// probably just be natively extended to do what you need. Injecting flag precedence transformations
	// is something of an exception because the caller of this controller (cmd/) is aware of flags, but this
	// controller's tree (pkg/) is not.
	transform TransformFunc

	// pendingConfigSource; write to this channel to indicate that the config source needs to be synced from the API server
	pendingConfigSource chan bool

	// configStatus manages the status we report on the Node object
	configStatus status.NodeConfigStatus

	// nodeInformer is the informer that watches the Node object
	nodeInformer cache.SharedInformer

	// remoteConfigSourceInformer is the informer that watches the assigned config source
	remoteConfigSourceInformer cache.SharedInformer

	// checkpointStore persists config source checkpoints to a storage layer
	checkpointStore store.Store
}
