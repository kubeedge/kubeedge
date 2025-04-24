/*
Copyright 2025 The KubeEdge Authors.

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

package util

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestKubeConfig(t *testing.T) {
	p1 := gomonkey.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
	defer p1.Reset()

	config, err := kubeConfig("fake/path")
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, float32(constants.DefaultKubeQPS), config.QPS)
	assert.Equal(t, int(constants.DefaultKubeBurst), config.Burst)
	assert.Equal(t, constants.DefaultKubeContentType, config.ContentType)

	p2 := gomonkey.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return nil, errors.New("mock error")
		})
	defer p2.Reset()

	config, err = kubeConfig("fake/path")
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestKubeClient(t *testing.T) {
	p1 := gomonkey.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
	defer p1.Reset()

	mockClientset := &kubernetes.Clientset{}
	p2 := gomonkey.ApplyFunc(kubernetes.NewForConfig,
		func(c *rest.Config) (*kubernetes.Clientset, error) {
			return mockClientset, nil
		})
	defer p2.Reset()

	client, err := KubeClient("fake/path")
	assert.NoError(t, err)
	assert.Equal(t, mockClientset, client)

	p3 := gomonkey.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return nil, errors.New("mock error")
		})
	defer p3.Reset()

	client, err = KubeClient("fake/path")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "get kube config failed")

	p4 := gomonkey.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
	defer p4.Reset()

	p5 := gomonkey.ApplyFunc(kubernetes.NewForConfig,
		func(c *rest.Config) (*kubernetes.Clientset, error) {
			return nil, errors.New("mock client error")
		})
	defer p5.Reset()

	client, err = KubeClient("fake/path")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "mock client error")
}

func TestCleanNameSpaceErrorPath(t *testing.T) {
	co := &Common{}

	p1 := gomonkey.ApplyFunc(KubeClient,
		func(kubeConfigPath string) (*kubernetes.Clientset, error) {
			return nil, errors.New("mock error")
		})
	defer p1.Reset()

	err := co.CleanNameSpace("test-namespace", "fake/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create KubeClient")
}

func TestIsCloudcoreContainerRunningErrorPath(t *testing.T) {
	p1 := gomonkey.ApplyFunc(KubeClient,
		func(kubeConfigPath string) (*kubernetes.Clientset, error) {
			return nil, errors.New("mock error")
		})
	defer p1.Reset()

	running, err := IsCloudcoreContainerRunning("test-namespace", "fake/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create KubeClient")
	assert.False(t, running)
}
