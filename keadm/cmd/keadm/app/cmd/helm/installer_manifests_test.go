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

package helm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHelmManifestAppendsGivenManifests(t *testing.T) {
	chartFS := fstest.MapFS{
		"chart/Chart.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v2\nname: testchart\nversion: 1.0.0"),
		},
		"chart/values.yaml": &fstest.MapFile{
			Data: []byte("key: value"),
		},
		"chart/templates/configmap.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: rendered"),
		},
	}
	renderer := NewGenericRenderer(chartFS, "chart", "testcomponent", "default", map[string]interface{}{}, true)
	require.NoError(t, renderer.LoadChart())

	dir := t.TempDir()
	first := "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: first\n"
	second := "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: second\n"
	firstFile := filepath.Join(dir, "first.yaml")
	secondFile := filepath.Join(dir, "second.yaml")
	require.NoError(t, os.WriteFile(firstFile, []byte(first), 0o600))
	require.NoError(t, os.WriteFile(secondFile, []byte(second), 0o600))

	cu := &KubeCloudHelmInstTool{Manifests: firstFile + "," + secondFile}
	var out bytes.Buffer
	require.NoError(t, cu.runHelmManifest(renderer, &out))

	// The given manifests come first, in the given order, each followed by a YAML separator,
	// then the rendered manifests.
	wantPrefix := first + YAMLSeparator + second + YAMLSeparator
	got := out.String()
	assert.True(t, strings.HasPrefix(got, wantPrefix), "unexpected output: %q", got)
	assert.Contains(t, strings.TrimPrefix(got, wantPrefix), "name: rendered")
}
