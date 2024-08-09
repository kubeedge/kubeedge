/*
Copyright 2021 The KubeEdge Authors.

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

package validation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

func TestValidateCloudCoreConfiguration(t *testing.T) {
	assert := assert.New(t)

	dir := t.TempDir()
	ef, err := os.CreateTemp(dir, "existFile")
	assert.NoError(err)

	config := v1alpha1.NewDefaultCloudCoreConfig()
	config.Modules.CloudHub.UnixSocket.Address = "unix://" + ef.Name()
	config.KubeAPIConfig.KubeConfig = ef.Name()

	errList := ValidateCloudCoreConfiguration(config)
	assert.Empty(errList)
}

func TestValidateModuleCloudHub(t *testing.T) {
	assert := assert.New(t)
	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "existFile")
	assert.NoError(err)
	unixAddr := "unix://" + ef.Name()

	cases := []struct {
		name     string
		input    v1alpha1.CloudHub
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha1.CloudHub{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 invalid https port",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 0,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("port"), uint32(0), "must be between 1 and 65535, inclusive")},
		},
		{
			name: "case3 invalid websocket port",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    0,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("port"), uint32(0), "must be between 1 and 65535, inclusive")},
		},
		{
			name: "case4 invalid websocket addr",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "xxx.xxx.xxx.xxx",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("Address"), "xxx.xxx.xxx.xxx", "must be a valid IP address, (e.g. 10.9.8.7)")},
		},
		{
			name: "case5 invalid quic port",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    0,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("port"), uint32(0), "must be between 1 and 65535, inclusive")},
		},
		{
			name: "case6 invalid websocket addr",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "xxx.xxx.xxx.xxx",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("Address"), "xxx.xxx.xxx.xxx", "must be a valid IP address, (e.g. 10.9.8.7)")},
		},
		{
			name: "case7 invalid unixSocketAddress",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: "var/lib/kubeedge/kubeedge.sock",
				},
				TokenRefreshDuration: 1,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("address"),
				"var/lib/kubeedge/kubeedge.sock", "unixSocketAddress must has prefix unix://")},
		},
		{
			name: "case8 invalid TokenRefreshDuration",
			input: v1alpha1.CloudHub{
				Enable: true,
				HTTPS: &v1alpha1.CloudHubHTTPS{
					Port: 10000,
				},
				WebSocket: &v1alpha1.CloudHubWebSocket{
					Port:    10002,
					Address: "127.0.0.1",
				},
				Quic: &v1alpha1.CloudHubQUIC{
					Port:    10002,
					Address: "127.0.0.1",
				},
				UnixSocket: &v1alpha1.CloudHubUnixSocket{
					Address: unixAddr,
				},
				TokenRefreshDuration: 0,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("TokenRefreshDuration"),
				time.Duration(0), "TokenRefreshDuration must be positive")},
		},
	}

	for _, c := range cases {
		result := ValidateModuleCloudHub(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateModuleEdgeController(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name     string
		input    v1alpha1.EdgeController
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha1.EdgeController{
				Enable:              false,
				NodeUpdateFrequency: 0,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 NodeUpdateFrequency not legal",
			input: v1alpha1.EdgeController{
				Enable:              true,
				NodeUpdateFrequency: 0,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("NodeUpdateFrequency"), int32(0),
				"NodeUpdateFrequency need > 0")},
		},
		{
			name: "case3 all ok",
			input: v1alpha1.EdgeController{
				Enable:              true,
				NodeUpdateFrequency: 10,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateModuleEdgeController(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateModuleDeviceController(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name     string
		input    v1alpha1.DeviceController
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha1.DeviceController{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 all ok",
			input: v1alpha1.DeviceController{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateModuleDeviceController(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateModuleSyncController(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name     string
		input    v1alpha1.SyncController
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha1.SyncController{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 all ok",
			input: v1alpha1.SyncController{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateModuleSyncController(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateModuleDynamicController(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		name     string
		input    v1alpha1.DynamicController
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha1.DynamicController{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 all ok",
			input: v1alpha1.DynamicController{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateModuleDynamicController(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateModuleCloudStream(t *testing.T) {
	assert := assert.New(t)

	dir := t.TempDir()
	ef, err := os.CreateTemp(dir, "existFile")
	assert.NoError(err)

	nonexistentDir := filepath.Join(dir, "not_exist_dir")
	notExistFile := filepath.Join(nonexistentDir, "not_exist_file")

	cases := []struct {
		name     string
		input    v1alpha1.CloudStream
		expected field.ErrorList
	}{
		{
			name: "case1 not enable",
			input: v1alpha1.CloudStream{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 TLSStreamPrivateKeyFile not exist",
			input: v1alpha1.CloudStream{
				Enable:                  true,
				TLSStreamPrivateKeyFile: notExistFile,
				TLSStreamCertFile:       ef.Name(),
				TLSStreamCAFile:         ef.Name(),
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("TLSStreamPrivateKeyFile"), notExistFile,
				"TLSStreamPrivateKeyFile not exist")},
		},
		{
			name: "case3 TLSStreamCertFile not exist",
			input: v1alpha1.CloudStream{
				Enable:                  true,
				TLSStreamPrivateKeyFile: ef.Name(),
				TLSStreamCertFile:       notExistFile,
				TLSStreamCAFile:         ef.Name(),
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("TLSStreamCertFile"), notExistFile,
				"TLSStreamCertFile not exist")},
		},
		{
			name: "case4 TLSStreamCAFile not exist",
			input: v1alpha1.CloudStream{
				Enable:                  true,
				TLSStreamPrivateKeyFile: ef.Name(),
				TLSStreamCertFile:       ef.Name(),
				TLSStreamCAFile:         notExistFile,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("TLSStreamCAFile"), notExistFile,
				"TLSStreamCAFile not exist")},
		},
		{
			name: "case5 all ok",
			input: v1alpha1.CloudStream{
				Enable:                  true,
				TLSStreamPrivateKeyFile: ef.Name(),
				TLSStreamCertFile:       ef.Name(),
				TLSStreamCAFile:         ef.Name(),
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateModuleCloudStream(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateKubeAPIConfig(t *testing.T) {
	assert := assert.New(t)

	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "existFile")
	assert.NoError(err)
	defer ef.Close()

	nonexistentDir := filepath.Join(dir, "not_exist_dir")
	notExistFile := filepath.Join(nonexistentDir, "not_exist_file")

	cases := []struct {
		name     string
		input    v1alpha1.KubeAPIConfig
		expected field.ErrorList
	}{
		{
			name: "case1 not abs path",
			input: v1alpha1.KubeAPIConfig{
				KubeConfig: ".",
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("kubeconfig"), ".",
				"kubeconfig need abs path")},
		},
		{
			name: "case2 file not exist",
			input: v1alpha1.KubeAPIConfig{
				KubeConfig: notExistFile,
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("kubeconfig"), notExistFile,
				"kubeconfig not exist")},
		},
		{
			name: "case3 all ok",
			input: v1alpha1.KubeAPIConfig{
				KubeConfig: ef.Name(),
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		result := ValidateKubeAPIConfig(c.input)
		assert.Equal(c.expected, result, c.name)
	}
}

func TestValidateCommonConfig(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name         string
		commonConfig v1alpha1.CommonConfig
		expectedErr  bool
	}{
		{
			name: "invalid metric server addr",
			commonConfig: v1alpha1.CommonConfig{
				MonitorServer: v1alpha1.MonitorServer{
					BindAddress: "xxx.xxx.xxx.xxx:9091",
				},
			},
			expectedErr: true,
		},
		{
			name: "invalid metric server port",
			commonConfig: v1alpha1.CommonConfig{
				MonitorServer: v1alpha1.MonitorServer{
					BindAddress: "127.0.0.1:88888",
				},
			},
			expectedErr: true,
		},
		{
			name: "valid metric server config",
			commonConfig: v1alpha1.CommonConfig{
				MonitorServer: v1alpha1.MonitorServer{
					BindAddress: "127.0.0.1:9091",
				},
			},
			expectedErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errList := ValidateCommonConfig(tt.commonConfig)
			if tt.expectedErr {
				assert.NotEmpty(errList, "ValidateCommonConfig expected to get an error, but errList is empty")
			} else {
				assert.Empty(errList, "ValidateCommonConfig expected to get no error, but errList is not empty")
			}
		})
	}
}
