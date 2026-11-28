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

package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cloudcoreConfig "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
)

type MockRESTMapper struct {
	metav1.RESTMapper
}

func createTempKubeConfig(t *testing.T, server string) string {
	t.Helper()

	tempDir := t.TempDir()
	kubeConfigPath := filepath.Join(tempDir, "kubeconfig")
	kubeConfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: local
contexts:
- context:
    cluster: local
    user: test-user
  name: test
current-context: test
users:
- name: test-user
  user:
    token: test-token
`, server)

	err := os.WriteFile(kubeConfigPath, []byte(kubeConfigContent), 0644)
	assert.NoError(t, err)

	return kubeConfigPath
}

func TestSyncOncePattern(t *testing.T) {
	var localOnce sync.Once

	firstCallExecuted := false
	localOnce.Do(func() {
		firstCallExecuted = true
	})
	assert.True(t, firstCallExecuted, "First call to Once.Do should execute the function")

	secondCallExecuted := false
	localOnce.Do(func() {
		secondCallExecuted = true
	})
	assert.False(t, secondCallExecuted, "Second call to Once.Do should not execute the function")
}

func TestInitKubeEdgeClient(t *testing.T) {
	origKubeClient := kubeClient
	origCrdClient := crdClient
	origDynamicClient := dynamicClient
	origKubeConfig := KubeConfig
	origCrdConfig := CrdConfig

	kubeClient = nil
	crdClient = nil
	dynamicClient = nil
	KubeConfig = nil
	CrdConfig = nil

	initCount := 0
	initFunc := func() {
		initCount++
	}

	defer func() {
		kubeClient = origKubeClient
		crdClient = origCrdClient
		dynamicClient = origDynamicClient
		KubeConfig = origKubeConfig
		CrdConfig = origCrdConfig

		if r := recover(); r != nil {
			t.Logf("Recovered from panic in TestInitKubeEdgeClient: %v", r)
		}
	}()

	var testOnce sync.Once
	testOnce.Do(initFunc)
	assert.Equal(t, 1, initCount, "initFunc should be called once")
	testOnce.Do(initFunc)
	assert.Equal(t, 1, initCount, "initFunc should not be called again")

	testConfig := &cloudcoreConfig.KubeAPIConfig{
		Master:     "https://invalid-host:8443",
		KubeConfig: "invalid-config",
		QPS:        100,
		Burst:      200,
	}

	func() {
		defer func() {
			_ = recover()
		}()

		InitKubeEdgeClient(testConfig, false)
	}()
}

func TestInitKubeEdgeClientSuccessAndOnce(t *testing.T) {
	origInitOnce := initOnce
	origKubeClient := kubeClient
	origCrdClient := crdClient
	origDynamicClient := dynamicClient
	origKubeConfig := KubeConfig
	origCrdConfig := CrdConfig

	defer func() {
		initOnce = origInitOnce
		kubeClient = origKubeClient
		crdClient = origCrdClient
		dynamicClient = origDynamicClient
		KubeConfig = origKubeConfig
		CrdConfig = origCrdConfig
	}()

	initOnce = sync.Once{}
	kubeClient = nil
	crdClient = nil
	dynamicClient = nil
	KubeConfig = nil
	CrdConfig = nil

	kubeConfigPath := createTempKubeConfig(t, "http://localhost:6443")
	config := &cloudcoreConfig.KubeAPIConfig{
		Master:     "",
		KubeConfig: kubeConfigPath,
		QPS:        123,
		Burst:      456,
	}

	assert.NotPanics(t, func() {
		InitKubeEdgeClient(config, false)
	})

	assert.NotNil(t, GetKubeClient())
	assert.NotNil(t, GetCRDClient())
	assert.NotNil(t, GetDynamicClient())
	assert.NotNil(t, KubeConfig)
	assert.NotNil(t, CrdConfig)
	assert.Equal(t, float32(123), KubeConfig.QPS)
	assert.Equal(t, 456, KubeConfig.Burst)
	assert.Equal(t, runtime.ContentTypeProtobuf, KubeConfig.ContentType)
	assert.Equal(t, runtime.ContentTypeJSON, CrdConfig.ContentType)

	firstKubeConfig := KubeConfig
	assert.NotPanics(t, func() {
		InitKubeEdgeClient(&cloudcoreConfig.KubeAPIConfig{KubeConfig: "non-existent"}, false)
	})
	assert.Same(t, firstKubeConfig, KubeConfig)
}

func TestGetClientFunctions(t *testing.T) {
	origKubeClient := kubeClient
	origCrdClient := crdClient
	origDynamicClient := dynamicClient

	defer func() {
		kubeClient = origKubeClient
		crdClient = origCrdClient
		dynamicClient = origDynamicClient
	}()

	kubeClient = nil
	crdClient = nil
	dynamicClient = nil

	assert.Nil(t, GetKubeClient(), "GetKubeClient should return nil when kubeClient is nil")
	assert.Nil(t, GetCRDClient(), "GetCRDClient should return nil when crdClient is nil")
	assert.Nil(t, GetDynamicClient(), "GetDynamicClient should return nil when dynamicClient is nil")

	mockKubeClient := kubernetes.Interface(nil)
	mockCrdClient := crdClientset.Interface(nil)
	mockDynamicClient := dynamic.Interface(nil)

	kubeClient = mockKubeClient
	crdClient = mockCrdClient
	dynamicClient = mockDynamicClient

	assert.Equal(t, mockKubeClient, GetKubeClient(), "GetKubeClient should return the mockKubeClient")
	assert.Equal(t, mockCrdClient, GetCRDClient(), "GetCRDClient should return the mockCrdClient")
	assert.Equal(t, mockDynamicClient, GetDynamicClient(), "GetDynamicClient should return the mockDynamicClient")
}

func TestGetK8sCA(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "client-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validPEM := `-----BEGIN CERTIFICATE-----
MIICLDCCAdKgAwIBAgIBADAKBggqhkjOPQQDAjB8MQswCQYDVQQGEwJVUzETMBEG
A1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNjbzEYMBYGA1UE
ChMPS3ViZWVkZ2UgVGVzdGluZzEQMA4GA1UECxMHVGVzdGluZzEUMBIGA1UEAxML
ZXhhbXBsZS5jb20wHhcNMjUwMzEzMDAwMDAwWhcNMjYwMzEzMDAwMDAwWjB8MQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzEYMBYGA1UEChMPS3ViZWVkZ2UgVGVzdGluZzEQMA4GA1UECxMHVGVz
dGluZzEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAARnWWm5gZFNqh1JegATYgAKjHTj3Tc7jAqAkpHLK+LdnLH2gfTJ4gBvH6H1
IGJrQvVGzh3UgLZZ7B5CjrFuXJpAo0IwQDAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0T
AQH/BAUwAwEB/zAdBgNVHQ4EFgQUTyxdKLTWTzDN0mBKuYHJFVrGy2gwCgYIKoZI
zj0EAwIDSAAwRQIgayA3TybjNpJ+a5foC7laH8rZHjM8zLJ3g4Cq9umtPqkCIQDG
FZsaKg3OzLPQmGzYvRJYIkisF/zdqM4Tp562x9GKGQ==
-----END CERTIFICATE-----`

	caPath := filepath.Join(tempDir, "ca.crt")
	if err := os.WriteFile(caPath, []byte(validPEM), 0644); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	origKubeConfig := KubeConfig
	defer func() {
		KubeConfig = origKubeConfig
	}()

	KubeConfig = &rest.Config{
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: caPath,
		},
	}

	caData := GetK8sCA()
	assert.NotNil(t, caData, "CA data should not be nil")
	assert.Equal(t, []byte(validPEM), caData, "CA data should match test data")

	KubeConfig.TLSClientConfig.CAFile = "nonexistent-file"
	caData = GetK8sCA()
	assert.Nil(t, caData, "CA data should be nil for nonexistent file")
}

func TestGetRestMapper(t *testing.T) {
	origKubeConfig := KubeConfig
	origDefaultGetRestMapper := DefaultGetRestMapper

	defer func() {
		KubeConfig = origKubeConfig
		DefaultGetRestMapper = origDefaultGetRestMapper
	}()

	KubeConfig = &rest.Config{
		Host: "https://test-host:8443",
	}

	DefaultGetRestMapper = func() (metav1.RESTMapper, error) {
		return nil, fmt.Errorf("test error")
	}

	mapper, err := DefaultGetRestMapper()
	assert.Error(t, err, "Getting REST mapper should return error")
	assert.Nil(t, mapper, "REST mapper should be nil on error")
	assert.Contains(t, err.Error(), "test error", "Error message should contain 'test error'")

	DefaultGetRestMapper = func() (metav1.RESTMapper, error) {
		return &MockRESTMapper{}, nil
	}

	mapper, err = DefaultGetRestMapper()
	assert.NoError(t, err, "Getting REST mapper should not error")
	assert.NotNil(t, mapper, "REST mapper should not be nil")
	assert.IsType(t, &MockRESTMapper{}, mapper, "REST mapper should be of expected type")

	initialMapper := DefaultGetRestMapper
	DefaultGetRestMapper = GetRestMapper

	KubeConfig = &rest.Config{
		Host: "",
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte("invalid-ca-data"),
		},
	}

	_, err = DefaultGetRestMapper()
	assert.Error(t, err, "GetRestMapper should return error with invalid config")

	DefaultGetRestMapper = initialMapper
}

func TestGetRestMapperWithValidHTTPClient(t *testing.T) {
	origKubeConfig := KubeConfig
	defer func() {
		KubeConfig = origKubeConfig
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[]}`))
		case "/apis":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIGroupList","groups":[]}`))
		case "/api/v1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIResourceList","groupVersion":"v1","resources":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	KubeConfig = &rest.Config{
		Host: server.URL,
	}

	mapper, err := GetRestMapper()
	assert.NoError(t, err)
	assert.NotNil(t, mapper)
}
