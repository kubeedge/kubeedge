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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

func TestConfigUpdateHelmExpressions(t *testing.T) {
	tests := []struct {
		name    string
		sets    string
		update  func(*v1alpha2.EdgeCoreConfig)
		wantErr string
	}{
		{
			name: "scalar assignment",
			sets: "modules.edgeHub.heartbeat=30",
			update: func(cfg *v1alpha2.EdgeCoreConfig) {
				cfg.Modules.EdgeHub.Heartbeat = 30
			},
		},
		{
			name: "comma-separated scalar assignments",
			sets: "modules.edgeHub.heartbeat=30,modules.metaManager.remoteQueryTimeout=90",
			update: func(cfg *v1alpha2.EdgeCoreConfig) {
				cfg.Modules.EdgeHub.Heartbeat = 30
				cfg.Modules.MetaManager.RemoteQueryTimeout = 90
			},
		},
		{
			name: "list followed by a scalar assignment",
			sets: "modules.metaManager.metaServer.serviceAccountIssuers={issuer-one,issuer-two},modules.edgeHub.heartbeat=30",
			update: func(cfg *v1alpha2.EdgeCoreConfig) {
				cfg.Modules.MetaManager.MetaServer.ServiceAccountIssuers = []string{"issuer-one", "issuer-two"}
				cfg.Modules.EdgeHub.Heartbeat = 30
			},
		},
		{
			name: "escaped comma followed by a scalar assignment",
			sets: `database.aliasName=edge\,cache,modules.edgeHub.heartbeat=30`,
			update: func(cfg *v1alpha2.EdgeCoreConfig) {
				cfg.DataBase.AliasName = "edge,cache"
				cfg.Modules.EdgeHub.Heartbeat = 30
			},
		},
		{
			name:    "unterminated list",
			sets:    "modules.metaManager.metaServer.serviceAccountIssuers={issuer-one,issuer-two",
			wantErr: "list must terminate",
		},
		{
			name:    "unknown configuration field",
			sets:    "modules.metaManager.unknownField=true",
			wantErr: "unknownField",
		},
		{
			name:    "invalid field type",
			sets:    "modules.edgeHub.heartbeat=not-a-number",
			wantErr: "cannot unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := v1alpha2.NewDefaultEdgeCoreConfig()
			cfg.DataBase.DataSource = filepath.Join(dir, "edgecore.db")
			cfg.Modules.Edged.HostnameOverride = "test-node"
			original, err := yaml.Marshal(cfg)
			require.NoError(t, err)
			configFile := filepath.Join(dir, "edgecore.yaml")
			require.NoError(t, os.WriteFile(configFile, original, 0600))

			// Keep configuration parsing, validation, and file writes real. Never
			// restart a service from the test.
			restarts := 0
			patches := gomonkey.ApplyMethod(reflect.TypeOf(&execs.Command{}), "Exec", func(_ *execs.Command) error {
				restarts++
				return nil
			})
			defer patches.Reset()

			executor := newConfigUpdateExecutor()
			err = executor.configUpdate(ConfigUpdateOptions{
				BaseOptions: BaseOptions{Config: configFile},
				Sets:        tt.sets,
			})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				unchanged, readErr := os.ReadFile(configFile)
				require.NoError(t, readErr)
				require.Equal(t, original, unchanged, "invalid input must not change the config")
				require.Zero(t, restarts, "invalid input must not restart EdgeCore")
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, restarts)

			want := &v1alpha2.EdgeCoreConfig{}
			require.NoError(t, yaml.UnmarshalStrict(original, want))
			tt.update(want)
			written, err := os.ReadFile(configFile)
			require.NoError(t, err)
			got := &v1alpha2.EdgeCoreConfig{}
			require.NoError(t, yaml.UnmarshalStrict(written, got))
			require.Equal(t, want, got, "preserve unrelated configuration fields")
		})
	}
}
