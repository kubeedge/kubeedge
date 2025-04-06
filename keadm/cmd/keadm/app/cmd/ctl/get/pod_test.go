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
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/cmd/get"
	api "k8s.io/kubernetes/pkg/apis/core"
	k8s_v1_api "k8s.io/kubernetes/pkg/apis/core/v1"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	ctlcommon "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

const (
	defaultNamespace = "default"
	testNodeName     = "test-node"
	testPodName      = "test-pod"
)

func setupTestEnvironment() (*PodGetOptions, *gomonkey.Patches) {
	podGetOptions := NewGetOpts()

	edgeCoreConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	edgeCoreConfig.Modules.Edged.HostnameOverride = testNodeName

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return edgeCoreConfig, nil
		})

	return podGetOptions, patches
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

func TestConvertDataToTable(t *testing.T) {
	podList := &api.PodList{
		Items: []api.Pod{
			{},
		},
	}

	result, err := ConvertDataToTable(podList)

	assert.NoError(t, err, "ConvertDataToTable should not return an error")
	assert.NotNil(t, result, "ConvertDataToTable should return a result")
}

func TestGetPodsErrorConfig(t *testing.T) {
	podGetOptions := NewGetOpts()

	patches := gomonkey.ApplyFunc(util.ParseEdgecoreConfig,
		func(configPath string) (*v1alpha2.EdgeCoreConfig, error) {
			return nil, errors.New("config parsing failed")
		})
	defer patches.Reset()

	err := podGetOptions.getPods([]string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get edge config failed")
}

func TestGetPodsNoResources(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPods",
		func(_ *client.PodRequest, _ context.Context) (*v1.PodList, error) {
			return &v1.PodList{Items: []v1.Pod{}}, nil
		})

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr, "Failed to create pipe")
	os.Stdout = w

	err := podGetOptions.getPods([]string{})

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No resources found in default namespace")
}

func TestGetPodsJSON(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()
	podGetOptions.Output = "json"

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPods",
		func(_ *client.PodRequest, _ context.Context) (*v1.PodList, error) {
			return &v1.PodList{
				Items: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testPodName,
							Namespace: defaultNamespace,
						},
						Spec: v1.PodSpec{
							NodeName: testNodeName,
						},
					},
				},
			}, nil
		})

	patches.ApplyMethod(reflect.TypeOf(&ctlcommon.ExtPrintFlags{}), "PrintToJSONYaml",
		func(_ *ctlcommon.ExtPrintFlags, _ interface{}) error {
			return nil
		})

	err := podGetOptions.getPods([]string{})

	assert.NoError(t, err)
}

func TestGetPodByName(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPod",
		func(_ *client.PodRequest, _ context.Context) (*v1.Pod, error) {
			return &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPodName,
					Namespace: defaultNamespace,
				},
				Spec: v1.PodSpec{
					NodeName: testNodeName,
				},
			}, nil
		})

	patches.ApplyFunc(k8s_v1_api.Convert_v1_Pod_To_core_Pod,
		func(_ *v1.Pod, apiPod *api.Pod, _ interface{}) error {
			apiPod.Name = testPodName
			apiPod.Namespace = defaultNamespace
			return nil
		})

	patches.ApplyFunc(ConvertDataToTable,
		func(_ interface{}) (interface{}, error) {
			return &metav1.Table{}, nil
		})

	patches.ApplyMethod(reflect.TypeOf(&ctlcommon.ExtPrintFlags{}), "PrintToTable",
		func(_ *ctlcommon.ExtPrintFlags, _ interface{}, _ bool, _ io.Writer) error {
			return nil
		})

	err := podGetOptions.getPods([]string{testPodName})

	assert.NoError(t, err)
}

func TestPodNotOnNode(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPod",
		func(_ *client.PodRequest, _ context.Context) (*v1.Pod, error) {
			return &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPodName,
					Namespace: defaultNamespace,
				},
				Spec: v1.PodSpec{
					NodeName: "other-node",
				},
			}, nil
		})

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr, "Failed to create pipe")
	os.Stdout = w

	err := podGetOptions.getPods([]string{testPodName})

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "can't to query pod")
}

func TestGetPodError(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPod",
		func(_ *client.PodRequest, _ context.Context) (*v1.Pod, error) {
			return nil, errors.New("pod not found")
		})

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr, "Failed to create pipe")
	os.Stdout = w

	err := podGetOptions.getPods([]string{testPodName})

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "pod not found")
}

func TestConversionError(t *testing.T) {
	podGetOptions, patches := setupTestEnvironment()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(&client.PodRequest{}), "GetPods",
		func(_ *client.PodRequest, _ context.Context) (*v1.PodList, error) {
			return &v1.PodList{
				Items: []v1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testPodName,
							Namespace: defaultNamespace,
						},
						Spec: v1.PodSpec{
							NodeName: testNodeName,
						},
					},
				},
			}, nil
		})

	patches.ApplyFunc(k8s_v1_api.Convert_v1_Pod_To_core_Pod,
		func(_ *v1.Pod, _ *api.Pod, _ interface{}) error {
			return errors.New("conversion failed")
		})

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr, "Failed to create pipe")
	os.Stdout = w

	err := podGetOptions.getPods([]string{})

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	assert.NoError(t, copyErr)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "pod revert to apiPod with err")
}
