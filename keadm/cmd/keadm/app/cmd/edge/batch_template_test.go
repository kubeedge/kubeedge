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
	"errors"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// fakeSSHConn lets a zero-value ssh.Client be closed safely in tests.
type fakeSSHConn struct {
	ssh.Conn
}

func (fakeSSHConn) Close() error { return nil }

// loadGenConfigTemplate parses the file written by `keadm batch gen-config`,
// the way `keadm batch` parses a config file. Optional keys that the template
// leaves as "" must behave like keys that were left out.
func loadGenConfigTemplate(t *testing.T) *common.Config {
	t.Helper()

	cfg := &common.Config{}
	require.NoError(t, yaml.Unmarshal([]byte(getConfigTemplate()), cfg))
	// The template asks the user to fill in the required keadmVersion.
	cfg.Keadm.KeadmVersion = "v1.19.0"
	return cfg
}

func TestDownloadKeadmPackagesWithEmptyURL(t *testing.T) {
	cfg := loadGenConfigTemplate(t)

	var urls []string
	extractCalls := 0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error { return nil })
	patches.ApplyFunc(extractTarGz, func(string, string) error {
		// Fail the first extraction, so that the package gets downloaded.
		extractCalls++
		if extractCalls == 1 {
			return errors.New("package is not downloaded yet")
		}
		return nil
	})
	patches.ApplyFunc(downloadFile, func(url, _ string) error {
		urls = append(urls, url)
		return nil
	})

	require.NoError(t, downloadKeadmPackages(cfg))
	assert.Equal(t, []string{
		defautKeadmDownloadURL + "/v1.19.0/keadm-v1.19.0-linux-amd64.tar.gz",
	}, urls)
}

func TestDownloadKeadmPackagesWithCustomURL(t *testing.T) {
	cfg := loadGenConfigTemplate(t)
	customURL := "https://mirror.example.com/keadm"
	cfg.Keadm.Download.URL = &customURL

	var urls []string
	extractCalls := 0

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error { return nil })
	patches.ApplyFunc(extractTarGz, func(string, string) error {
		extractCalls++
		if extractCalls == 1 {
			return errors.New("package is not downloaded yet")
		}
		return nil
	})
	patches.ApplyFunc(downloadFile, func(url, _ string) error {
		urls = append(urls, url)
		return nil
	})

	require.NoError(t, downloadKeadmPackages(cfg))
	assert.Equal(t, []string{customURL + "/keadm-v1.19.0-linux-amd64.tar.gz"}, urls)
}

func TestProcessNodeWithCopyFrom(t *testing.T) {
	tests := []struct {
		name         string
		copyFrom     string
		wantUploaded []string
	}{
		{
			name:         "empty copyFrom as in the generated template",
			copyFrom:     "",
			wantUploaded: nil,
		},
		{
			name:         "copyFrom is set",
			copyFrom:     "/opt/files",
			wantUploaded: []string{"/opt/files"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadGenConfigTemplate(t)
			node := &cfg.Nodes[0]
			node.CopyFrom = &tt.copyFrom

			var uploaded []string

			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyFunc(connectSSH, func(common.SSH) (*ssh.Client, error) {
				return &ssh.Client{Conn: fakeSSHConn{}}, nil
			})
			patches.ApplyFunc(createRemoteDir, func(*ssh.Client, string) error { return nil })
			patches.ApplyFunc(uploadFiles, func(_ *ssh.Client, _, srcDir, _ string) error {
				uploaded = append(uploaded, srcDir)
				return nil
			})
			patches.ApplyFunc(getNodeArch, func(*ssh.Client) (string, error) { return "amd64", nil })
			patches.ApplyFunc(uploadFile, func(*ssh.Client, string, string, string) error { return nil })
			patches.ApplyFunc(executeKeadmCommand, func(*ssh.Client, string, string, *string) error { return nil })

			require.NoError(t, processNode(node, cfg))
			assert.Equal(t, tt.wantUploaded, uploaded)
		})
	}
}
