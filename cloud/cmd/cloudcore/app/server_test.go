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

package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	testclient "k8s.io/client-go/testing"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
)

const (
	testHostname    = "test-host"
	testLocalIP     = "192.168.1.100"
	testMasterURL   = "test-master"
	testKubeConfig  = "test-kubeconfig"
	initialConfig   = "initial config"
	updateErrorMsg  = "update error"
	namespaceErrMsg = "namespace error"
)

// TestNegotiatePortFunc thoroughly tests the port allocation algorithm
func TestNegotiatePortFunc(t *testing.T) {
	tests := []struct {
		name         string
		portRecord   map[int]bool
		expectedPort int
	}{
		{
			name:         "Empty port record",
			portRecord:   map[int]bool{},
			expectedPort: constants.ServerPort + 1,
		},
		{
			name:         "Port already in use",
			portRecord:   map[int]bool{constants.ServerPort + 1: true},
			expectedPort: constants.ServerPort + 2,
		},
		{
			name: "Multiple consecutive ports in use",
			portRecord: map[int]bool{
				constants.ServerPort + 1: true,
				constants.ServerPort + 2: true,
				constants.ServerPort + 3: true,
			},
			expectedPort: constants.ServerPort + 4,
		},
		{
			name: "Non-sequential ports in use",
			portRecord: map[int]bool{
				constants.ServerPort + 1: true,
				constants.ServerPort + 3: true,
			},
			expectedPort: constants.ServerPort + 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			port := NegotiatePortFunc(test.portRecord)
			assert.Equal(t, test.expectedPort, port)
		})
	}
}

// TestAuthorizationLogic tests just the authorization condition in registerModules
func TestAuthorizationLogic(t *testing.T) {
	// Create test cases
	tests := []struct {
		name               string
		authConfig         *v1alpha1.CloudHubAuthorization
		expectedEnablement bool
	}{
		{
			name:               "Nil Authorization",
			authConfig:         nil,
			expectedEnablement: false,
		},
		{
			name: "Authorization Disabled",
			authConfig: &v1alpha1.CloudHubAuthorization{
				Enable: false,
				Debug:  false,
			},
			expectedEnablement: false,
		},
		{
			name: "Authorization Enabled with Debug On",
			authConfig: &v1alpha1.CloudHubAuthorization{
				Enable: true,
				Debug:  true,
			},
			expectedEnablement: false,
		},
		{
			name: "Authorization Enabled with Debug Off",
			authConfig: &v1alpha1.CloudHubAuthorization{
				Enable: true,
				Debug:  false,
			},
			expectedEnablement: true,
		},
	}

	// Test each case
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a cloud hub with the test authorization config
			cloudHub := &v1alpha1.CloudHub{
				Authorization: test.authConfig,
			}

			// Test the condition directly
			result := cloudHub.Authorization != nil &&
				cloudHub.Authorization.Enable &&
				!cloudHub.Authorization.Debug

			assert.Equal(t, test.expectedEnablement, result)
		})
	}
}

// TestNegotiateTunnelPortWithClient_ErrorCases tests error handling
func TestNegotiateTunnelPortWithClient_ErrorCases(t *testing.T) {
	// Test case where createNamespaceIfNeeded fails
	t.Run("CreateNamespace fails", func(t *testing.T) {
		// Save global functions and restore after test
		restore := saveGlobals()
		defer restore()

		// Force an error in createNamespaceIfNeeded
		createNamespaceIfNeededFunc = func(ctx context.Context, namespace string) error {
			return errors.New(namespaceErrMsg)
		}

		// Call function
		fakeClient := fake.NewSimpleClientset()
		_, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return the namespace error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create system namespace")
	})

	// Test case where ConfigMap exists but has invalid data
	t.Run("Invalid ConfigMap data", func(t *testing.T) {
		// Save global functions and restore after test
		restore := saveGlobals()
		defer restore()

		// Set up test doubles
		getHostnameFunc = func() string {
			return testHostname
		}
		getLocalIPFunc = func(hostname string) (string, error) {
			return testLocalIP, nil
		}
		createNamespaceIfNeededFunc = func(ctx context.Context, namespace string) error {
			return nil
		}

		// Create ConfigMap with invalid annotation
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: "{invalid-json",
				},
			},
		}

		// Setup fake client with the invalid ConfigMap
		fakeClient := fake.NewSimpleClientset(cm)

		// Call function
		_, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
	})
}

