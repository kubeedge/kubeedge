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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// tlsFiles returns the CA, certificate and key paths of every module that
// uses the edge certificate, keyed by their edgecore.yaml field names.
func tlsFiles(c *cfgv1alpha2.EdgeCoreConfig) map[string]string {
	return map[string]string{
		"edgeHub.tlsCaFile":                          c.Modules.EdgeHub.TLSCAFile,
		"edgeHub.tlsCertFile":                        c.Modules.EdgeHub.TLSCertFile,
		"edgeHub.tlsPrivateKeyFile":                  c.Modules.EdgeHub.TLSPrivateKeyFile,
		"edgeStream.tlsTunnelCAFile":                 c.Modules.EdgeStream.TLSTunnelCAFile,
		"edgeStream.tlsTunnelCertFile":               c.Modules.EdgeStream.TLSTunnelCertFile,
		"edgeStream.tlsTunnelPrivateKeyFile":         c.Modules.EdgeStream.TLSTunnelPrivateKeyFile,
		"eventBus.eventBusTLS.tlsMqttCAFile":         c.Modules.EventBus.TLS.TLSMqttCAFile,
		"eventBus.eventBusTLS.tlsMqttCertFile":       c.Modules.EventBus.TLS.TLSMqttCertFile,
		"eventBus.eventBusTLS.tlsMqttPrivateKeyFile": c.Modules.EventBus.TLS.TLSMqttPrivateKeyFile,
		"metaManager.metaServer.tlsCaFile":           c.Modules.MetaManager.MetaServer.TLSCaFile,
		"metaManager.metaServer.tlsCertFile":         c.Modules.MetaManager.MetaServer.TLSCertFile,
		"metaManager.metaServer.tlsPrivateKeyFile":   c.Modules.MetaManager.MetaServer.TLSPrivateKeyFile,
	}
}

// wantTLSFiles returns the expected tlsFiles result: the default CA file and
// the given certificate and key for every module.
func wantTLSFiles(certFile, keyFile string) map[string]string {
	return map[string]string{
		"edgeHub.tlsCaFile":                          constants.DefaultCAFile,
		"edgeHub.tlsCertFile":                        certFile,
		"edgeHub.tlsPrivateKeyFile":                  keyFile,
		"edgeStream.tlsTunnelCAFile":                 constants.DefaultCAFile,
		"edgeStream.tlsTunnelCertFile":               certFile,
		"edgeStream.tlsTunnelPrivateKeyFile":         keyFile,
		"eventBus.eventBusTLS.tlsMqttCAFile":         constants.DefaultMqttCAFile,
		"eventBus.eventBusTLS.tlsMqttCertFile":       certFile,
		"eventBus.eventBusTLS.tlsMqttPrivateKeyFile": keyFile,
		"metaManager.metaServer.tlsCaFile":           constants.DefaultCAFile,
		"metaManager.metaServer.tlsCertFile":         certFile,
		"metaManager.metaServer.tlsPrivateKeyFile":   keyFile,
	}
}

func TestCreateEdgeConfigFilesCertPath(t *testing.T) {
	customCertPath := filepath.Join(t.TempDir(), "certs")
	customCertFile := filepath.Join(customCertPath, "server.crt")
	customKeyFile := filepath.Join(customCertPath, "server.key")

	wd, err := os.Getwd()
	require.NoError(t, err)

	cases := []struct {
		name     string
		certPath string
		sets     []string
		want     map[string]string
	}{
		{
			name:     "default cert path keeps the default files",
			certPath: constants.DefaultCertPath,
			want:     wantTLSFiles(constants.DefaultCertFile, constants.DefaultKeyFile),
		},
		{
			name:     "empty cert path keeps the default files",
			certPath: "",
			want:     wantTLSFiles(constants.DefaultCertFile, constants.DefaultKeyFile),
		},
		{
			name:     "custom cert path sets the certificate and key files",
			certPath: customCertPath,
			want:     wantTLSFiles(customCertFile, customKeyFile),
		},
		{
			name:     "relative cert path is made absolute",
			certPath: "certs",
			want: wantTLSFiles(filepath.Join(wd, "certs", "server.crt"),
				filepath.Join(wd, "certs", "server.key")),
		},
		{
			name:     "set flag still takes precedence over the cert path",
			certPath: customCertPath,
			sets:     []string{"modules.edgeHub.tlsCertFile=/opt/edge/edge.crt"},
			want: func() map[string]string {
				want := wantTLSFiles(customCertFile, customKeyFile)
				want["edgeHub.tlsCertFile"] = "/opt/edge/edge.crt"
				return want
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			written := false
			patches := gomonkey.NewPatches()
			defer patches.Reset()
			patches.ApplyMethodFunc(reflect.TypeOf(&cfgv1alpha2.EdgeCoreConfig{}), "WriteTo",
				func(_ string) error {
					written = true
					return nil
				})

			// Start from the default config, but keep the database directory
			// that validation creates inside the test's temp dir.
			edgeCoreConfig = cfgv1alpha2.NewDefaultEdgeCoreConfig()
			edgeCoreConfig.DataBase.DataSource = filepath.Join(t.TempDir(), "edgecore.db")
			defer func() { edgeCoreConfig = nil }()

			opt := &common.JoinOptions{
				KubeEdgeVersion: "v1.21.0",
				CloudCoreIPPort: "192.168.1.10:10000",
				EdgeNodeName:    "edge-node",
				CGroupDriver:    cfgv1alpha2.CGroupDriverCGroupFS,
				HubProtocol:     api.ProtocolTypeWS,
				CertPath:        tc.certPath,
				Sets:            tc.sets,
			}
			require.NoError(t, createEdgeConfigFiles(opt))
			require.True(t, written)
			assert.Equal(t, tc.want, tlsFiles(edgeCoreConfig))
		})
	}
}
