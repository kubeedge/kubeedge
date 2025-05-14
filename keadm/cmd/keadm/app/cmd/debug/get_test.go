/*
Copyright 2020 The KubeEdge Authors.
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

package debug

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/beego/beego/v2/client/orm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const (
	testNamespace    = "test-namespace"
	testPodName      = "test-pod"
	testNodeName     = "test-node"
	testCMName       = "test-configmap"
	testSecretName   = "test-secret"
	testServiceName  = "test-service"
	testEndpointName = "test-endpoint"
)

func createTestPod(name, namespace, nodeName string) *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
		},
	}
}

func createTestNode(name string) *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"node-role.kubernetes.io/edge": "",
			},
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:   v1.NodeReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}
}

func createTestConfigMap(name, namespace string) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}
}

func createTestSecret(name, namespace string) *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret"),
		},
		Type: v1.SecretTypeOpaque,
	}
}

func createTestService(name, namespace string) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:     80,
					Protocol: "TCP",
				},
			},
			Selector: map[string]string{
				"app": "test",
			},
		},
	}
}

func createTestEndpoints(name, namespace string) *v1.Endpoints {
	return &v1.Endpoints{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Endpoints",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "192.168.1.1",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port: 80,
					},
				},
			},
		},
	}
}

func createTestMeta(key, value, resourceType string) dao.Meta {
	return dao.Meta{
		Key:   key,
		Value: value,
		Type:  resourceType,
	}
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(fmt.Sprintf("failed to create pipe: %v", err))
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var out []byte
	out, err = io.ReadAll(r)
	if err != nil {
		panic(fmt.Sprintf("failed to read from pipe: %v", err))
	}
	return string(out)
}

func setupFileExistMock(exists bool) *gomonkey.Patches {
	return gomonkey.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
		if exists {
			return nil, nil
		}
		return nil, os.ErrNotExist
	})
}
func TestNewCmdDebugGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCmdDebugGet()

	assert.NotNil(cmd)
	assert.Equal("get", cmd.Use)
	assert.Equal("Display one or many resources", cmd.Short)
	assert.Equal(debugGetLong, cmd.Long)
	assert.Equal(debugGetExample, cmd.Example)

	assert.NotNil(cmd.Run)

	getOption := NewGetOptions()

	flag := cmd.Flag("namespace")
	assert.NotNil(flag)
	assert.Equal(getOption.Namespace, flag.DefValue)
	assert.Equal("namespace", flag.Name)
	assert.Equal("n", flag.Shorthand)

	flag = cmd.Flag("output")
	assert.NotNil(flag)
	assert.Equal(*getOption.PrintFlags.OutputFormat, flag.DefValue)
	assert.Equal("output", flag.Name)
	assert.Equal("o", flag.Shorthand)

	flag = cmd.Flag("selector")
	assert.NotNil(flag)
	assert.Equal(getOption.LabelSelector, flag.DefValue)
	assert.Equal("selector", flag.Name)
	assert.Equal("l", flag.Shorthand)

	flag = cmd.Flag("edgedb-path")
	assert.NotNil(flag)
	assert.Equal(getOption.DataPath, flag.DefValue)
	assert.Equal("edgedb-path", flag.Name)
	assert.Equal("p", flag.Shorthand)

	flag = cmd.Flag("all-namespaces")
	assert.NotNil(flag)
	assert.Equal(getOption.AllNamespace, flag.DefValue == "true")
	assert.Equal("all-namespaces", flag.Name)
	assert.Equal("A", flag.Shorthand)
}

func TestAddGetOtherFlags(t *testing.T) {
	getOption := NewGetOptions()
	cmd := &cobra.Command{}

	addGetOtherFlags(cmd, getOption)

	assert := assert.New(t)

	flag := cmd.Flag("namespace")
	assert.NotNil(flag)
	assert.Equal(getOption.Namespace, flag.DefValue)
	assert.Equal("namespace", flag.Name)
	assert.Equal("n", flag.Shorthand)

	flag = cmd.Flag("output")
	assert.NotNil(flag)
	assert.Equal(*getOption.PrintFlags.OutputFormat, flag.DefValue)
	assert.Equal("output", flag.Name)
	assert.Equal("o", flag.Shorthand)

	flag = cmd.Flag("selector")
	assert.NotNil(flag)
	assert.Equal(getOption.LabelSelector, flag.DefValue)
	assert.Equal("selector", flag.Name)
	assert.Equal("l", flag.Shorthand)

	flag = cmd.Flag("edgedb-path")
	assert.NotNil(flag)
	assert.Equal(getOption.DataPath, flag.DefValue)
	assert.Equal("edgedb-path", flag.Name)
	assert.Equal("p", flag.Shorthand)

	flag = cmd.Flag("all-namespaces")
	assert.NotNil(flag)
	assert.Equal(getOption.AllNamespace, flag.DefValue == "true")
	assert.Equal("all-namespaces", flag.Name)
	assert.Equal("A", flag.Shorthand)
}

func TestIsAllowedFormat(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewGetOptions()

	tests := []struct {
		format   string
		expected bool
	}{
		{
			"yaml",
			true,
		},
		{
			"json",
			true,
		},
		{
			"wide",
			true,
		},
		{
			"xml",
			false,
		},
		{
			"",
			false,
		},
		{
			"plain",
			false,
		},
	}

	for _, test := range tests {
		stdResult := getOptions.IsAllowedFormat(test.format)
		assert.Equal(test.expected, stdResult)
	}
}

func TestSplitSelectorParameters(t *testing.T) {
	assert := assert.New(t)
	type args struct {
		args string
	}
	tests := []struct {
		name    string
		args    args
		want    []Selector
		wantErr bool
	}{
		{
			name: "testWithAllLabels",
			args: args{args: "key1==value1,key2!=value2,key3=value3"},
			want: []Selector{
				{Key: "key1", Value: "value1", Exist: true},
				{Key: "key2", Value: "value2", Exist: false},
				{Key: "key3", Value: "value3", Exist: true},
			},
			wantErr: false,
		},
		{
			name:    "testWithoutLabel",
			args:    args{args: "key1"},
			want:    []Selector{},
			wantErr: false,
		},
		{
			name:    "testWithEmptyValue",
			args:    args{args: "key1!="},
			want:    []Selector{{Key: "key1", Value: "", Exist: false}},
			wantErr: false,
		},
		{
			name:    "testWithMoreThanOneLabel",
			args:    args{args: "key1=value1=,key2=value2"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testEmptyString",
			args:    args{args: ""},
			want:    []Selector{},
			wantErr: false,
		},
		{
			name:    "testOnlyCommas",
			args:    args{args: ",,,"},
			want:    []Selector{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitSelectorParameters(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitSelectorParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestIsExistName(t *testing.T) {
	tests := []struct {
		name     string
		resNames []string
		key      string
		expected bool
	}{
		{
			name:     "Key exists in resNames",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "pod1",
			expected: true,
		},
		{
			name:     "Key does not exist in resNames",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "pod4",
			expected: false,
		},
		{
			name:     "Empty resNames",
			resNames: []string{},
			key:      "pod1",
			expected: false,
		},
		{
			name:     "Empty key",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "",
			expected: false,
		},
		{
			name:     "Key is substring of one element in resNames",
			resNames: []string{"pod123", "pod2", "pod3"},
			key:      "pod1",
			expected: false,
		},
		{
			name:     "ResNames contains special characters",
			resNames: []string{"pod@123", "pod#2", "pod$3"},
			key:      "pod@123",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isExistName(test.resNames, test.key)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestValidate(t *testing.T) {
	patches1 := gomonkey.ApplyFunc(os.Stat, func(path string) (os.FileInfo, error) {
		return nil, nil
	})
	defer patches1.Reset()

	patches2 := gomonkey.ApplyFunc(InitDB, func(driverName, dbName, dataSource string) error {
		return nil
	})
	defer patches2.Reset()

	tests := []struct {
		name    string
		args    []string
		options *GetOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no args",
			args:    []string{},
			options: NewGetOptions(),
			wantErr: true,
			errMsg:  "you must specify the type of resource to get",
		},
		{
			name:    "invalid resource type",
			args:    []string{"invalidtype"},
			options: NewGetOptions(),
			wantErr: true,
			errMsg:  "unrecognized resource type",
		},
		{
			name:    "all with additional args",
			args:    []string{"all", "extra"},
			options: NewGetOptions(),
			wantErr: true,
			errMsg:  "you must specify only one resource",
		},
		{
			name: "invalid output format",
			args: []string{"pod"},
			options: func() *GetOptions {
				opts := NewGetOptions()
				format := "invalid"
				opts.PrintFlags.OutputFormat = &format
				return opts
			}(),
			wantErr: true,
			errMsg:  "invalid output format",
		},
		{
			name: "invalid database path",
			args: []string{"pod"},
			options: func() *GetOptions {
				opts := NewGetOptions()
				opts.DataPath = "/nonexistent/path"
				return opts
			}(),
			wantErr: true,
			errMsg:  "not exist",
		},
		{
			name:    "database init error",
			args:    []string{"pod"},
			options: NewGetOptions(),
			wantErr: true,
			errMsg:  "failed to initialize database",
		},
		{
			name:    "valid args and options",
			args:    []string{"pod"},
			options: NewGetOptions(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "invalid database path" {
				p := gomonkey.ApplyFunc(os.Stat, func(path string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				})
				defer p.Reset()
			}

			if tt.name == "database init error" {
				p := gomonkey.ApplyFunc(InitDB, func(driverName, dbName, dataSource string) error {
					return errors.New("failed to initialize database")
				})
				defer p.Reset()
			}

			if tt.name == "valid args and options" {
				p1 := setupFileExistMock(true)
				defer p1.Reset()

				p2 := gomonkey.ApplyFunc(InitDB, func(driverName, dbName, dataSource string) error {
					return nil
				})
				defer p2.Reset()
			}

			err := tt.options.Validate(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckErr(t *testing.T) {
	assert := assert.New(t)

	mockHandler := func(msg string, exitCode int) {
		t.Error("handleErr should not be called for nil error")
	}
	CheckErr(nil, mockHandler)

	expectedMsg := "Test error"
	expectedExitCode := DefaultErrorExitCode
	expectedErr := errors.New(expectedMsg)

	var handledMsg string
	var handledExitCode int

	mockHandler = func(msg string, exitCode int) {
		handledMsg = msg
		handledExitCode = exitCode
	}

	CheckErr(expectedErr, mockHandler)

	assert.Equal(expectedMsg, handledMsg)
	assert.Equal(expectedExitCode, handledExitCode)
}

func TestFatal(t *testing.T) {
	patches := gomonkey.ApplyFunc(os.Exit, func(code int) {
		assert.Equal(t, 1, code)
	})
	defer patches.Reset()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	fatal("test error", 1)

	w.Close()
	os.Stderr = oldStderr

	var out []byte
	out, err = io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}
	assert.Contains(t, string(out), "test error")
}

func TestRunMethod(t *testing.T) {
	patchInitDB := gomonkey.ApplyFunc(InitDB, func(driverName, dbName, dataSource string) error {
		return nil
	})
	defer patchInitDB.Reset()

	patchFileExist := setupFileExistMock(true)
	defer patchFileExist.Reset()

	testPod := createTestPod(testPodName, testNamespace, testNodeName)
	podJSON, err := json.Marshal(testPod)
	if err != nil {
		t.Fatalf("failed to marshal testPod: %v", err)
	}

	t.Run("Empty results", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc((*GetOptions).queryDataFromDatabase,
			func(_ *GetOptions, _ string, _ []string) ([]dao.Meta, error) {
				return []dao.Meta{}, nil
			})

		g := NewGetOptions()
		output := captureOutput(func() {
			err := g.Run([]string{"pod"})
			assert.NoError(t, err)
		})

		assert.Contains(t, output, "No resources found")
	})

	t.Run("Error querying data", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc((*GetOptions).queryDataFromDatabase,
			func(_ *GetOptions, _ string, _ []string) ([]dao.Meta, error) {
				return nil, errors.New("database error")
			})

		g := NewGetOptions()
		err := g.Run([]string{"pod"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("Label selector error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc((*GetOptions).queryDataFromDatabase,
			func(_ *GetOptions, _ string, _ []string) ([]dao.Meta, error) {
				return []dao.Meta{
					{
						Key:   "test-key",
						Value: string(podJSON),
						Type:  model.ResourceTypePod,
					},
				}, nil
			})

		patches.ApplyFunc(FilterSelector,
			func(_ []dao.Meta, _ string) ([]dao.Meta, error) {
				return nil, errors.New("selector error")
			})

		g := NewGetOptions()
		g.LabelSelector = "app=test"

		err := g.Run([]string{"pod"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "selector error")
	})

	t.Run("Successful run with JSON/YAML output", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc((*GetOptions).queryDataFromDatabase,
			func(_ *GetOptions, _ string, _ []string) ([]dao.Meta, error) {
				podValue, err := json.Marshal(testPod)
				if err != nil {
					t.Fatalf("failed to marshal testPod: %v", err)
				}
				return []dao.Meta{
					{
						Key:   "test-key",
						Value: string(podValue),
						Type:  model.ResourceTypePod,
					},
				}, nil
			})

		patches.ApplyFunc((*PrintFlags).EnsureWithNamespace,
			func(_ *PrintFlags) error {
				return nil
			})

		patches.ApplyFunc((*PrintFlags).ToPrinter,
			func(_ *PrintFlags) (printers.ResourcePrinter, error) {
				return &mockResourcePrinter{}, nil
			})

		patches.ApplyFunc(JSONYamlPrint,
			func(_ []dao.Meta, _ printers.ResourcePrinter) error {
				return nil
			})

		g := NewGetOptions()
		format := "json"
		g.PrintFlags.OutputFormat = &format

		err := g.Run([]string{"pod"})
		assert.NoError(t, err)
	})
}

func TestJSONYamlPrint(t *testing.T) {
	testPod := createTestPod(testPodName, testNamespace, testNodeName)
	podJSON, err := json.Marshal(testPod)
	if err != nil {
		t.Fatalf("failed to marshal testPod: %v", err)
	}

	testCases := []struct {
		name       string
		results    []dao.Meta
		setupMocks func() *gomonkey.Patches
		expectErr  bool
	}{
		{
			name: "Single resource",
			results: []dao.Meta{
				{
					Key:   "pod-key",
					Value: string(podJSON),
					Type:  model.ResourceTypePod,
				},
			},
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToV1List, func(results []dao.Meta) ([]runtime.Object, error) {
					return []runtime.Object{testPod}, nil
				})

				patches.ApplyFunc(PrintGeneric, func(printer printers.ResourcePrinter, obj runtime.Object) error {
					return nil
				})

				return patches
			},
			expectErr: false,
		},
		{
			name: "ParseMetaToV1List error",
			results: []dao.Meta{
				{
					Key:   "pod-key",
					Value: "{invalid json}",
					Type:  model.ResourceTypePod,
				},
			},
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToV1List, func(results []dao.Meta) ([]runtime.Object, error) {
					return nil, errors.New("parsing error")
				})

				return patches
			},
			expectErr: true,
		},
		{
			name: "PrintGeneric error",
			results: []dao.Meta{
				{
					Key:   "pod-key",
					Value: string(podJSON),
					Type:  model.ResourceTypePod,
				},
			},
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToV1List, func(results []dao.Meta) ([]runtime.Object, error) {
					return []runtime.Object{testPod}, nil
				})

				patches.ApplyFunc(PrintGeneric, func(printer printers.ResourcePrinter, obj runtime.Object) error {
					return errors.New("print error")
				})

				return patches
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := tc.setupMocks()
			defer patches.Reset()

			mockPrinter := &mockResourcePrinter{}

			err := JSONYamlPrint(tc.results, mockPrinter)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHumanReadablePrint(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func() *gomonkey.Patches
		expectErr  bool
	}{
		{
			name: "Successful printing",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToAPIList, func(metas []dao.Meta) ([]runtime.Object, error) {
					return []runtime.Object{&v1.PodList{}}, nil
				})

				patches.ApplyFunc(ConvertDataToTable, func(obj runtime.Object) (runtime.Object, error) {
					return &metav1.Table{}, nil
				})

				return patches
			},
			expectErr: false,
		},
		{
			name: "Error in ParseMetaToAPIList",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToAPIList, func(metas []dao.Meta) ([]runtime.Object, error) {
					return nil, errors.New("parse error")
				})

				return patches
			},
			expectErr: true,
		},
		{
			name: "Error in ConvertDataToTable",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToAPIList, func(metas []dao.Meta) ([]runtime.Object, error) {
					return []runtime.Object{&v1.PodList{}}, nil
				})

				patches.ApplyFunc(ConvertDataToTable, func(obj runtime.Object) (runtime.Object, error) {
					return nil, errors.New("conversion error")
				})

				return patches
			},
			expectErr: true,
		},
		{
			name: "Error in PrintObj",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(ParseMetaToAPIList, func(metas []dao.Meta) ([]runtime.Object, error) {
					return []runtime.Object{&v1.PodList{}}, nil
				})

				patches.ApplyFunc(ConvertDataToTable, func(obj runtime.Object) (runtime.Object, error) {
					return &metav1.Table{}, nil
				})

				return patches
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := tc.setupMocks()
			defer patches.Reset()

			mockPrinter := &mockResourcePrinter{
				errorOnPrint: tc.name == "Error in PrintObj",
			}

			err := HumanReadablePrint([]dao.Meta{}, mockPrinter)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewGetOptions(t *testing.T) {
	options := NewGetOptions()

	assert.NotNil(t, options)
	assert.Equal(t, "default", options.Namespace)
	assert.NotEqual(t, "", options.DataPath)
	assert.NotNil(t, options.PrintFlags)
	assert.NotNil(t, options.PrintFlags.OutputFormat)
}

func TestPrintGeneric(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func() *gomonkey.Patches
		expectErr  bool
	}{
		{
			name: "Print non-list object",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(meta.IsListType, func(obj runtime.Object) bool {
					return false
				})

				patches.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
					return []byte("{}"), nil
				})

				patches.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
					return nil
				})

				return patches
			},
			expectErr: false,
		},
		{
			name: "Extract list error",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(meta.IsListType, func(obj runtime.Object) bool {
					return true
				})

				patches.ApplyFunc(meta.ExtractList, func(obj runtime.Object) ([]runtime.Object, error) {
					return nil, errors.New("extract error")
				})

				return patches
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := tc.setupMocks()
			defer patches.Reset()

			mockPrinter := &mockResourcePrinter{
				errorOnPrint: false,
			}

			testObj := &v1.Pod{}

			err := PrintGeneric(mockPrinter, testObj)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsAvailableResources(t *testing.T) {
	testCases := []struct {
		name     string
		resType  string
		expected bool
	}{
		{
			name:     "Available resource: pod",
			resType:  "pod",
			expected: true,
		},
		{
			name:     "Available resource: pods",
			resType:  "pods",
			expected: true,
		},
		{
			name:     "Available resource: po",
			resType:  "po",
			expected: true,
		},
		{
			name:     "Available resource: configmap",
			resType:  "configmap",
			expected: true,
		},
		{
			name:     "Available resource: service",
			resType:  "service",
			expected: true,
		},
		{
			name:     "Available resource: node",
			resType:  "node",
			expected: true,
		},
		{
			name:     "Available resource: all",
			resType:  "all",
			expected: true,
		},
		{
			name:     "Unavailable resource: deployment",
			resType:  "deployment",
			expected: false,
		},
		{
			name:     "Unavailable resource: empty string",
			resType:  "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isAvailableResources(tc.resType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInitDB(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func() *gomonkey.Patches
		expectErr  bool
	}{
		{
			name: "Successful initialization",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(orm.RegisterDriver, func(driverName string, typ orm.DriverType) error {
					return nil
				})

				patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...orm.DBOption) error {
					return nil
				})

				patches.ApplyFunc(orm.NewOrmUsingDB, func(aliasName string) orm.Ormer {
					return nil
				})

				return patches
			},
			expectErr: false,
		},
		{
			name: "Error registering driver",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(orm.RegisterDriver, func(driverName string, typ orm.DriverType) error {
					return errors.New("driver registration error")
				})

				return patches
			},
			expectErr: true,
		},
		{
			name: "Error registering database",
			setupMocks: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(orm.RegisterDriver, func(driverName string, typ orm.DriverType) error {
					return nil
				})

				patches.ApplyFunc(orm.RegisterDataBase, func(aliasName, driverName, dataSource string, params ...orm.DBOption) error {
					return errors.New("database registration error")
				})

				return patches
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := tc.setupMocks()
			defer patches.Reset()

			err := InitDB("sqlite3", "default", "/path/to/db")

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsFileExist(t *testing.T) {
	testCases := []struct {
		name     string
		filepath string
		mockFunc func(string) (os.FileInfo, error)
		expected bool
	}{
		{
			name:     "File exists",
			filepath: "/path/exists",
			mockFunc: func(path string) (os.FileInfo, error) {
				return nil, nil
			},
			expected: true,
		},
		{
			name:     "File does not exist - ErrNotExist",
			filepath: "/path/not/exists",
			mockFunc: func(path string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			expected: false,
		},
		{
			name:     "File does not exist - other error",
			filepath: "/path/error",
			mockFunc: func(path string) (os.FileInfo, error) {
				return nil, errors.New("other error")
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.Stat, tc.mockFunc)
			defer patches.Reset()

			result := isFileExist(tc.filepath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

type mockResourcePrinter struct {
	errorOnPrint bool
}

func (m *mockResourcePrinter) PrintObj(obj runtime.Object, w io.Writer) error {
	if m.errorOnPrint {
		return errors.New("print error")
	}
	return nil
}

func TestPrintFlagsAllowedFormats(t *testing.T) {
	printFlags := NewGetPrintFlags()

	formats := printFlags.AllowedFormats()
	assert.Contains(t, formats, "json")
	assert.Contains(t, formats, "yaml")
	assert.Contains(t, formats, "wide")
}

func TestPrintFlagsEnsureWithNamespace(t *testing.T) {
	printFlags := NewGetPrintFlags()

	err := printFlags.EnsureWithNamespace()
	assert.NoError(t, err)
}

func TestPrintFlagsToPrinter(t *testing.T) {
	printFlags := NewGetPrintFlags()

	patches := gomonkey.ApplyFunc(genericclioptions.NewJSONYamlPrintFlags, func() *genericclioptions.JSONYamlPrintFlags {
		return &genericclioptions.JSONYamlPrintFlags{}
	})
	defer patches.Reset()

	patches.ApplyFunc((*genericclioptions.JSONYamlPrintFlags).ToPrinter, func(_ *genericclioptions.JSONYamlPrintFlags) (printers.ResourcePrinter, error) {
		return &mockResourcePrinter{}, nil
	})

	printer, err := printFlags.ToPrinter()
	assert.NoError(t, err)
	assert.NotNil(t, printer)
}

func TestParseMetaToAPIList(t *testing.T) {
	testPod := createTestPod(testPodName, testNamespace, testNodeName)
	podJSON, err := json.Marshal(testPod)
	if err != nil {
		t.Fatalf("failed to marshal testPod: %v", err)
	}

	testCM := createTestConfigMap(testCMName, testNamespace)
	cmJSON, err := json.Marshal(testCM)
	if err != nil {
		t.Fatalf("failed to marshal testCM: %v", err)
	}

	testCases := []struct {
		name          string
		metas         []dao.Meta
		expectErr     bool
		expectedCount int
	}{
		{
			name: "Parse Pod and ConfigMap",
			metas: []dao.Meta{
				{
					Key:   "pod-key",
					Value: string(podJSON),
					Type:  model.ResourceTypePod,
				},
				{
					Key:   "cm-key",
					Value: string(cmJSON),
					Type:  model.ResourceTypeConfigmap,
				},
			},
			expectErr:     false,
			expectedCount: 6,
		},
		{
			name: "Error parsing invalid Pod JSON",
			metas: []dao.Meta{
				{
					Key:   "pod-key",
					Value: `{invalid json`,
					Type:  model.ResourceTypePod,
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := ParseMetaToAPIList(tc.metas)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, results, tc.expectedCount)
			}
		})
	}
}

func TestParseMetaToV1List(t *testing.T) {
	testPod := createTestPod(testPodName, testNamespace, testNodeName)
	podJSON, err := json.Marshal(testPod)
	if err != nil {
		t.Fatalf("failed to marshal testPod: %v", err)
	}

	testSvc := createTestService(testServiceName, testNamespace)
	svcJSON, err := json.Marshal(testSvc)
	if err != nil {
		t.Fatalf("failed to marshal testSvc: %v", err)
	}

	testSecret := createTestSecret(testSecretName, testNamespace)
	secretJSON, err := json.Marshal(testSecret)
	if err != nil {
		t.Fatalf("failed to marshal testSecret: %v", err)
	}

	testCM := createTestConfigMap(testCMName, testNamespace)
	cmJSON, err := json.Marshal(testCM)
	if err != nil {
		t.Fatalf("failed to marshal testCM: %v", err)
	}

	testEndpoints := createTestEndpoints(testEndpointName, testNamespace)
	epJSON, err := json.Marshal(testEndpoints)
	if err != nil {
		t.Fatalf("failed to marshal testEndpoints: %v", err)
	}

	testNode := createTestNode(testNodeName)
	nodeJSON, err := json.Marshal(testNode)
	if err != nil {
		t.Fatalf("failed to marshal testNode: %v", err)
	}

	testCases := []struct {
		name          string
		metas         []dao.Meta
		expectErr     bool
		expectedCount int
	}{
		{
			name: "Parse Pod metadata",
			metas: []dao.Meta{
				{
					Key:   "pod-key",
					Value: string(podJSON),
					Type:  model.ResourceTypePod,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse Service metadata",
			metas: []dao.Meta{
				{
					Key:   "svc-key",
					Value: string(svcJSON),
					Type:  constants.ResourceTypeService,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse Secret metadata",
			metas: []dao.Meta{
				{
					Key:   "secret-key",
					Value: string(secretJSON),
					Type:  model.ResourceTypeSecret,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse ConfigMap metadata",
			metas: []dao.Meta{
				{
					Key:   "cm-key",
					Value: string(cmJSON),
					Type:  model.ResourceTypeConfigmap,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse Endpoints metadata",
			metas: []dao.Meta{
				{
					Key:   "ep-key",
					Value: string(epJSON),
					Type:  constants.ResourceTypeEndpoints,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse Node metadata",
			metas: []dao.Meta{
				{
					Key:   "node-key",
					Value: string(nodeJSON),
					Type:  model.ResourceTypeNode,
				},
			},
			expectErr:     false,
			expectedCount: 1,
		},
		{
			name: "Parse invalid Pod JSON",
			metas: []dao.Meta{
				{
					Key:   "pod-key",
					Value: `{invalid json`,
					Type:  model.ResourceTypePod,
				},
			},
			expectErr: true,
		},
		{
			name: "Parse unknown resource type",
			metas: []dao.Meta{
				{
					Key:   "unknown-key",
					Value: "{}",
					Type:  "unknown",
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := ParseMetaToV1List(tc.metas)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, results, tc.expectedCount)
			}
		})
	}
}
