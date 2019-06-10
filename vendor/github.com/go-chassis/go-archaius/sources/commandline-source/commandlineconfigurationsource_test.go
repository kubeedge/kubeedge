/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
* Created by on 2017/6/22.
 */
package commandlinesource

import (
	"github.com/go-chassis/go-archaius/core"
	"os"
	"testing"
)

type TestDynamicConfigHandler struct{}

func (t *TestDynamicConfigHandler) OnEvent(e *core.Event) {}

func populateCmdConfig() {
	os.Args = append(os.Args, "--testcmdkey1=cmdkey1")
	os.Args = append(os.Args, "--testcmdkey2=cmdkey2")
	os.Args = append(os.Args, "-A=cmdkey3")
}

func TestCommandLineConfigurationSource(t *testing.T) {

	populateCmdConfig()
	cmdsource := NewCommandlineConfigSource()

	t.Log("Test commandlineconfigurationsource.go")

	t.Log("verifying command line configurations by GetConfigurations method")
	_, err := cmdsource.GetConfigurations()
	if err != nil {
		t.Error("Failed to get existing configuration key value pair from cmdlinesource")
	}

	t.Log("verifying command line configurations by GetConfigurationByKey method")
	configkey1, err := cmdsource.GetConfigurationByKey("testcmdkey1")
	if err != nil {
		t.Error("Failed to get existing configuration key value pair from cmdlinesource")
	}

	//Accessing the cmdline config key
	configkey2, err := cmdsource.GetConfigurationByKey("A")
	if err != nil {
		t.Error("Failed to get existing configuration key value pair from cmdlinesource")
	}

	if configkey1 != "cmdkey1" && configkey2 != "cmdkey3" {
		t.Error("cmdlinesource configuration key value pairs is mismatched")
	}

	t.Log("Verifying the cmdlinesource priority")
	cmdpriority := cmdsource.GetPriority()
	if cmdpriority != 2 {
		t.Error("commandlinesource priority is mismatched")
	}

	t.Log("Verifying the cmdlinesource name")
	cmdsourcename := cmdsource.GetSourceName()
	if cmdsourcename != "CommandlineSource" {
		t.Error("commandlinesource name is mismatched")
	}

	dynHandler := new(TestDynamicConfigHandler)
	cmddynamicconfig := cmdsource.DynamicConfigHandler(dynHandler)
	if cmddynamicconfig != nil {
		t.Error("Failed to get commandlinesource dynamic configuration")
	}

	data, err := cmdsource.GetConfigurationByKeyAndDimensionInfo("data@default#0.1", "hello")
	if data != nil || err != nil {
		t.Error("Failed to get configuration by dimension info and key")
	}

	t.Log("cmdlinesource cleanup")
	cmdcleanup := cmdsource.Cleanup()
	if cmdcleanup != nil {
		t.Error("commandlinesource cleanup is Failed")
	}

	t.Log("verifying cmdline configurations after cleanup")
	configkey1, _ = cmdsource.GetConfigurationByKey("testcmdkey1")
	configkey2, _ = cmdsource.GetConfigurationByKey("testcmdkey2")
	if configkey1 != nil && configkey2 != nil {
		t.Error("commandlinesource cleanup is Failed")
	}
}
