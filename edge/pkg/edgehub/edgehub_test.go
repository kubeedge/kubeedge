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

package edgehub

// TODO Re-optimize testcase @kadisi
/*

import (
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commodule "github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// coreContext is beehive context used for communication between modules
var coreContext *context.Context

// edgeHubModule is edgeHub implementation of Module interface
var edgeHubModule core.Module

//TestName is function that registers the module and tests whether the correct name of the module is returned
func TestName(t *testing.T) {
	modules := core.GetModules()
	core.Register(&EdgeHub{controller: NewEdgeHubController()})
	for name, module := range modules {
		if name == ModuleNameEdgeHub {
			edgeHubModule = module
			break
		}
	}
	t.Run("ModuleRegistration", func(t *testing.T) {
		if edgeHubModule == nil {
			t.Errorf("EdgeHub Module not Registered with beehive core")
			return
		}
		if ModuleNameEdgeHub != edgeHubModule.Name() {
			t.Errorf("Name of module is not correct wanted: %v and got: %v", ModuleNameEdgeHub, edgeHubModule.Name())
			return
		}
	})
}

//TestGroup is function that registers the module and tests whether the correct group name is returned
func TestGroup(t *testing.T) {
	modules := core.GetModules()
	core.Register(&EdgeHub{controller: NewEdgeHubController()})
	for name, module := range modules {
		if name == ModuleNameEdgeHub {
			edgeHubModule = module
			break
		}
	}
	t.Run("ModuleRegistration", func(t *testing.T) {
		if edgeHubModule == nil {
			t.Errorf("EdgeHub Module not Registered with beehive core")
			return
		}
		if commodule.HubGroup != edgeHubModule.Group() {
			t.Errorf("Group of module is not correct wanted: %v and got: %v", commodule.HubGroup, edgeHubModule.Group())
		}
	})
}

//TestStart is a function to test the start of the edge hub module
func TestStart(t *testing.T) {
	// time to let config be synced again
	time.Sleep(10 * time.Second)
	coreContext = context.GetContext(context.MsgCtxTypeChannel)
	modules := core.GetModules()
	for name, module := range modules {
		coreContext.AddModule(name)
		coreContext.AddModuleGroup(name, module.Group())
	}
	go edgeHubModule.Start(coreContext)
	time.Sleep(2 * time.Second)
}

// TestCleanup is function to test cleanup
func TestCleanup(t *testing.T) {
	edgeHubModule.Cleanup()
	var test model.Message

	// Send message to avoid deadlock if channel deletion has failed after cleanup
	go coreContext.Send(ModuleNameEdgeHub, test)

	_, err := coreContext.Receive(ModuleNameEdgeHub)
	t.Run("CheckCleanUp", func(t *testing.T) {
		if err == nil {
			t.Errorf("Edgehub Module still has channel after cleanup")
		}
	})
}

*/