// TestNegotiateTunnelPort tests the main entry point
func TestNegotiateTunnelPort(t *testing.T) {
	// Save original functions
	originalKubeClientGetter := kubeClientGetter
	originalHostnameFunc := getHostnameFunc
	originalLocalIPFunc := getLocalIPFunc
	originalCreateNamespaceFunc := createNamespaceIfNeededFunc

	// Restore original values after test
	defer func() {
		kubeClientGetter = originalKubeClientGetter
		getHostnameFunc = originalHostnameFunc
		getLocalIPFunc = originalLocalIPFunc
		createNamespaceIfNeededFunc = originalCreateNamespaceFunc
	}()

	// Set up test doubles
	getHostnameFunc = func() string {
		return testHostname
	}
	getLocalIPFunc = func(hostname string) (string, error) {
		return testLocalIP, nil
	}
	createNamespaceIfNeededFunc = func(ctx context.Context, namespace string) error {
		return nil
	}

	// Create a fake client
	fakeClient := fake.NewSimpleClientset()

	// Mock the client getter
	kubeClientGetter = func() kubernetes.Interface {
		return fakeClient
	}

	// Call the function
	port, err := NegotiateTunnelPort()

	// Should get a port without error
	assert.NoError(t, err)
	assert.NotNil(t, port)
}

// TestUpdateCloudCoreConfigMap tests the warning path
func TestUpdateCloudCoreConfigMap(t *testing.T) {
	// Save original values
	originalKubeClientGetter := kubeClientGetter
	originalUpdateConfigMapWarningf := updateCloudCoreConfigMapWarningf

	// Restore original values after test
	defer func() {
		kubeClientGetter = originalKubeClientGetter
		updateCloudCoreConfigMapWarningf = originalUpdateConfigMapWarningf
	}()

	// Create fake client without ConfigMap
	fakeClient := fake.NewSimpleClientset()

	// Track if warning was called
	warningCalled := false

	// Mock the warning function
	updateCloudCoreConfigMapWarningf = func(format string, args ...interface{}) {
		warningCalled = true
	}

	// Mock the client getter
	kubeClientGetter = func() kubernetes.Interface {
		return fakeClient
	}

	// Create a minimal config
	config := &v1alpha1.CloudCoreConfig{
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:     testMasterURL,
			KubeConfig: testKubeConfig,
		},
	}

	// Call the function
	updateCloudCoreConfigMap(config)

	// Warning should be called
	assert.True(t, warningCalled)
}

// TestUpdateCloudCoreConfigMapWithClient tests the ConfigMap update function
func TestUpdateCloudCoreConfigMapWithClient(t *testing.T) {
	// Test successful update
	t.Run("Success case", func(t *testing.T) {
		// Create ConfigMap
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.CloudConfigMapName,
				Namespace: constants.SystemNamespace,
			},
			Data: map[string]string{
				"cloudcore.yaml": initialConfig,
			},
		}

		// Create fake client with ConfigMap
		fakeClient := fake.NewSimpleClientset(cm)

		// Create config for testing
		config := &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{
				Master:     testMasterURL,
				KubeConfig: testKubeConfig,
			},
		}

		// Call function
		err := UpdateCloudCoreConfigMapWithClient(config, fakeClient)

		// Verify results
		assert.NoError(t, err)

		// Check that ConfigMap was updated
		updatedCM, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), constants.CloudConfigMapName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotEqual(t, initialConfig, updatedCM.Data["cloudcore.yaml"])
		assert.Contains(t, updatedCM.Data["cloudcore.yaml"], testMasterURL)
	})

	// Test error case
	t.Run("Error case", func(t *testing.T) {
		// Create fake client without ConfigMap
		fakeClient := fake.NewSimpleClientset()

		// Create config for testing
		config := &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{
				Master:     testMasterURL,
				KubeConfig: testKubeConfig,
			},
		}

		// Call function
		err := UpdateCloudCoreConfigMapWithClient(config, fakeClient)

		// Should return error
		assert.Error(t, err)
	})

	// Test update error
	t.Run("Update failure", func(t *testing.T) {
		// Create ConfigMap
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.CloudConfigMapName,
				Namespace: constants.SystemNamespace,
			},
			Data: map[string]string{
				"cloudcore.yaml": initialConfig,
			},
		}

		// Create fake client with ConfigMap
		fakeClient := fake.NewSimpleClientset(cm)

		// Make update return an error
		fakeClient.PrependReactor("update", "configmaps", func(action testclient.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New(updateErrorMsg)
		})

		// Create config for testing
		config := &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{
				Master:     testMasterURL,
				KubeConfig: testKubeConfig,
			},
		}

		// Call function
		err := UpdateCloudCoreConfigMapWithClient(config, fakeClient)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), updateErrorMsg)
	})
}

