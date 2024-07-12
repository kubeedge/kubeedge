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

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"k8s.io/component-base/cli/globalflag"

	"github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func TestNewAdmissionCommand(t *testing.T) {
	assert := assert.New(t)

	cmd := NewAdmissionCommand()
	assert.NotNil(cmd)
	assert.Equal("admission", cmd.Use)
	assert.Equal(cmd.Long, `Admission leverage the feature of Dynamic Admission Control from kubernetes, start it
if want to admission control some kubeedge resources.`)

	fs := cmd.Flags()
	assert.NotNil(fs, "Command should have flags")
	namedFs := options.NewAdmissionOptions().Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())

	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	for _, f := range namedFs.FlagSets {
		f.VisitAll(func(flag *pflag.Flag) {
			assert.NotNil(fs.Lookup(flag.Name), "Flag %s should be added to the command", flag.Name)
		})
	}

	usage := &bytes.Buffer{}
	cmd.SetOut(usage)
	err := cmd.Usage()
	assert.NoError(err)
	assert.Contains(usage.String(), "Usage:\n  admission")

	help := &bytes.Buffer{}
	cmd.SetOut(help)
	err = cmd.Help()
	assert.NoError(err)
	assert.Contains(help.String(), "Admission leverage the feature of Dynamic Admission Control from kubernetes")
}
