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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

func TestValidateCloudCoreConfiguration(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_Dir")
	if err != nil {
		t.Errorf("create temp dir error %v", err)
		return
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "existFile")
	if err != nil {
		t.Errorf("create temp file failed: %v", err)
		return
	}

	config := v1alpha1.NewDefaultCloudCoreConfig()
	config.Modules.CloudHub.UnixSocket.Address = "unix://" + ef.Name()
	config.KubeAPIConfig.KubeConfig = ef.Name()

	errList := ValidateCloudCoreConfiguration(config)
	if len(errList) > 0 {
		t.Errorf("cloudcore configuration is not correct: %v", errList)
	}
}

func TestValidateModuleCloudHub(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_Dir")
	if err != nil {
		t.Errorf("create temp dir error %v", err)
		return
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "existFile")
	if err != nil {
		t.Errorf("create temp file failed: %v", err)
		return
	}
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
		if result := ValidateModuleCloudHub(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleEdgeController(t *testing.T) {
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
		if result := ValidateModuleEdgeController(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleDeviceController(t *testing.T) {
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
		if result := ValidateModuleDeviceController(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleSyncController(t *testing.T) {
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
		if result := ValidateModuleSyncController(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleDynamicController(t *testing.T) {
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
		if result := ValidateModuleDynamicController(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleCloudStream(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_Dir")
	if err != nil {
		t.Errorf("create temp dir error %v", err)
		return
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "existFile")
	if err != nil {
		t.Errorf("create temp file failed: %v", err)
		return
	}

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
		if result := ValidateModuleCloudStream(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateKubeAPIConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_Dir")
	if err != nil {
		t.Errorf("create temp dir error %v", err)
		return
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "existFile")
	if err != nil {
		t.Errorf("create temp file failed: %v", err)
		return
	}

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
		if result := ValidateKubeAPIConfig(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}
