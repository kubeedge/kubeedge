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
package envconfigsource

import (
	"github.com/go-chassis/go-archaius/core"
	"os"
	"testing"
)

type TestDynamicConfigHandler struct{}

func (t *TestDynamicConfigHandler) OnEvent(e *core.Event) {}

func populatEnvConfiguration() {

	os.Setenv("testenvkey1", "envkey1")
	os.Setenv("testenvkey2", "envkey2")
	os.Setenv("testenvkey3", "a=b=c")
}

func TestEnvConfigurationSource(t *testing.T) {

	populatEnvConfiguration()
	envsource := NewEnvConfigurationSource()

	t.Log("Test envconfigurationsource.go")

	t.Log("verifying envsource configurations by GetConfigurations method")
	_, err := envsource.GetConfigurations()
	if err != nil {
		t.Error("Failed to get configurations from envsource")
	}

	t.Log("verifying envsource configurations by GetConfigurationByKey method")
	configkey1, err := envsource.GetConfigurationByKey("testenvkey1")
	if err != nil {
		t.Error("Failed to get existing configuration key value pair from envsource")
	}

	//Accessing the envsource config key
	configkey2, err := envsource.GetConfigurationByKey("testenvkey3")
	if err != nil {
		t.Error("Failed to get existing configuration key value pair from envsource")
	}

	if configkey1 != "envkey1" && configkey2 != "a=b=c" {
		t.Error("envsource key value pairs is mismatched")
	}

	t.Log("Verifying the envsource priority")
	envsorcepriority := envsource.GetPriority()
	if envsorcepriority != 3 {
		t.Error("envsource priority is mismatched")
	}

	t.Log("Verifying the envsource name")
	envsourcename := envsource.GetSourceName()
	if envsourcename != "EnvironmentSource" {
		t.Error("envsource name is mismatched")
	}

	dynHandler := new(TestDynamicConfigHandler)
	envdynamicconfig := envsource.DynamicConfigHandler(dynHandler)
	if envdynamicconfig != nil {
		t.Error("Failed to get envsource dynamic configuration")
	}

	t.Log("envsource cleanup")
	envsourcecleanup := envsource.Cleanup()
	if envsourcecleanup != nil {
		t.Error("envsource cleanup is Failed")
	}

	t.Log("verifying envsource configurations after cleanup")
	configkey1, _ = envsource.GetConfigurationByKey("testenvkey1")
	configkey2, _ = envsource.GetConfigurationByKey("testenvkey2")
	data, err := envsource.GetConfigurationByKeyAndDimensionInfo("data@default#0.1", "hello")
	if data != nil || err != nil {
		t.Error("Failed to get configuration by dimension info and key")
	}
	if configkey1 != nil && configkey2 != nil {
		t.Error("envsource cleanup is Failed")
	}
}
