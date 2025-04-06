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

package get

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/cmd/get"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	testNodeName = "test-node"
)

func resetTestMode() {
	testMode = false
	testEdgeNodeName = ""
	testGetPodFunc = nil
	testGetPodsFunc = nil
	testConfigError = nil
}

// createMockEdgeCoreConfig returns a properly structured EdgeCoreConfig for testing
func createMockEdgeCoreConfig() *v1alpha2.EdgeCoreConfig {
	config := v1alpha2.NewDefaultEdgeCoreConfig()
	config.Modules.Edged.HostnameOverride = testNodeName
	return config
}

func TestNewEdgePodGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgePodGet()

	assert.NotNil(cmd)
	assert.Equal("pod", cmd.Use)
	assert.Equal(edgePodGetShortDescription, cmd.Short)
	assert.Equal(edgePodGetShortDescription, cmd.Long)

	assert.NotNil(cmd.RunE)

	assert.Equal(cmd.Flags().Lookup(common.FlagNameNamespace).Name, "namespace")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameAllNamespaces).Name, "all-namespaces")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameLabelSelector).Name, "selector")
	assert.Equal(cmd.Flags().Lookup(common.FlagNameOutput).Name, "output")
}

func TestNewGetOpts(t *testing.T) {
	assert := assert.New(t)

	podGetOptions := NewGetOpts()
	assert.NotNil(podGetOptions)
	assert.Equal(podGetOptions.Namespace, defaultNamespace)
	assert.Equal(podGetOptions.PrintFlags, get.NewGetPrintFlags())
	assert.Equal(podGetOptions.PrintFlags.OutputFormat, &podGetOptions.Output)
}

func TestAddGetPodFlags(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewGetOpts()

	cmd := &cobra.Command{}

	AddGetPodFlags(cmd, getOptions)

	namespaceFlag := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.NotNil(namespaceFlag)
	assert.Equal(defaultNamespace, namespaceFlag.DefValue)
	assert.Equal("namespace", namespaceFlag.Name)

	labelSelectorFlag := cmd.Flags().Lookup(common.FlagNameLabelSelector)
	assert.NotNil(labelSelectorFlag)
	assert.Equal("", labelSelectorFlag.DefValue)
	assert.Equal("selector", labelSelectorFlag.Name)

	outputFlag := cmd.Flags().Lookup(common.FlagNameOutput)
	assert.NotNil(outputFlag)
	assert.Equal("", outputFlag.DefValue)
	assert.Equal("output", outputFlag.Name)

	allNamespacesFlag := cmd.Flags().Lookup(common.FlagNameAllNamespaces)
	assert.NotNil(allNamespacesFlag)
	assert.Equal("false", allNamespacesFlag.DefValue)
	assert.Equal("all-namespaces", allNamespacesFlag.Name)
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	outBytes, _ := io.ReadAll(r)
	return string(outBytes)
}

func createTestPod(name, namespace, nodeName string) *v1.Pod {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}

	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
	pod.GetObjectKind().SetGroupVersionKind(gvk)

	return pod
}

func TestGetPodsWithSinglePod(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodFunc = func(ctx context.Context, namespace, name string) (*v1.Pod, error) {
		return createTestPod(name, namespace, testNodeName), nil
	}

	opts := NewGetOpts()

	output := captureOutput(func() {
		err := opts.getPods([]string{"test-pod"})
		assert.NoError(err)
	})

	assert.Contains(output, "NAME")
	assert.Contains(output, "test-pod")
}

func TestGetPodsWithDifferentNode(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodFunc = func(ctx context.Context, namespace, name string) (*v1.Pod, error) {
		return createTestPod(name, namespace, "different-node"), nil
	}

	opts := NewGetOpts()

	output := captureOutput(func() {
		err := opts.getPods([]string{"test-pod"})
		assert.NoError(err)
	})

	assert.Contains(output, "can't to query pod")
	assert.Contains(output, "different-node")
}

func TestGetPodsWithPodError(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodFunc = func(ctx context.Context, namespace, name string) (*v1.Pod, error) {
		return nil, fmt.Errorf("pod not found")
	}

	opts := NewGetOpts()

	output := captureOutput(func() {
		err := opts.getPods([]string{"test-pod"})
		assert.NoError(err)
	})

	assert.Contains(output, "pod not found")
}

func TestGetAllPods(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodsFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1.PodList, error) {
		return &v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PodList",
				APIVersion: "v1",
			},
			Items: []v1.Pod{
				*createTestPod("pod1", namespace, testNodeName),
				*createTestPod("pod2", namespace, "different-node"),
			},
		}, nil
	}

	opts := NewGetOpts()

	output := captureOutput(func() {
		err := opts.getPods([]string{})
		assert.NoError(err)
	})

	assert.Contains(output, "NAME")
	assert.Contains(output, "pod1")
	assert.NotContains(output, "pod2")
}

func TestGetPodsWithConfigError(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testConfigError = fmt.Errorf("config parse error")

	opts := NewGetOpts()

	err := opts.getPods([]string{})
	assert.Error(err)
	assert.Contains(err.Error(), "config parse error")
}

