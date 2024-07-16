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

func TestNewCheck(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCheck()

	assert.NotNil(cmd)
	assert.Equal("check", cmd.Use)
	assert.Equal(edgeCheckShortDescription, cmd.Short)
	assert.Equal(edgeCheckLongDescription, cmd.Long)
	assert.Equal(edgeCheckExample, cmd.Example)

	for _, v := range common.CheckObjectMap {
		subCmd := NewSubEdgeCheck(CheckObject(v))
		cmd.AddCommand(subCmd)

		assert.NotNil(subCmd)
		assert.Equal(v.Use, subCmd.Use)
		assert.Equal(v.Desc, subCmd.Short)

		flags := subCmd.Flags()
		assert.NotNil(flags)

		switch v.Use {
		case common.ArgCheckAll:
			// Verify domain flag
			flag := flags.Lookup("domain")
			assert.NotNil(flag)
			assert.Equal("www.github.com", flag.DefValue)
			assert.Equal("d", flag.Shorthand)
			assert.Equal("specify test domain", flag.Usage)

			// Verify IP flag
			flag = flags.Lookup("ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("i", flag.Shorthand)
			assert.Equal("specify test ip", flag.Usage)

			// Verify cloud-hub-server flag
			flag = flags.Lookup("cloud-hub-server")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("s", flag.Shorthand)
			assert.Equal("specify cloudhub server", flag.Usage)

			// Verify dns-ip flag
			flag = flags.Lookup("dns-ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("D", flag.Shorthand)
			assert.Equal("specify test dns ip", flag.Usage)

			// Verify config flag
			flag = flags.Lookup("config")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("c", flag.Shorthand)
			expectedUsage := fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath)
			assert.Equal(expectedUsage, flag.Usage)

		case common.ArgCheckDNS:
			// Verify domain flag
			flag := flags.Lookup("domain")
			assert.NotNil(flag)
			assert.Equal("www.github.com", flag.DefValue)
			assert.Equal("d", flag.Shorthand)
			assert.Equal("specify test domain", flag.Usage)

			// Verify dns-ip flag
			flag = flags.Lookup("dns-ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("D", flag.Shorthand)
			assert.Equal("specify test dns ip", flag.Usage)

		case common.ArgCheckNetwork:
			// Verify IP flag
			flag := flags.Lookup("ip")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("i", flag.Shorthand)
			assert.Equal("specify test ip", flag.Usage)

			// Verify cloud-hub-server flag
			flag = flags.Lookup("cloud-hub-server")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("s", flag.Shorthand)
			assert.Equal("specify cloudhub server", flag.Usage)

			// Verify config flag
			flag = flags.Lookup("config")
			assert.NotNil(flag)
			assert.Equal("", flag.DefValue)
			assert.Equal("c", flag.Shorthand)
			expectedUsage := fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath)
			assert.Equal(expectedUsage, flag.Usage)
		}
	}
}

func TestNewSubEdgeCheck(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		use               string
		expectedDefValue  map[string]string
		expectedShorthand map[string]string
		expectedUsage     map[string]string
	}{
		{
			use: "all",
			expectedDefValue: map[string]string{
				"domain":           "www.github.com",
				"ip":               "",
				"cloud-hub-server": "",
				"dns-ip":           "",
				"config":           "",
			},
			expectedShorthand: map[string]string{
				"domain":           "d",
				"ip":               "i",
				"cloud-hub-server": "s",
				"dns-ip":           "D",
				"config":           "c",
			},
			expectedUsage: map[string]string{
				"domain":           "specify test domain",
				"ip":               "specify test ip",
				"cloud-hub-server": "specify cloudhub server",
				"dns-ip":           "specify test dns ip",
				"config":           fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath),
			},
		},
		{
			use: "dns",
			expectedDefValue: map[string]string{
				"domain": "www.github.com",
				"dns-ip": "",
			},
			expectedShorthand: map[string]string{
				"domain": "d",
				"dns-ip": "D",
			},
			expectedUsage: map[string]string{
				"domain": "specify test domain",
				"dns-ip": "specify test dns ip",
			},
		},
		{
			use: "network",
			expectedDefValue: map[string]string{
				"ip":               "",
				"cloud-hub-server": "",
				"config":           "",
			},
			expectedShorthand: map[string]string{
				"ip":               "i",
				"cloud-hub-server": "s",
				"config":           "c",
			},
			expectedUsage: map[string]string{
				"ip":               "specify test ip",
				"cloud-hub-server": "specify cloudhub server",
				"config":           fmt.Sprintf("Specify configuration file, default is %s", common.EdgecoreConfigPath),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.use, func(t *testing.T) {
			checkObj := CheckObject{
				Use:  tc.use,
				Desc: fmt.Sprintf("Check %s item", tc.use),
			}
			cmd := NewSubEdgeCheck(checkObj)

			assert.NotNil(cmd)

			flags := cmd.Flags()
			assert.NotNil(flags)

			for flagName, expectedDefValue := range tc.expectedDefValue {
				t.Run(flagName, func(t *testing.T) {
					flag := flags.Lookup(flagName)
					assert.NotNilf(flag, "Flag %s should exist", flagName)

					assert.Equal(expectedDefValue, flag.DefValue)
					assert.Equal(tc.expectedShorthand[flagName], flag.Shorthand)
					assert.Equal(tc.expectedUsage[flagName], flag.Usage)
				})
			}
		})
	}
}

func TestNewCheckOptions(t *testing.T) {
	assert := assert.New(t)
	co := NewCheckOptions()
	assert.NotNil(co)

	assert.Equal("www.github.com", co.Domain)
	assert.Equal(1, co.Timeout)
}
