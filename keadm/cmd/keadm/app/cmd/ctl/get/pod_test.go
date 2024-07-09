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

package get

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubectl/pkg/cmd/get"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewEdgePodGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgePodGet()

	assert.NotNil(cmd)
	assert.Equal("pod", cmd.Use)
	assert.Equal(edgePodGetShortDescription, cmd.Short)
	assert.Equal(edgePodGetShortDescription, cmd.Long)

	assert.NotNil(cmd.RunE)

	assert.Equal(cmd.Flags().Lookup(common.FlagNameNamespace).Name, "namespace")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameAllNamespaces).Name, "all-namespaces")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameLabelSelector).Name, "selector")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameOutput).Name, "output")
}

func TestNewGetOpts(t *testing.T) {
	assert := assert.New(t)

	podGetOptions := NewGetOpts()
	assert.NotNil(podGetOptions)
	assert.Equal(podGetOptions.Namespace, "default")
	assert.Equal(podGetOptions.PrintFlags, get.NewGetPrintFlags())
	assert.Equal(podGetOptions.PrintFlags.OutputFormat, &podGetOptions.Output)
}

func TestAddGetPodFlags(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewGetOpts()

	cmd := &cobra.Command{}

	AddGetPodFlags(cmd, getOptions)

	namespaceFlag := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.NotNil(namespaceFlag)
	assert.Equal("default", namespaceFlag.DefValue)
	assert.Equal("namespace", namespaceFlag.Name)

	labelSelectorFlag := cmd.Flags().Lookup(common.FlagNameLabelSelector)
	assert.NotNil(labelSelectorFlag)
	assert.Equal("", labelSelectorFlag.DefValue)
	assert.Equal("selector", labelSelectorFlag.Name)

	outputFlag := cmd.Flags().Lookup(common.FlagNameOutput)
	assert.NotNil(outputFlag)
	assert.Equal("", outputFlag.DefValue)
	assert.Equal("output", outputFlag.Name)

	allNamespacesFlag := cmd.Flags().Lookup(common.FlagNameAllNamespaces)
	assert.NotNil(allNamespacesFlag)
	assert.Equal("false", allNamespacesFlag.DefValue)
	assert.Equal("all-namespaces", allNamespacesFlag.Name)
}
