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

package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIptablesManagerCommand(t *testing.T) {
	assert := assert.New(t)

	cmd := NewIptablesManagerCommand()

	assert.NotNil(cmd)
	assert.Equal("iptables", cmd.Use)
	assert.Equal("IptablesManager from KubeEdge on the cloudside", cmd.Long)

	assert.NotNil(cmd.Run)

	flags := cmd.Flags()
	assert.NotNil(flags, "Expected command to have flags")

	kubeConfigFlag := flags.Lookup("kubeconfig")
	assert.NotNil(kubeConfigFlag)
	assert.Equal("The KubeConfig path. Flags override values in this file.", kubeConfigFlag.Usage)
	assert.Equal("", kubeConfigFlag.DefValue)

	forwardPortFlag := flags.Lookup("forwardport")
	assert.NotNil(forwardPortFlag)
	assert.Equal("The forward port, default is the stream port, 10003.", forwardPortFlag.Usage, "Expected correct usage message for forwardport flag")
	assert.Equal("10003", forwardPortFlag.DefValue)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	usageFunc := cmd.UsageFunc()
	err := usageFunc(cmd)
	assert.NoError(err)

	output := buf.String()
	expectedUsage := "Usage:\n  iptables [flags]\n"
	assert.Contains(output, expectedUsage)

	buf.Reset()
	helpFunc := cmd.HelpFunc()
	helpFunc(cmd, []string{})

	output = buf.String()
	expectedHelp := "IptablesManager from KubeEdge on the cloudside\n\nUsage:\n  iptables [flags]\n"
	assert.Contains(output, expectedHelp)
}