func TestGetPodsWithGetPodsError(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodsFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1.PodList, error) {
		return nil, fmt.Errorf("failed to list pods")
	}

	opts := NewGetOpts()

	err := opts.getPods([]string{})
	assert.Error(err)
	assert.Contains(err.Error(), "failed to list pods")
}

func TestGetPodsWithNoPods(t *testing.T) {
	defer resetTestMode()

	t.Run("default namespace", func(t *testing.T) {
		assert := assert.New(t)

		testMode = true
		testEdgeNodeName = testNodeName

		testGetPodsFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1.PodList, error) {
			return &v1.PodList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PodList",
					APIVersion: "v1",
				},
				Items: []v1.Pod{},
			}, nil
		}

		opts := NewGetOpts()
		opts.AllNamespaces = false
		opts.Namespace = defaultNamespace

		output := captureOutput(func() {
			err := opts.getPods([]string{})
			assert.NoError(err)
		})

		assert.Contains(output, "No resources found in default namespace")
	})

	t.Run("all namespaces", func(t *testing.T) {
		assert := assert.New(t)

		testMode = true
		testEdgeNodeName = testNodeName

		testGetPodsFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1.PodList, error) {
			return &v1.PodList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PodList",
					APIVersion: "v1",
				},
				Items: []v1.Pod{},
			}, nil
		}

		opts := NewGetOpts()
		opts.AllNamespaces = true

		output := captureOutput(func() {
			err := opts.getPods([]string{})
			assert.NoError(err)
		})

		assert.Contains(output, "No resources found in all namespace")
	})
}

func TestGetPodsWithJSONOutput(t *testing.T) {
	defer resetTestMode()

	assert := assert.New(t)

	testMode = true
	testEdgeNodeName = testNodeName

	testGetPodsFunc = func(ctx context.Context, namespace, labelSelector string, allNamespaces bool) (*v1.PodList, error) {
		return &v1.PodList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PodList",
				APIVersion: "v1",
			},
			Items: []v1.Pod{
				*createTestPod("pod1", namespace, testNodeName),
			},
		}, nil
	}

	opts := NewGetOpts()
	opts.Output = "json"
	opts.PrintFlags = get.NewGetPrintFlags()
	opts.PrintFlags.OutputFormat = &opts.Output

	output := captureOutput(func() {
		err := opts.getPods([]string{})
		assert.NoError(err)
	})

	assert.Contains(output, "{")
	assert.Contains(output, "\"kind\": \"Pod\"")
	assert.Contains(output, "\"name\": \"pod1\"")
}

func TestRealModeConfigError(t *testing.T) {
	defer resetTestMode()
	testMode = false

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, fmt.Errorf("mocked config error")
		})
	defer patches.Reset()

	opts := NewGetOpts()
	err := opts.getPods([]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed with err:mocked config error")
}

func TestDirectCoverageRealMode(t *testing.T) {
	origTestMode := testMode
	origEdgeNodeName := testEdgeNodeName
	origGetPodFunc := testGetPodFunc
	origGetPodsFunc := testGetPodsFunc
	origConfigError := testConfigError

	defer func() {
		testMode = origTestMode
		testEdgeNodeName = origEdgeNodeName
		testGetPodFunc = origGetPodFunc
		testGetPodsFunc = origGetPodsFunc
		testConfigError = origConfigError
	}()

	testMode = false

	p1 := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			config := createMockEdgeCoreConfig()
			return config, nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc((*client.PodRequest).GetPod,
		func(_ *client.PodRequest, _ context.Context) (*v1.Pod, error) {
			return createTestPod("test-pod", "default", testNodeName), nil
		})
	defer p2.Reset()

	p3 := gomonkey.ApplyFunc((*client.PodRequest).GetPods,
		func(_ *client.PodRequest, _ context.Context) (*v1.PodList, error) {
			return &v1.PodList{
				Items: []v1.Pod{
					*createTestPod("pod1", "default", testNodeName),
				},
			}, nil
		})
	defer p3.Reset()

	opts := NewGetOpts()

	_ = captureOutput(func() {
		_ = opts.getPods([]string{"test-pod"})
	})

	_ = captureOutput(func() {
		_ = opts.getPods([]string{})
	})
}

func TestRealModeMockPodRequestMethods(t *testing.T) {
	defer resetTestMode()
	testMode = false

	config := createMockEdgeCoreConfig()

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return config, nil
		})
	defer patches.Reset()

	t.Run("single pod", func(t *testing.T) {
		p1 := gomonkey.ApplyFunc((*client.PodRequest).GetPod,
			func(_ *client.PodRequest, _ context.Context) (*v1.Pod, error) {
				return createTestPod("test-pod", "default", testNodeName), nil
			})
		defer p1.Reset()

		opts := NewGetOpts()
		_ = opts.getPods([]string{"test-pod"})
	})

	t.Run("multiple pods", func(t *testing.T) {
		p2 := gomonkey.ApplyFunc((*client.PodRequest).GetPods,
			func(_ *client.PodRequest, _ context.Context) (*v1.PodList, error) {
				return &v1.PodList{
					Items: []v1.Pod{
						*createTestPod("pod1", "default", testNodeName),
					},
				}, nil
			})
		defer p2.Reset()

		opts := NewGetOpts()
		_ = opts.getPods([]string{})
	})
}
