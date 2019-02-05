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

package containers

import (
	"os"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/pkg/edged/apis/runtime/cri"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/kubernetes/pkg/kubelet/gpu"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
)

// edged socket is the path of the edged registry socket
var kubePath = "/var/lib/kubelet"
var pluginPath = kubePath + "/device-plugins/"

func tearDownTestNewContainerManager(t *testing.T) {
	if _, serr := os.Stat(pluginPath); serr == nil {
		merr := os.RemoveAll(kubePath)
		if merr != nil {
			t.Fatalf("removeAll %s directory failed: %s", kubePath, merr.Error())
			os.Exit(1)
		}
	} else {
		t.Fatalf("stat of %s path failed: %s", pluginPath, serr.Error())
		os.Exit(1)
	}
}

func TestNewContainerManager(t *testing.T) {

	cases := []struct {
		// name is name of the testcase
		name string
		// testcase Identifier
		testID int
		// container runtime service
		runtimeService cri.RuntimeService
		// container probe manager
		livenessManager proberesults.Manager
		// containerBackOff for flow control
		containerBackOff *flowcontrol.Backoff
		// Enable device plugin it is either true or false
		devicePluginEnabled bool
		// GPU manager instance
		gpuManager gpu.GPUManager
		// interface name
		interfaceName string
		// returnErr is second return of mock interface ormerMock which is also expected error
		returnErr string
	}{{
		name:                "NewContainerManager - SuccessCase with device plugin enabled",
		testID:              1,
		runtimeService:      nil,
		livenessManager:     proberesults.NewManager(),
		containerBackOff:    flowcontrol.NewBackOff(10*time.Second, 300*time.Second),
		devicePluginEnabled: true,
		gpuManager:          gpu.NewGPUManagerStub(),
		interfaceName:       "docker0",
		returnErr:           "",
	}, {
		name:                "NewContainerManager - SuccessCase with device plugin disabled",
		testID:              2,
		runtimeService:      nil,
		livenessManager:     proberesults.NewManager(),
		containerBackOff:    flowcontrol.NewBackOff(10*time.Second, 300*time.Second),
		devicePluginEnabled: false,
		gpuManager:          gpu.NewGPUManagerStub(),
		interfaceName:       "docker0",
		returnErr:           "",
	},
	// There aren't any kind of failure scenario, unless system calls are mocked to stat and create directory.
	// It would be an un-necessary effort to create mocks for those functions.
	// Hence for this function failure scenario is not covered.
	}

	// run the test cases
	for _, test := range cases {
		result, err := NewContainerManager(test.runtimeService, test.livenessManager, test.containerBackOff, test.devicePluginEnabled, test.gpuManager, test.interfaceName)
		assert.NotEmptyf(t, result, "container Manager object creation for testID %d failed", test.testID)
		assert.Emptyf(t, err, "container Manager object creation for testID %d failed", test.testID)
		if err != nil {
			t.Fatalf("test failed due to error %s", err.Error())
		}

		if test.devicePluginEnabled == true {
			// While creating newContainerManager instance, it creates directory structure of pluginapi edgeD Socket path("/var/lib/xxxx/device-plugins/") if it didn't exist.
			// Checking this paths creation as well for UT verification, before teardown (cleanup of filesystem from UT residuals)
			_, serr := os.Stat(pluginPath)
			assert.Emptyf(t, serr, "pluginapi edgeD Socket path creation for testID %d failed", test.testID)

			tearDownTestNewContainerManager(t)
		}
	}
}
