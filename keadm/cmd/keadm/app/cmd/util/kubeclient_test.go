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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func newFakeKubeClient(t *testing.T, handler http.HandlerFunc) *kubernetes.Clientset {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cli, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	assert.NoError(t, err)
	return cli
}

func writeNamespace(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
	}
	assert.NoError(t, json.NewEncoder(w).Encode(ns))
}

func writeNotFound(w http.ResponseWriter, name string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	status := &metav1.Status{
		Status:  metav1.StatusFailure,
		Reason:  metav1.StatusReasonNotFound,
		Code:    http.StatusNotFound,
		Message: "namespaces \"" + name + "\" not found",
	}
	_ = json.NewEncoder(w).Encode(status)
}

func TestCleanNameSpace_WaitsForActualDeletion(t *testing.T) {
	origTimeout, origInterval := namespaceDeleteTimeout, namespaceDeletePollInterval
	namespaceDeleteTimeout, namespaceDeletePollInterval = 5*time.Second, 10*time.Millisecond
	defer func() { namespaceDeleteTimeout, namespaceDeletePollInterval = origTimeout, origInterval }()

	const ns = "kubeedge"
	var getCalls int

	cli := newFakeKubeClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			writeNamespace(t, w, ns)
		case http.MethodGet:
			getCalls++
			if getCalls < 3 {
				writeNamespace(t, w, ns)
				return
			}
			writeNotFound(w, ns)
		}
	})

	p := gomonkey.ApplyFunc(KubeClient, func(string) (*kubernetes.Clientset, error) {
		return cli, nil
	})
	defer p.Reset()

	co := &Common{}
	err := co.CleanNameSpace(ns, "fake/path")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 3)
}

func TestCleanNameSpace_TimesOutWhenStuckTerminating(t *testing.T) {
	origTimeout, origInterval := namespaceDeleteTimeout, namespaceDeletePollInterval
	namespaceDeleteTimeout, namespaceDeletePollInterval = 30*time.Millisecond, 10*time.Millisecond
	defer func() { namespaceDeleteTimeout, namespaceDeletePollInterval = origTimeout, origInterval }()

	const ns = "kubeedge"

	cli := newFakeKubeClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			writeNamespace(t, w, ns)
		case http.MethodGet:
			writeNamespace(t, w, ns)
		}
	})

	p := gomonkey.ApplyFunc(KubeClient, func(string) (*kubernetes.Clientset, error) {
		return cli, nil
	})
	defer p.Reset()

	co := &Common{}
	err := co.CleanNameSpace(ns, "fake/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "was not fully terminated")
}

func TestCleanNameSpace_AlreadyDeleted(t *testing.T) {
	const ns = "kubeedge"
	var getCalls int

	cli := newFakeKubeClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			writeNotFound(w, ns)
		case http.MethodGet:
			getCalls++
			writeNotFound(w, ns)
		}
	})

	p := gomonkey.ApplyFunc(KubeClient, func(string) (*kubernetes.Clientset, error) {
		return cli, nil
	})
	defer p.Reset()

	co := &Common{}
	err := co.CleanNameSpace(ns, "fake/path")
	assert.NoError(t, err)
	assert.Zero(t, getCalls)
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
