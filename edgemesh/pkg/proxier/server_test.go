/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

-----------------------------------------------------------------------------
CHANGELOG
KubeEdge Authors:
- Remove useless functions
*/

package proxier

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	utilpointer "k8s.io/utils/pointer"

	kubeproxyconfig "k8s.io/kubernetes/pkg/proxy/apis/config"
	"k8s.io/kubernetes/pkg/util/configz"
)

// This test verifies that NewProxyServer does not crash when CleanupAndExit is true.
func TestProxyServerWithCleanupAndExit(t *testing.T) {
	// Each bind address below is a separate test case
	bindAddresses := []string{
		"0.0.0.0",
		"::",
	}
	for _, addr := range bindAddresses {
		options := NewOptions()

		options.config = &kubeproxyconfig.KubeProxyConfiguration{
			BindAddress: addr,
		}

		proxyserver, err := NewProxyServer(options)

		assert.Nil(t, err, "unexpected error in NewProxyServer, addr: %s", addr)
		assert.NotNil(t, proxyserver, "nil proxy server obj, addr: %s", addr)
		assert.NotNil(t, proxyserver.IptInterface, "nil iptables intf, addr: %s", addr)

		// Clean up config for next test case
		configz.Delete(kubeproxyconfig.GroupName)
	}
}

func TestGetConntrackMax(t *testing.T) {
	ncores := runtime.NumCPU()
	testCases := []struct {
		min        int32
		maxPerCore int32
		expected   int
		err        string
	}{
		{
			expected: 0,
		},
		{
			maxPerCore: 67890, // use this if Max is 0
			min:        1,     // avoid 0 default
			expected:   67890 * ncores,
		},
		{
			maxPerCore: 1, // ensure that Min is considered
			min:        123456,
			expected:   123456,
		},
		{
			maxPerCore: 0, // leave system setting
			min:        123456,
			expected:   0,
		},
	}

	for i, tc := range testCases {
		cfg := kubeproxyconfig.KubeProxyConntrackConfiguration{
			Min:        utilpointer.Int32Ptr(tc.min),
			MaxPerCore: utilpointer.Int32Ptr(tc.maxPerCore),
		}
		x, e := getConntrackMax(cfg)
		if e != nil {
			if tc.err == "" {
				t.Errorf("[%d] unexpected error: %v", i, e)
			} else if !strings.Contains(e.Error(), tc.err) {
				t.Errorf("[%d] expected an error containing %q: %v", i, tc.err, e)
			}
		} else if x != tc.expected {
			t.Errorf("[%d] expected %d, got %d", i, tc.expected, x)
		}
	}
}

// Run runs the specified ProxyServer.
func (s *fakeProxyServerLongRun) Run() error {
	for {
		time.Sleep(2 * time.Second)
	}
}

// CleanupAndExit runs in the specified ProxyServer.
func (s *fakeProxyServerLongRun) CleanupAndExit() error {
	return nil
}

type fakeProxyServerError struct{}

// Run runs the specified ProxyServer.
func (s *fakeProxyServerError) Run() error {
	for {
		time.Sleep(2 * time.Second)
		return fmt.Errorf("mocking error from ProxyServer.Run()")
	}
}

// CleanupAndExit runs in the specified ProxyServer.
func (s *fakeProxyServerError) CleanupAndExit() error {
	return errors.New("mocking error from ProxyServer.CleanupAndExit()")
}