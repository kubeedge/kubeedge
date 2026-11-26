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

package edge

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func newBatchTestConfig(names ...string) *common.Config {
	nodes := make([]common.Node, len(names))
	for i, name := range names {
		nodes[i].NodeName = name
	}
	return &common.Config{Nodes: nodes, MaxRunNum: len(names)}
}

func runBatchProcessNodesTest(t *testing.T, cfg *common.Config) (error, string) {
	t.Helper()
	var output bytes.Buffer
	logWriter := bufio.NewWriter(&output)
	err := batchProcessNodes(cfg, logWriter)
	assert.NoError(t, logWriter.Flush())
	return err, output.String()
}

func TestBatchProcessNodes(t *testing.T) {
	var processNodeMock func(*common.Node) error
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(processNode, func(node *common.Node, _cfg *common.Config) error {
		return processNodeMock(node)
	})

	t.Run("all nodes succeed", func(t *testing.T) {
		processNodeMock = func(_node *common.Node) error {
			return nil
		}

		cfg := newBatchTestConfig("node-1", "node-2")
		err, output := runBatchProcessNodesTest(t, cfg)

		assert.NoError(t, err)
		for _, node := range cfg.Nodes {
			assert.Contains(t, output, "Successfully processed node "+node.NodeName)
		}
	})

	t.Run("failed nodes return an error", func(t *testing.T) {
		processNodeMock = func(_node *common.Node) error {
			return errors.New("test node failure")
		}

		cfg := newBatchTestConfig("failed-node")
		err, output := runBatchProcessNodesTest(t, cfg)

		assert.EqualError(t, err, "1 node(s) failed")
		assert.Contains(t, output, "Failed to process node failed-node: test node failure")
	})

	t.Run("all nodes run before returning an error", func(t *testing.T) {
		var calls int32
		processNodeMock = func(node *common.Node) error {
			atomic.AddInt32(&calls, 1)
			if node.NodeName == "failed-node" {
				return errors.New("test node failure")
			}
			return nil
		}

		cfg := newBatchTestConfig("failed-node", "successful-node-1", "successful-node-2")
		err, output := runBatchProcessNodesTest(t, cfg)

		assert.Equal(t, int32(len(cfg.Nodes)), atomic.LoadInt32(&calls))
		assert.EqualError(t, err, "1 node(s) failed")
		assert.Contains(t, output, "Failed to process node failed-node: test node failure")
		assert.Contains(t, output, "Successfully processed node successful-node-1")
		assert.Contains(t, output, "Successfully processed node successful-node-2")
	})

	t.Run("writes one log line per node", func(t *testing.T) {
		processNodeMock = func(_node *common.Node) error {
			return nil
		}

		cfg := newBatchTestConfig("node-1", "node-2", "node-3", "node-4")
		err, output := runBatchProcessNodesTest(t, cfg)

		assert.NoError(t, err)
		assert.Len(t, strings.Split(strings.TrimSpace(output), "\n"), len(cfg.Nodes))
	})
}
