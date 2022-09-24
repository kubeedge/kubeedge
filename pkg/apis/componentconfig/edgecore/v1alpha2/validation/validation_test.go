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

package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func TestValidateEdgeCoreConfiguration(t *testing.T) {
	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "existFile")
	if err != nil {
		t.Errorf("create temp file failed: %v", err)
		return
	}

	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.DataBase.DataSource = ef.Name()

	errList := ValidateEdgeCoreConfiguration(config)
	if len(errList) > 0 {
		t.Errorf("configuration is not right: %v", errList)
	}
}

func TestValidateDataBase(t *testing.T) {
	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "FileIsExist")
	if err == nil {
		db := v1alpha2.DataBase{
			DataSource: ef.Name(),
		}
		if errs := ValidateDataBase(db); len(errs) > 0 {
			t.Errorf("file %v should exist: err is %v", db, errs)
		}
	}

	nonexistentDir := filepath.Join(dir, "not_exists_dir")
	nonexistentFile := filepath.Join(nonexistentDir, "not_exist_file")

	db := v1alpha2.DataBase{
		DataSource: nonexistentFile,
	}

	if errs := ValidateDataBase(db); len(errs) > 0 {
		t.Errorf("file %v should not created, err is %v", nonexistentFile, errs)
	}
}

func TestValidateModuleEdged(t *testing.T) {
	cases := []struct {
		name   string
		input  v1alpha2.Edged
		result field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.Edged{
				Enable: false,
			},
			result: field.ErrorList{},
		},
		{
			name: "case2 not right CGroupDriver",
			input: v1alpha2.Edged{
				Enable: true,
				TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
					HostnameOverride: "example.com",
				},
				TailoredKubeletConfig: &kubeletconfigv1beta1.KubeletConfiguration{
					CgroupDriver: "fake",
				},
			},
			result: field.ErrorList{field.Invalid(field.NewPath("CGroupDriver"), "fake",
				"CGroupDriver value error")},
		},
		{
			name: "case3 invalid hostname",
			input: v1alpha2.Edged{
				Enable: true,
				TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
					HostnameOverride: "Example%$#com",
				},
				TailoredKubeletConfig: &kubeletconfigv1beta1.KubeletConfiguration{
					CgroupDriver: v1alpha2.CGroupDriverCGroupFS,
				},
			},
			result: field.ErrorList{field.Invalid(field.NewPath("HostnameOverride"), "Example%$#com", `a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')`)},
		},
		{
			name: "case4 success",
			input: v1alpha2.Edged{
				Enable: true,
				TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
					HostnameOverride: "example.com",
				},
				TailoredKubeletConfig: &kubeletconfigv1beta1.KubeletConfiguration{
					CgroupDriver: v1alpha2.CGroupDriverCGroupFS,
				},
			},
			result: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if got := ValidateModuleEdged(c.input); !reflect.DeepEqual(got, c.result) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.result, got)
		}
	}
}

func TestValidateModuleEdgeHub(t *testing.T) {
	cases := []struct {
		name   string
		input  v1alpha2.EdgeHub
		result field.ErrorList
	}{
		{
			name: "case1 not enable",
			input: v1alpha2.EdgeHub{
				Enable: false,
			},
			result: field.ErrorList{},
		},
		{
			name: "case2 both quic and websocket are enabled",
			input: v1alpha2.EdgeHub{
				Enable: true,
				Quic: &v1alpha2.EdgeHubQUIC{
					Enable: true,
				},
				WebSocket: &v1alpha2.EdgeHubWebSocket{
					Enable: true,
				},
			},
			result: field.ErrorList{field.Invalid(field.NewPath("enable"),
				true, "websocket.enable and quic.enable cannot be true and false at the same time")},
		},
		{
			name: "case3 success",
			input: v1alpha2.EdgeHub{
				Enable: true,
				WebSocket: &v1alpha2.EdgeHubWebSocket{
					Enable: true,
				},
				Quic: &v1alpha2.EdgeHubQUIC{
					Enable: false,
				},
			},
			result: field.ErrorList{},
		},
		{
			name: "case4 MessageQPS must not be a negative number",
			input: v1alpha2.EdgeHub{
				Enable: true,
				WebSocket: &v1alpha2.EdgeHubWebSocket{
					Enable: true,
				},
				Quic: &v1alpha2.EdgeHubQUIC{
					Enable: false,
				},
				MessageQPS: -1,
			},
			result: field.ErrorList{field.Invalid(field.NewPath("messageQPS"),
				int32(-1), "MessageQPS must not be a negative number")},
		},
		{
			name: "case5 MessageBurst must not be a negative number",
			input: v1alpha2.EdgeHub{
				Enable: true,
				WebSocket: &v1alpha2.EdgeHubWebSocket{
					Enable: true,
				},
				Quic: &v1alpha2.EdgeHubQUIC{
					Enable: false,
				},
				MessageBurst: -1,
			},
			result: field.ErrorList{field.Invalid(field.NewPath("messageBurst"),
				int32(-1), "MessageBurst must not be a negative number")},
		},
	}

	for _, c := range cases {
		if got := ValidateModuleEdgeHub(c.input); !reflect.DeepEqual(got, c.result) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.result, got)
		}
	}
}

func TestValidateModuleEventBus(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.EventBus
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.EventBus{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 mqtt not right",
			input: v1alpha2.EventBus{
				Enable:   true,
				MqttMode: v1alpha2.MqttMode(3),
			},
			expected: field.ErrorList{field.Invalid(field.NewPath("Mode"), v1alpha2.MqttMode(3),
				fmt.Sprintf("Mode need in [%v,%v] range", v1alpha2.MqttModeInternal,
					v1alpha2.MqttModeExternal))},
		},
		{
			name: "case2 all ok",
			input: v1alpha2.EventBus{
				Enable:   true,
				MqttMode: 2,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleEventBus(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleMetaManager(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.MetaManager
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.MetaManager{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 enabled",
			input: v1alpha2.MetaManager{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleMetaManager(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleServiceBus(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.ServiceBus
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.ServiceBus{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 enabled",
			input: v1alpha2.ServiceBus{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleServiceBus(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleDeviceTwin(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.DeviceTwin
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.DeviceTwin{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 enabled",
			input: v1alpha2.DeviceTwin{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleDeviceTwin(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleDBTest(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.DBTest
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.DBTest{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 enabled",
			input: v1alpha2.DBTest{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleDBTest(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}

func TestValidateModuleEdgeStream(t *testing.T) {
	cases := []struct {
		name     string
		input    v1alpha2.EdgeStream
		expected field.ErrorList
	}{
		{
			name: "case1 not enabled",
			input: v1alpha2.EdgeStream{
				Enable: false,
			},
			expected: field.ErrorList{},
		},
		{
			name: "case2 enabled",
			input: v1alpha2.EdgeStream{
				Enable: true,
			},
			expected: field.ErrorList{},
		},
	}

	for _, c := range cases {
		if result := ValidateModuleEdgeStream(c.input); !reflect.DeepEqual(result, c.expected) {
			t.Errorf("%v: expected %v, but got %v", c.name, c.expected, result)
		}
	}
}
