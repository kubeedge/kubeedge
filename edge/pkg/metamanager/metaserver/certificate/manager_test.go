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

package certificate

import (
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/cert"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
)

// patchCAPolling shortens the CA polling for a test and connects to the cloud.
func patchCAPolling(t *testing.T, interval, timeout time.Duration) {
	t.Helper()
	oldInterval, oldTimeout := caPollInterval, caPollTimeout
	caPollInterval, caPollTimeout = interval, timeout
	cloudconnection.SetConnected(true)
	t.Cleanup(func() {
		caPollInterval, caPollTimeout = oldInterval, oldTimeout
		cloudconnection.SetConnected(false)
	})
}

func TestWaitForCAReadyRetriesAfterFailure(t *testing.T) {
	patchCAPolling(t, 10*time.Millisecond, 5*time.Second)

	calls := 0
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(beehivecontext.SendSync, func(_ string, _ model.Message, _ time.Duration) (model.Message, error) {
		calls++
		if calls < 3 {
			return model.Message{}, errors.New("timeout")
		}
		return *model.NewMessage("").FillBody([]byte("ca data")), nil
	})
	var written []byte
	patches.ApplyFunc(cert.WriteCert, func(_ string, data []byte) error {
		written = data
		return nil
	})

	scm := &ServerCertificateManager{}
	require.NoError(t, scm.WaitForCAReady())
	assert.Equal(t, 3, calls)
	assert.Equal(t, []byte("ca data"), written)
}

func TestWaitForCAReadyTimeoutReportsLastError(t *testing.T) {
	patchCAPolling(t, 10*time.Millisecond, 100*time.Millisecond)

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(beehivecontext.SendSync, func(_ string, _ model.Message, _ time.Duration) (model.Message, error) {
		return model.Message{}, errors.New("cloudcore is not ready")
	})

	scm := &ServerCertificateManager{}
	err := scm.WaitForCAReady()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloudcore is not ready")
}

func TestWaitForCAReadyWaitsForCloudConnection(t *testing.T) {
	patchCAPolling(t, 10*time.Millisecond, 100*time.Millisecond)
	cloudconnection.SetConnected(false)

	calls := 0
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(beehivecontext.SendSync, func(_ string, _ model.Message, _ time.Duration) (model.Message, error) {
		calls++
		return model.Message{}, errors.New("unexpected call")
	})

	scm := &ServerCertificateManager{}
	require.Error(t, scm.WaitForCAReady())
	assert.Equal(t, 0, calls, "the CA must not be requested while the cloud is not connected")
}
