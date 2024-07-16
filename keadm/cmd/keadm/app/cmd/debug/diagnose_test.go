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

package debug

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewDiagnose(t *testing.T) {
	assert := assert.New(t)
	cmd := NewDiagnose()

	assert.NotNil(cmd)
	assert.Equal("diagnose", cmd.Use)
	assert.Equal(edgeDiagnoseShortDescription, cmd.Short)
	assert.Equal(edgeDiagnoseLongDescription, cmd.Long)
	assert.Equal(edgeDiagnoseExample, cmd.Example)

	subcommands := cmd.Commands()
	assert.NotNil(subcommands)
}

func TestNewSubDiagnose(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		use               string
		expectedDefValue  map[string]string
		expectedShorthand map[string]string
		expectedUsage     map[string]string
	}{
		{
			use: common.ArgDiagnoseNode,
			expectedDefValue: map[string]string{
				common.EdgecoreConfig: common.EdgecoreConfigPath,
			},
			expectedShorthand: map[string]string{
				common.EdgecoreConfig: "c",
			},
			expectedUsage: map[string]string{
				common.EdgecoreConfig: fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath),
			},
		},
		{
			use: common.ArgDiagnosePod,
			expectedDefValue: map[string]string{
				"namespace": "default",
			},
			expectedShorthand: map[string]string{
				"namespace": "n",
			},
			expectedUsage: map[string]string{
				"namespace": "specify namespace",
			},
		},
		{
			use: common.ArgDiagnoseInstall,
			expectedDefValue: map[string]string{
				"dns-ip":           "",
				"domain":           "",
				"ip":               "",
				"cloud-hub-server": "",
			},
			expectedShorthand: map[string]string{
				"dns-ip":           "D",
				"domain":           "d",
				"ip":               "i",
				"cloud-hub-server": "s",
			},
			expectedUsage: map[string]string{
				"dns-ip":           "specify test dns server ip",
				"domain":           "specify test domain",
				"ip":               "specify test ip",
				"cloud-hub-server": "specify cloudhub server",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.use, func(t *testing.T) {
			diagnoseObj := Diagnose{
				Use:  test.use,
				Desc: fmt.Sprintf("Diagnose %s", test.use),
			}
			cmd := NewSubDiagnose(diagnoseObj)

			assert.NotNil(cmd)
			assert.Equal(diagnoseObj.Use, cmd.Use)
			assert.Equal(diagnoseObj.Desc, cmd.Short)

			flags := cmd.Flags()
			assert.NotNil(flags)

			for flagName, expectedDefValue := range test.expectedDefValue {
				t.Run(flagName, func(t *testing.T) {
					flag := flags.Lookup(flagName)
					assert.NotNil(flag)

					assert.Equal(expectedDefValue, flag.DefValue)
					assert.Equal(test.expectedShorthand[flagName], flag.Shorthand)
					assert.Equal(test.expectedUsage[flagName], flag.Usage)
				})
			}
		})
	}
}

func TestNewDiagnoseOptions(t *testing.T) {
	assert := assert.New(t)

	do := NewDiagnoseOptions()
	assert.NotNil(do)

	assert.Equal("default", do.Namespace)
	assert.Equal(common.EdgecoreConfigPath, do.Config)
	assert.Equal("", do.CheckOptions.IP)
	assert.Equal(3, do.CheckOptions.Timeout)
}
