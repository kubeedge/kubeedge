/*
Copyright 2024 The KubeEdge Authors.

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

package cloudstream

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
)

func TestNewCloudStream(t *testing.T) {
	assert := assert.New(t)
	enable := true
	tunnelPort := 8000

	cs := newCloudStream(true, 8000)

	assert.Equal(cs.enable, enable)
	assert.Equal(cs.tunnelPort, tunnelPort)
}

func TestRegister(t *testing.T) {
	assert := assert.New(t)

	controller := &v1alpha1.CloudStream{
		Enable:                  true,
		TLSTunnelCAFile:         "/path/to/ca/file",
		TLSTunnelCertFile:       "/path/to/cert/file",
		TLSTunnelPrivateKeyFile: "/path/to/key/file",
	}
	commonConfig := &v1alpha1.CommonConfig{
		TunnelPort: 8000,
	}
	config.InitConfigure(controller)

	Register(controller, commonConfig)

	Coremodules := core.GetModules()
	mod, exists := Coremodules[modules.CloudStreamModuleName]
	assert.True(exists)
	assert.NotNil(mod)
	assert.IsType(&cloudStream{}, mod.GetModule())

	cs, ok := mod.GetModule().(*cloudStream)
	assert.True(ok)
	assert.Equal(cs.enable, controller.Enable)
	assert.Equal(cs.tunnelPort, commonConfig.TunnelPort)
}

func TestName(t *testing.T) {
	assert := assert.New(t)
	stdResult := "cloudStream"

	cs := &cloudStream{
		enable:     true,
		tunnelPort: 8000,
	}
	name := cs.Name()
	assert.Equal(name, stdResult)
}

func TestGroup(t *testing.T) {
	assert := assert.New(t)
	stdResult := "cloudStream"

	cs := &cloudStream{
		enable:     true,
		tunnelPort: 8000,
	}
	group := cs.Group()
	assert.Equal(group, stdResult)
}

func TestEnable(t *testing.T) {
	assert := assert.New(t)

	cs := &cloudStream{
		enable:     true,
		tunnelPort: 8000,
	}
	stdResult := cs.Enable()
	assert.Equal(cs.enable, stdResult)
}
