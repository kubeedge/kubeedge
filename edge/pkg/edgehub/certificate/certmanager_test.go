/*
Copyright 2022 The KubeEdge Authors.

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
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/common/constants"
	commhttp "github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	httpfake "github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http/fake"
)

func TestGetCACert(t *testing.T) {
	const fakehost = "http://localhost"

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(commhttp.SendRequest,
		func(_ *http.Request, _ *http.Client) (*http.Response, error) {
			return &http.Response{Body: httpfake.NewFakeBodyReader([]byte{})}, nil
		})

	_, err := GetCACert(fakehost + "/ca.crt")
	require.NoError(t, err)
}

func TestGetEdgeCert(t *testing.T) {
	const (
		fakehost = "http://localhost"

		capem = `-----BEGIN CERTIFICATE-----
MIIBejCCAR+gAwIBAgICBAAwCgYIKoZIzj0EAwIwEzERMA8GA1UEAxMIS3ViZUVk
Z2UwIBcNMjQwNTA5MDczNzU2WhgPMjEyNDAxMDYwNzM3NTZaMBMxETAPBgNVBAMT
CEt1YmVFZGdlMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEq4Rd11aJ/FXEYBE2
YCUMjRZVpqytxDBq2anuzokPculGaTrSDiRy1IKukPhlg34bq7J6wqkF0cmFUvcT
jtReq6NhMF8wDgYDVR0PAQH/BAQDAgKkMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggr
BgEFBQcDAjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTvK3f704DC7OOiVbmO
PyKwJAUwQjAKBggqhkjOPQQDAgNJADBGAiEAkOgvZtFy+aYSqsfxIVMXxScsGilA
P1Iiy/r5PerqODcCIQCH+qeEuxIgZzUAD/Wm6xameEmyn/mO4su/4UE6APNZFQ==
-----END CERTIFICATE-----`
	)

	t.Run("request failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(commhttp.SendRequest,
			func(_ *http.Request, _ *http.Client) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       httpfake.NewFakeBodyReader([]byte("test error")),
				}, nil
			})

		cm := &CertManager{}
		tls, err := tls.LoadX509KeyPair("testdata/server.crt", "testdata/server.key")
		require.NoError(t, err)

		_, _, err = cm.GetEdgeCert(fakehost+constants.DefaultCAURL, []byte(capem), tls, "")
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to call http, code: 500, message: test error")
	})

	t.Run("request successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(commhttp.SendRequest,
			func(_ *http.Request, _ *http.Client) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       httpfake.NewFakeBodyReader([]byte("test cert...")),
				}, nil
			})

		cm := &CertManager{}
		tls, err := tls.LoadX509KeyPair("testdata/server.crt", "testdata/server.key")
		require.NoError(t, err)

		_, _, err = cm.GetEdgeCert(fakehost+constants.DefaultCAURL, []byte(capem), tls, "")
		require.NoError(t, err)
	})
}
