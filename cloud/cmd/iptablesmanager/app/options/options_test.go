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

package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIptablesManagerOptions(t *testing.T) {
	assert := assert.New(t)

	options := NewIptablesManagerOptions()

	assert.NotNil(options)
	assert.Equal("", options.KubeConfig)
	assert.Equal(10003, options.ForwardPort)
}

func TestFlags(t *testing.T) {
	assert := assert.New(t)

	options := NewIptablesManagerOptions()
	flagSets := options.Flags()

	fs := flagSets.FlagSet("IptablesManager")
	assert.NotNil(fs)

	kubeConfigFlag := fs.Lookup("kubeconfig")
	assert.NotNil(kubeConfigFlag)
	assert.Equal("The KubeConfig path. Flags override values in this file.", kubeConfigFlag.Usage)
	assert.Equal("", kubeConfigFlag.DefValue)

	forwardPortFlag := fs.Lookup("forwardport")
	assert.NotNil(forwardPortFlag)
	assert.Equal("The forward port, default is the stream port, 10003.", forwardPortFlag.Usage, "Expected correct usage message for forwardport flag")
	assert.Equal("10003", forwardPortFlag.DefValue)
}
