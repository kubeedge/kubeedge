/*
Copyright 2023 The KubeEdge Authors.

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

package devicecontroller

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/testtools"
)

func init() {
	err := testtools.InitKubeClient()
	if err != nil {
		klog.Errorf("fail to init kubeclient with err: %+v", err)
	}

	dc := &v1alpha1.DeviceController{
		Enable: false,
		Buffer: &v1alpha1.DeviceControllerBuffer{
			UpdateDeviceStatus: constants.DefaultUpdateDeviceStatusBuffer,
			DeviceEvent:        constants.DefaultDeviceEventBuffer,
			DeviceModelEvent:   constants.DefaultDeviceModelEventBuffer,
		},
		Load: &v1alpha1.DeviceControllerLoad{
			UpdateDeviceStatusWorkers: constants.DefaultUpdateDeviceStatusWorkers,
		},
	}

	config.InitConfigure(dc)
}

func TestNewDeviceControllerAndStartIt(t *testing.T) {
	tests := []struct {
		name       string
		controller *DeviceController
		want       *DeviceController
	}{
		{
			name: "New Device controller",
			controller: &DeviceController{
				enable: true,
			},
			want: &DeviceController{
				enable: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testController := newDeviceController(tt.controller.enable)
			if !reflect.DeepEqual(tt.want.enable, testController.Enable()) {
				t.Errorf("TestNewDeviceController() = %v, want %v", (*testController).Enable(), *tt.want)
			}
			testController.Start()
			time.Sleep(2 * time.Second)
			beehiveContext.Cancel()
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		controller *v1alpha1.DeviceController
		want       *v1alpha1.DeviceController
	}{
		{
			name: "Register Device controller",
			controller: &v1alpha1.DeviceController{
				Enable: false,
			},
			want: &v1alpha1.DeviceController{
				Enable: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Register(tt.controller)
			if !reflect.DeepEqual(tt.want.Enable, config.Config.DeviceController.Enable) {
				t.Errorf("TestRegister() = %v, want %v", config.Config.DeviceController, *tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	t.Run("DeviceController.Name()", func(t *testing.T) {
		if got := (&DeviceController{}).Name(); got != modules.DeviceControllerModuleName {
			t.Errorf("DeviceController.Name() returned unexpected result. got = %s, want = DeviceController", got)
		}
	})
}

func TestGroup(t *testing.T) {
	t.Run("DeviceController.Group()", func(t *testing.T) {
		if got := (&DeviceController{}).Group(); got != modules.DeviceControllerModuleGroup {
			t.Errorf("DeviceController.Group() returned unexpected result. got = %s, want = DeviceController", got)
		}
	})
}
