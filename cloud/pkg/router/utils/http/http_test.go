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

package http

import (
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTLSServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeServerCA(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caFile, caPEM, 0600))
	return caFile
}

func get(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func TestNewRestTargetTLSConfig(t *testing.T) {
	srv := newTLSServer(t)
	caFile := writeServerCA(t, srv)

	invalidCAFile := filepath.Join(t.TempDir(), "invalid.crt")
	require.NoError(t, os.WriteFile(invalidCAFile, []byte("not a certificate"), 0600))

	cases := []struct {
		name               string
		caFile             string
		insecureSkipVerify bool
		wantConfigErr      bool
		wantRequestErr     bool
	}{
		{
			name:               "insecure skip verify accepts any certificate",
			insecureSkipVerify: true,
		},
		{
			name:   "target certificate verified with caFile",
			caFile: caFile,
		},
		{
			name:           "untrusted target certificate is rejected",
			wantRequestErr: true,
		},
		{
			name:          "caFile does not exist",
			caFile:        filepath.Join(t.TempDir(), "not-exist.crt"),
			wantConfigErr: true,
		},
		{
			name:          "caFile without certificates",
			caFile:        invalidCAFile,
			wantConfigErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tlsConfig, err := NewRestTargetTLSConfig(c.caFile, c.insecureSkipVerify)
			if c.wantConfigErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			err = get(NewHTTPClientWithTLSConfig(tlsConfig), srv.URL)
			if c.wantRequestErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewHTTPClientWithTLSConfigReusesConnections(t *testing.T) {
	var newConns int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			atomic.AddInt32(&newConns, 1)
		}
	}
	srv.StartTLS()
	defer srv.Close()

	tlsConfig, err := NewRestTargetTLSConfig(writeServerCA(t, srv), false)
	require.NoError(t, err)
	client := NewHTTPClientWithTLSConfig(tlsConfig)

	for i := 0; i < 10; i++ {
		require.NoError(t, get(client, srv.URL))
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&newConns))
}
