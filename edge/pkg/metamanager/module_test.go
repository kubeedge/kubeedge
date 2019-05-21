/*
Copyright 2018 The KubeEdge Authors.

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

package metamanager

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	commodule "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// coreContext is beehive context used for communication between modules
var coreContext *context.Context

// metaModule is metamanager implementation of Module interface
var metaModule core.Module

// TestName will initialize CONFIG and register metaManager and test Name
func TestName(t *testing.T) {
	//Load Configurations as go test runs in /tmp
	modules := core.GetModules()
	core.Register(&metaManager{})
	for name, module := range modules {
		if name == MetaManagerModuleName {
			metaModule = module
			break
		}
	}
	t.Run("ModuleRegistration", func(t *testing.T) {
		if metaModule == nil {
			t.Errorf("MetaManager Module not Registered with beehive core")
			return
		}
		if MetaManagerModuleName != metaModule.Name() {
			t.Errorf("Name of module is not correct wanted: %v and got: %v", MetaManagerModuleName, metaModule.Name())
			return
		}
		if commodule.MetaGroup != metaModule.Group() {
			t.Errorf("Group of module is not correct wanted: %v and got: %v", commodule.MetaGroup, metaModule.Group())
		}
	})

}

// TestStart is used for starting metaManager and testing if sync message is sent correctly
func TestStart(t *testing.T) {
	coreContext = context.GetContext(context.MsgCtxTypeChannel)
	modules := core.GetModules()
	for name, module := range modules {
		coreContext.AddModule(name)
		coreContext.AddModuleGroup(name, module.Group())
	}
	dbm.InitDBManager()
	defer dbm.Cleanup()
	go metaModule.Start(coreContext)

	// wait to hit sync interval and receive message
	message, err := coreContext.Receive(MetaManagerModuleName)
	t.Run("TestMessageContent", func(t *testing.T) {
		if err != nil {
			t.Errorf("error while receiving message")
			return
		}
		if (message.GetSource() != MetaManagerModuleName) || (message.GetGroup() != GroupResource) || (message.GetResource() != model.ResourceTypePodStatus) || (message.GetOperation() != OperationMetaSync) {
			t.Errorf("Wrong message received")
		}
	})
}

// TestCleanup is function to test cleanup
func TestCleanup(t *testing.T) {
	metaModule.Cleanup()
	var test model.Message

	// Send message to avoid deadlock if channel deletion has failed after cleanup
	go coreContext.Send(MetaManagerModuleName, test)

	_, err := coreContext.Receive(MetaManagerModuleName)
	t.Run("CheckCleanUp", func(t *testing.T) {
		if err == nil {
			t.Errorf("MetaManager Module still has channel after cleanup")
		}
	})
}
