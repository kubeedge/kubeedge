/*
Copyright 2024 The KubeEdge Authors.

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

// Note: Tests in this file use gomonkey for function patching.
// Run tests with inlining disabled:
// go test -gcflags=all=-l ./cloud/pkg/cloudhub/servers/httpserver/...

package httpserver

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
)

// TestGenerateAndRefreshToken_StopsOnContextCancel guards against a
// break-in-select bug where cancelling ctx only exited the select
// statement instead of the refresh goroutine, leaving it spinning in a
// tight loop forever. On a fixed implementation the goroutine (and its
// ticker) must terminate shortly after ctx is cancelled.
func TestGenerateAndRefreshToken_StopsOnContextCancel(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFuncReturn(client.GetKubeClient, kubernetes.Interface(fakeClient))

	cahandler := certs.GetCAHandler(certs.CAHandlerTypeX509)
	pk, err := cahandler.GenPrivateKey()
	require.NoError(t, err)
	caPem, err := cahandler.NewSelfSigned(pk)
	require.NoError(t, err)

	hubconfig.Config.Ca = caPem.Bytes
	hubconfig.Config.CaKey = pk.DER()
	hubconfig.Config.CloudHub.TokenRefreshDuration = 12

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	before := runtime.NumGoroutine()

	err = GenerateAndRefreshToken(ctx)
	require.NoError(t, err)

	require.True(t, waitForGoroutineCount(func(n int) bool { return n > before }, time.Second),
		"refresh goroutine did not start")

	cancel()

	require.True(t, waitForGoroutineCount(func(n int) bool { return n <= before }, time.Second),
		"refresh goroutine did not exit after context cancellation")
}

// waitForGoroutineCount polls runtime.NumGoroutine synchronously (no
// extra goroutines of its own) until cond is satisfied or the deadline
// passes, since spawning a checker goroutine would itself skew the count.
func waitForGoroutineCount(cond func(n int) bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if cond(runtime.NumGoroutine()) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}
