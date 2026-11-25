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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runBatchGenConfig executes the real `keadm batch gen-config` command with
// dir as the working directory, the way the CLI resolves it.
func runBatchGenConfig(t *testing.T, dir string) error {
	t.Helper()
	t.Chdir(dir)

	cmd := NewEdgeBatchProcess()
	cmd.SetArgs([]string{"gen-config"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

func TestBatchProcessGenConfig(t *testing.T) {
	t.Run("generates the template in an empty directory", func(t *testing.T) {
		dir := t.TempDir()

		err := runBatchGenConfig(t, dir)

		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, getConfigTemplate(), string(got))
	})

	// Regression test for https://github.com/kubeedge/kubeedge/issues/7483
	t.Run("keeps an edited config.yaml and reports an error", func(t *testing.T) {
		dir := t.TempDir()
		existing := strings.Replace(getConfigTemplate(), "nodeName: edge-node", "nodeName: my-edge", 1)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(existing), 0644))

		err := runBatchGenConfig(t, dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
		got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, existing, string(got))
	})

	t.Run("keeps an unrelated config.yaml", func(t *testing.T) {
		dir := t.TempDir()
		existing := "name: prod-cluster-config\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(existing), 0644))

		err := runBatchGenConfig(t, dir)

		require.Error(t, err)
		got, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, existing, string(got))
	})

	t.Run("refuses and preserves when config.yaml is a directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "config.yaml"), 0755))

		err := runBatchGenConfig(t, dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
		require.DirExists(t, filepath.Join(dir, "config.yaml"))
	})
}
