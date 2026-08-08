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
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/pkg/containers"
	"github.com/kubeedge/kubeedge/pkg/version"
)

func TestPrepareKeadmRequiresImageDigest(t *testing.T) {
	err := prepareKeadm(&commontypes.NodeUpgradeJobRequest{
		Version: "v1.21.0",
		Image:   "kubeedge/installation-package:v1.21.0",
	})

	require.EqualError(t, err, "imageDigest is required for node upgrade jobs")
}

func TestPrepareKeadmCopiesFromImmutableImage(t *testing.T) {
	const validDigest = "sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695"
	cfg := &cfgv1alpha2.EdgeCoreConfig{
		Modules: &cfgv1alpha2.Modules{
			Edged: &cfgv1alpha2.Edged{
				TailoredKubeletConfig: &cfgv1alpha2.TailoredKubeletConfiguration{
					ContainerRuntimeEndpoint: "unix:///var/run/containerd/containerd.sock",
					CgroupDriver:             "systemd",
				},
			},
		},
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(options.GetEdgeCoreConfig, func() *cfgv1alpha2.EdgeCoreConfig {
		return cfg
	})
	patches.ApplyFunc(containers.NewContainerRuntime, func(endpoint, cgroupDriver string,
	) (containers.ContainerRuntime, error) {
		return &containers.ContainerRuntimeImpl{}, nil
	})
	patches.ApplyMethodFunc(reflect.TypeOf(&containers.ContainerRuntimeImpl{}), "PullImages",
		func(_ctx context.Context, images []string, _authConfig *runtimeapi.AuthConfig) error {
			require.Equal(t, []string{"kubeedge/installation-package:v1.21.0"}, images)
			return nil
		})
	patches.ApplyMethodFunc(reflect.TypeOf(&containers.ContainerRuntimeImpl{}), "GetImageDigest",
		func(_ctx context.Context, image string) (string, error) {
			require.Equal(t, "kubeedge/installation-package:v1.21.0", image)
			return validDigest, nil
		})
	patches.ApplyMethodFunc(reflect.TypeOf(&containers.ContainerRuntimeImpl{}), "CopyResources",
		func(_ctx context.Context, image string, files map[string]string) error {
			require.Equal(t, "docker.io/kubeedge/installation-package@"+validDigest, image)
			require.Equal(t, "/usr/local/bin/keadm", files["/usr/local/bin/keadm"])
			return nil
		})

	err := prepareKeadm(&commontypes.NodeUpgradeJobRequest{
		Version:     "v1.21.0",
		Image:       "kubeedge/installation-package:v1.21.0",
		ImageDigest: validDigest,
	})
	require.NoError(t, err)
}

func TestBuildKeadmUpgradeArgsDoesNotUseShell(t *testing.T) {
	const validDigest = "sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695"
	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID:   "upgrade-1; touch /tmp/pwned",
		HistoryID:   "history-1$(touch /tmp/pwned)",
		Version:     "v1.23.1; rm -rf /",
		Image:       "kubeedge/installation-package; touch /tmp/pwned",
		ImageDigest: validDigest,
	}
	opts := &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml; touch /tmp/pwned",
	}

	args, err := buildKeadmUpgradeArgs(upgradeReq, opts)
	require.NoError(t, err)

	want := []string{
		"upgrade", "edge",
		"--upgradeID", upgradeReq.UpgradeID,
		"--historyID", upgradeReq.HistoryID,
		"--fromVersion", version.Get().String(),
		"--toVersion", upgradeReq.Version,
		"--config", opts.ConfigFile,
		"--image", upgradeReq.Image,
		"--image-digest", validDigest,
	}

	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected args: got %v, want %v", args, want)
	}
}

func TestBuildKeadmUpgradeArgsRequiresValidDigest(t *testing.T) {
	upgradeReq := commontypes.NodeUpgradeJobRequest{
		UpgradeID: "upgrade-1",
		HistoryID: "history-1",
		Version:   "v1.23.1",
		Image:     "kubeedge/installation-package:v1.23.1",
	}
	opts := &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
	}

	_, err := buildKeadmUpgradeArgs(upgradeReq, opts)
	require.Error(t, err)
}

func TestKeadmUpgradeReturnsErrorWhenCommandMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	const validDigest = "sha256:e47afdf2746ad10ee76dd64289eae01895000327c0f23c5b498959eca6953695"

	err := keadmUpgrade(commontypes.NodeUpgradeJobRequest{
		UpgradeID:   "upgrade-1",
		HistoryID:   "history-1",
		Version:     "v1.23.1",
		Image:       "kubeedge/installation-package:v1.23.1",
		ImageDigest: validDigest,
	}, &options.EdgeCoreOptions{
		ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
	})

	if err == nil {
		t.Fatal("expected error when keadm command cannot be started")
	}
	if !strings.Contains(err.Error(), "failed to start keadm upgrade command") {
		t.Fatalf("unexpected error: %v", err)
	}
}
