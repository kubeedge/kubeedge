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

import "testing"

func TestIsHotReloadable(t *testing.T) {
	cases := []struct {
		name string
		sets string
		want bool
	}{
		{
			name: "single hot reloadable field",
			sets: "modules.edgeHub.heartbeat=30",
			want: true,
		},
		{
			name: "multiple hot reloadable fields are case insensitive",
			sets: "Modules.EdgeHub.Heartbeat=30,modules.metamanager.remotequerytimeout=90",
			want: true,
		},
		{
			name: "field requiring a restart falls back",
			sets: "modules.edgeHub.websocket.server=example.com",
			want: false,
		},
		{
			name: "mixing a safe and unsafe field falls back",
			sets: "modules.edgeHub.heartbeat=30,modules.edged.address=0.0.0.0",
			want: false,
		},
		{
			name: "empty sets is not hot reloadable",
			sets: "",
			want: false,
		},
		{
			name: "malformed field falls back",
			sets: "modules.edgeHub.heartbeat",
			want: false,
		},
		{
			name: "whitespace around fields is ignored",
			sets: " modules.edgeHub.heartbeat = 30 , modules.metaManager.remoteQueryTimeout = 90 ",
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isHotReloadable(tc.sets); got != tc.want {
				t.Errorf("isHotReloadable(%q) = %v, want %v", tc.sets, got, tc.want)
			}
		})
	}
}
