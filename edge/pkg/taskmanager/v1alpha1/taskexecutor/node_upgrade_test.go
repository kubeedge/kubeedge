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

package taskexecutor

import (
	"reflect"
	"testing"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func TestBuildKeadmUpgradeArgsDoesNotUseShell(t *testing.T) {
	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-1; touch /tmp/pwned",
		HistoryID: "history-1$(touch /tmp/pwned)",
		Version:   "v1.23.1; rm -rf /",
		Image:     "kubeedge/installation-package; touch /tmp/pwned",
	}
	opts := &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml; touch /tmp/pwned",
	}

	args := buildKeadmUpgradeArgs(upgradeReq, opts)

	want := []string{
		"upgrade", "edge",
		"--upgradeID", upgradeReq.UpgradeID,
		"--historyID", upgradeReq.HistoryID,
		"--fromVersion", version.Get().String(),
		"--toVersion", upgradeReq.Version,
		"--config", opts.ConfigFile,
		"--image", upgradeReq.Image,
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got %v, want %v", args, want)
	}
}