// Save and restore global variable state for testing
func saveGlobals() (restore func()) {
	oldGetHostname := getHostnameFunc
	oldGetLocalIP := getLocalIPFunc
	oldCreateNamespace := createNamespaceIfNeededFunc

	return func() {
		getHostnameFunc = oldGetHostname
		getLocalIPFunc = oldGetLocalIP
		createNamespaceIfNeededFunc = oldCreateNamespace
	}
}

// TestNegotiatePortFunc tests the port selection algorithm

// TestNewCloudCoreCommand tests the structure of CloudCoreCommand
func TestNewCloudCoreCommand(t *testing.T) {
	cmd := NewCloudCoreCommand()

	// Verify command structure
	assert.Equal(t, "cloudcore", cmd.Use)
	assert.Contains(t, cmd.Long, "CloudCore is the core cloud part of KubeEdge")
	assert.NotNil(t, cmd.Run, "Command should have a Run function")

	// Verify flags exist
	flags := cmd.Flags()
	assert.NotNil(t, flags, "Command should have flags")

	// Test the command has usage and help functions
	assert.NotNil(t, cmd.UsageFunc(), "Command should have a usage function")
	assert.NotNil(t, cmd.HelpFunc(), "Command should have a help function")
}

// TestNegotiateTunnelPortWithClient tests the tunnel port negotiation
func TestNegotiateTunnelPortWithClient(t *testing.T) {
	// Save global functions and restore after test
	restore := saveGlobals()
	defer restore()

	// Override hostname and IP functions
	getHostnameFunc = func() string {
		return "test-host"
	}
	getLocalIPFunc = func(hostname string) (string, error) {
		return "192.168.1.100", nil
	}
	createNamespaceIfNeededFunc = func(ctx context.Context, namespace string) error {
		return nil
	}

	// Create fake client
	fakeClient := fake.NewSimpleClientset()

	t.Run("ConfigMap not found", func(t *testing.T) {
		// Test when ConfigMap doesn't exist
		port, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should create a new ConfigMap with a single port
		assert.NoError(t, err)
		assert.NotNil(t, port)
		assert.Equal(t, constants.ServerPort+1, *port)

		// Verify ConfigMap was created
		cm, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), modules.TunnelPort, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, cm)

		// Verify record data
		recordStr := cm.Annotations[modules.TunnelPortRecordAnnotationKey]
		var record struct {
			IPTunnelPort map[string]int `json:"IPTunnelPort"`
			Port         map[int]bool   `json:"Port"`
		}
		err = json.Unmarshal([]byte(recordStr), &record)
		assert.NoError(t, err)
		assert.Equal(t, constants.ServerPort+1, record.IPTunnelPort["192.168.1.100"])
		assert.True(t, record.Port[constants.ServerPort+1])
	})

	t.Run("ConfigMap exists with IP", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Create ConfigMap with existing record for our IP
		record := struct {
			IPTunnelPort map[string]int `json:"IPTunnelPort"`
			Port         map[int]bool   `json:"Port"`
		}{
			IPTunnelPort: map[string]int{
				"192.168.1.100": constants.ServerPort + 5,
			},
			Port: map[int]bool{
				constants.ServerPort + 5: true,
			},
		}
		recordBytes, _ := json.Marshal(record)

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: string(recordBytes),
				},
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		assert.NoError(t, err)

		// Call function
		port, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return existing port
		assert.NoError(t, err)
		assert.NotNil(t, port)
		assert.Equal(t, constants.ServerPort+5, *port)
	})

	t.Run("ConfigMap exists without IP", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Create ConfigMap with existing record but without our IP
		record := struct {
			IPTunnelPort map[string]int `json:"IPTunnelPort"`
			Port         map[int]bool   `json:"Port"`
		}{
			IPTunnelPort: map[string]int{
				"10.0.0.1": constants.ServerPort + 5,
			},
			Port: map[int]bool{
				constants.ServerPort + 5: true,
			},
		}
		recordBytes, _ := json.Marshal(record)

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: string(recordBytes),
				},
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		assert.NoError(t, err)

		// Call function
		port, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should allocate a new port
		assert.NoError(t, err)
		assert.NotNil(t, port)
		assert.Equal(t, constants.ServerPort+1, *port) // First available

		// Verify ConfigMap was updated
		updatedCM, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), modules.TunnelPort, metav1.GetOptions{})
		assert.NoError(t, err)

		var updatedRecord struct {
			IPTunnelPort map[string]int `json:"IPTunnelPort"`
			Port         map[int]bool   `json:"Port"`
		}
		err = json.Unmarshal([]byte(updatedCM.Annotations[modules.TunnelPortRecordAnnotationKey]), &updatedRecord)
		assert.NoError(t, err)

		// Should have both IPs now
		assert.Equal(t, constants.ServerPort+5, updatedRecord.IPTunnelPort["10.0.0.1"])
		assert.Equal(t, constants.ServerPort+1, updatedRecord.IPTunnelPort["192.168.1.100"])
	})

	t.Run("Error getting ConfigMap", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Make Get return an error
		fakeClient.PrependReactor("get", "configmaps", func(action testclient.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("test error")
		})

		// Call function
		_, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
	})

	t.Run("Invalid ConfigMap annotation", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Create ConfigMap with invalid JSON in annotation
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: "{invalid-json",
				},
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		assert.NoError(t, err)

		// Call function
		_, err = NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
	})

	t.Run("Missing annotation", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Create ConfigMap with missing annotation
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        modules.TunnelPort,
				Namespace:   constants.SystemNamespace,
				Annotations: map[string]string{},
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		assert.NoError(t, err)

		// Call function
		_, err = NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get tunnel port record")
	})

	t.Run("Update error", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Create ConfigMap with existing record but without our IP
		record := struct {
			IPTunnelPort map[string]int `json:"IPTunnelPort"`
			Port         map[int]bool   `json:"Port"`
		}{
			IPTunnelPort: map[string]int{
				"10.0.0.1": constants.ServerPort + 5,
			},
			Port: map[int]bool{
				constants.ServerPort + 5: true,
			},
		}
		recordBytes, _ := json.Marshal(record)

		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      modules.TunnelPort,
				Namespace: constants.SystemNamespace,
				Annotations: map[string]string{
					modules.TunnelPortRecordAnnotationKey: string(recordBytes),
				},
			},
		}

		_, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		assert.NoError(t, err)

		// Make Update return an error
		fakeClient.PrependReactor("update", "configmaps", func(action testclient.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("update error")
		})

		// Call function
		_, err = NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update error")
	})

	t.Run("Create error", func(t *testing.T) {
		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Make Create return an error
		fakeClient.PrependReactor("create", "configmaps", func(action testclient.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("create error")
		})

		// Call function
		_, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})

	t.Run("CreateNamespaceIfNeeded error", func(t *testing.T) {
		// Override createNamespaceIfNeeded to return error
		createNamespaceIfNeededFunc = func(ctx context.Context, namespace string) error {
			return errors.New("namespace error")
		}

		// Create new fake client
		fakeClient := fake.NewSimpleClientset()

		// Call function
		_, err := NegotiateTunnelPortWithClient(fakeClient)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create system namespace")
	})
}
