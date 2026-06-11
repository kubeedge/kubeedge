/*
Copyright 2026 The KubeEdge Authors.

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

package unhold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEdgeUnholdUpgrade(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()

	assert.NotNil(t, cmd)
	assert.Equal(t, "unhold-upgrade <resource-type> [<name>] [--namespace namespace]", cmd.Use)
	assert.Equal(t, "Unhold an upgrade for a pod or node-wide", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag)
	assert.Equal(t, "default", namespaceFlag.DefValue)
}

func TestUnholdUpgradeRequiresResourceType(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()

	err := cmd.Args(cmd, []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg")
}

func TestUnholdPodUpgradeRequiresPodName(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()

	err := cmd.RunE(cmd, []string{"pod"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pod name is required")
}

func TestUnholdUpgradeUnknownResourceType(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()

	err := cmd.RunE(cmd, []string{"deployment", "test-deployment"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type: deployment")
}

func TestUnholdUpgradeNamespaceFlag(t *testing.T) {
	cmd := NewEdgeUnholdUpgrade()

	err := cmd.Flags().Set("namespace", "test-namespace")
	assert.NoError(t, err)

	namespace, err := cmd.Flags().GetString("namespace")
	assert.NoError(t, err)
	assert.Equal(t, "test-namespace", namespace)
}
