// +build !linux,!windows,!dockerless

/*
Copyright 2015 The Kubernetes Authors.

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

package dockershim

import (
	"fmt"

	"github.com/blang/semver"
	dockertypes "github.com/docker/docker/api/types"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
)

// DefaultMemorySwap always returns -1 for no memory swap in a sandbox
func DefaultMemorySwap() int64 {
	return -1
}

func (ds *dockerService) getSecurityOpts(seccompProfile string, separator rune) ([]string, error) {
	klog.InfoS("getSecurityOpts is unsupported in this build")
	return nil, nil
}

func (ds *dockerService) getSandBoxSecurityOpts(separator rune) []string {
	klog.InfoS("getSandBoxSecurityOpts is unsupported in this build")
	return nil
}

func (ds *dockerService) updateCreateConfig(
	createConfig *dockertypes.ContainerCreateConfig,
	config *runtimeapi.ContainerConfig,
	sandboxConfig *runtimeapi.PodSandboxConfig,
	podSandboxID string, securityOptSep rune, apiVersion *semver.Version) error {
	klog.InfoS("updateCreateConfig is unsupported in this build")
	return nil
}

func (ds *dockerService) determinePodIPBySandboxID(uid string) []string {
	klog.InfoS("determinePodIPBySandboxID is unsupported in this build")
	return nil
}

func getNetworkNamespace(c *dockertypes.ContainerJSON) (string, error) {
	return "", fmt.Errorf("unsupported platform")
}
