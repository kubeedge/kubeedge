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

package hotreload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	edgehubconfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	metamanagerconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
)

func TestReload(t *testing.T) {
	edgehubconfig.InitConfigure(&v1alpha2.EdgeHub{
		Heartbeat: 15,
		WebSocket: &v1alpha2.EdgeHubWebSocket{Server: "test-server"},
	}, "test-node")
	metamanagerconfig.InitConfigure(&v1alpha2.MetaManager{
		RemoteQueryTimeout: 60,
		MetaServer:         &v1alpha2.MetaServer{},
	})

	configFile := filepath.Join(t.TempDir(), "edgecore.yaml")
	const content = `
modules:
  edgeHub:
    heartbeat: 45
  metaManager:
    remoteQueryTimeout: 120
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	reload(configFile)

	if got := edgehubconfig.GetHeartbeat(); got != 45 {
		t.Errorf("GetHeartbeat() = %d, want 45", got)
	}
	if got := metamanagerconfig.GetRemoteQueryTimeout(); got != 120 {
		t.Errorf("GetRemoteQueryTimeout() = %d, want 120", got)
	}
}

func TestReloadMissingFile(t *testing.T) {
	// reload must not panic when the config file cannot be read.
	reload(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
}

func TestReloadNoModulesSection(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "edgecore.yaml")
	if err := os.WriteFile(configFile, []byte("edgecoreVersion: v1.0.0\n"), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// reload must not panic when the modules section is absent.
	reload(configFile)
}
