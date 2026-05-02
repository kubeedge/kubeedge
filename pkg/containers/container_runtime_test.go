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

package containers

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyResourcesCmd(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		contains []string
		wantLen  int
	}{
		{
			name:     "single file",
			files:    map[string]string{"/usr/bin/edgecore": "/usr/bin/edgecore"},
			contains: []string{fmt.Sprintf("cp /usr/bin/edgecore %s", filepath.Join("/tmp", "/usr/bin/edgecore"))},
			wantLen:  1,
		},
		{
			name:     "multiple files",
			files:    map[string]string{"/src1": "/dest1", "/src2": "/dest2"},
			contains: []string{
				fmt.Sprintf("cp /src1 %s", filepath.Join("/tmp", "/dest1")),
				fmt.Sprintf("cp /src2 %s", filepath.Join("/tmp", "/dest2")),
			},
			wantLen:  2,
		},
		{
			name:     "empty map",
			files:    map[string]string{},
			contains: []string{},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyResourcesCmd(tt.files)

			if tt.wantLen == 0 {
				if got != "" {
					t.Errorf("copyResourcesCmd() = %v, want empty string", got)
				}
				return
			}

			parts := strings.Split(got, " && ")
			if len(parts) != tt.wantLen {
				t.Errorf("copyResourcesCmd() generated %d commands, want %d", len(parts), tt.wantLen)
			}

			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("copyResourcesCmd() = %v, must contain %v", got, c)
				}
			}
		})
	}
}
