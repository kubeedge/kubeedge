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

package rest

import (
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	routerconfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
)

func TestGoToTargetVerifiesAndReusesClient(t *testing.T) {
	nodeNames := make(chan string, 3)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nodeNames <- r.Header.Get("NodeName")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	require.NoError(t, os.WriteFile(caFile, caPEM, 0600))

	oldTargetTLS := routerconfig.Config.RestTargetTLS
	t.Cleanup(func() {
		routerconfig.Config.RestTargetTLS = oldTargetTLS
	})
	routerconfig.Config.RestTargetTLS = &v1alpha1.RouterRestTargetTLS{
		CAFile:             caFile,
		InsecureSkipVerify: false,
	}

	r := &Rest{Endpoint: srv.URL}
	for i := 0; i < 3; i++ {
		v, err := r.GoToTarget(map[string]interface{}{
			"data":     []byte("payload"),
			"nodeName": "edge-node",
		}, nil)
		require.NoError(t, err)

		resp, ok := v.(*http.Response)
		require.True(t, ok)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "ok", string(body))
		assert.Equal(t, "edge-node", <-nodeNames)
	}

	first, err := getTargetClient()
	require.NoError(t, err)
	second, err := getTargetClient()
	require.NoError(t, err)
	assert.Same(t, first, second)
}
