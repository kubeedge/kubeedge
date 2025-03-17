/*
Copyright 2025 The KubeEdge Authors.

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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEdegeCoreVersion(t *testing.T) {
	err := WriteEdgeCoreVersion("")
	assert.NoError(t, err)

	version, err := ReadEdgeCoreVersion("")
	assert.NoError(t, err)
	assert.NotEmpty(t, version)

	err = RemoveEdgeCoreVersion("")
	assert.NoError(t, err)

	version, err = ReadEdgeCoreVersion("")
	assert.NoError(t, err)
	assert.Empty(t, version)
}

func TestGetEdgeCoreVersionFeile(t *testing.T) {
	cases := []struct {
		configPath string
		want       string
	}{
		{
			configPath: "/etc/kubeedge/config/edgecore.yaml",
			want:       "/etc/kubeedge/edgecore_version",
		},
		{
			configPath: "/home/user/edgecore.yaml",
			want:       "/home/user/edgecore_version",
		},
	}
	for _, c := range cases {
		got := getEdgeCoreVersionFile(c.configPath)
		assert.Equal(t, c.want, got)
	}
}
