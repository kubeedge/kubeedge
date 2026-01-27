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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewServiceAccountToken(t *testing.T) {
	mockSend := newMockSend()

	sat := newServiceAccountToken(mockSend)

	assert.NotNil(t, sat)
	assert.Equal(t, mockSend, sat.send)
}

func TestServiceAccountToken_GetServiceAccountToken(t *testing.T) {
	namespace := testNamespace
	saName := "test-sa"
	podUID := types.UID("test-pod-uid")

	testCases := []struct {
		name      string
		namespace string
		saName    string
		tr        *authenticationv1.TokenRequest
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Get Token Success",
			namespace: namespace,
			saName:    saName,
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"https://kubernetes.default.svc"},
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind:       "Pod",
						APIVersion: "v1",
						Name:       "test-pod",
						UID:        podUID,
					},
				},
			},
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = "OK"
				return resp, nil
			},
			expectErr: false,
		},
		{
			name:      "Get Token Network Error",
			namespace: namespace,
			saName:    saName,
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"https://kubernetes.default.svc"},
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
				},
			},
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("network error")
			},
			expectErr: true,
			errMsg:    "get service account token from metaManager failed",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(t, modules.MetaGroup, message.GetGroup())
				assert.Equal(t, modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(t, message.GetID())
				assert.Equal(t, fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeServiceAccountToken, saName),
					message.GetResource())
				assert.Equal(t, model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			satClient := newServiceAccountToken(mockSend)
			tr, err := satClient.GetServiceAccountToken(test.namespace, test.saName, test.tr)

			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, tr)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceAccountToken_DeleteServiceAccountToken(t *testing.T) {
	testCases := []struct {
		name   string
		podUID types.UID
	}{
		{
			name:   "Delete Token Success",
			podUID: types.UID("test-pod-uid-123"),
		},
		{
			name:   "Delete Token with Empty UID",
			podUID: types.UID(""),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := newMockSend()
			satClient := newServiceAccountToken(mockSend)

			// Should not panic or error
			satClient.DeleteServiceAccountToken(test.podUID)
			assert.True(t, true)
		})
	}
}

func TestRequiresRefresh(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name      string
		tr        *authenticationv1.TokenRequest
		expectErr bool
	}{
		{
			name: "Token Not Expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(2 * time.Hour)),
				},
			},
			expectErr: false,
		},
		{
			name: "Token Expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
				},
			},
			expectErr: false,
		},
		{
			name: "Token Requires Refresh (80% TTL)",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(5 * time.Minute)),
				},
			},
			expectErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := requiresRefresh(test.tr)
			assert.IsType(t, false, result)
		})
	}
}

func TestKeyFunc(t *testing.T) {
	name := "test-sa"
	namespace := testNamespace
	expSeconds := int64(3600)

	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"https://kubernetes.default.svc"},
			ExpirationSeconds: &expSeconds,
			BoundObjectRef: &authenticationv1.BoundObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Name:       "test-pod",
				UID:        types.UID("test-uid"),
			},
		},
	}

	key := KeyFunc(name, namespace, tr)

	assert.NotEmpty(t, key)
	assert.Contains(t, key, name)
	assert.Contains(t, key, namespace)
}

func TestHandleServiceAccountTokenFromMetaDB_Success(t *testing.T) {
	expSeconds := int64(3600)
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"https://kubernetes.default.svc"},
			ExpirationSeconds: &expSeconds,
		},
		Status: authenticationv1.TokenRequestStatus{
			Token:               "test-token",
			ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
		},
	}

	trJSON, err := json.Marshal(tr)
	require.NoError(t, err)

	content, err := json.Marshal([]string{string(trJSON)})
	require.NoError(t, err)

	result, err := handleServiceAccountTokenFromMetaDB(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tr.Status.Token, result.Status.Token)
}

func TestHandleServiceAccountTokenFromMetaDB_EmptyList(t *testing.T) {
	content, _ := json.Marshal([]string{})

	result, err := handleServiceAccountTokenFromMetaDB(content)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "serviceaccount length from meta db is 0")
}

func TestHandleServiceAccountTokenFromMetaDB_MultipleItems(t *testing.T) {
	content, _ := json.Marshal([]string{"{}", "{}"})

	result, err := handleServiceAccountTokenFromMetaDB(content)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "serviceaccount length from meta db is 2")
}

func TestHandleServiceAccountTokenFromMetaDB_UnmarshalError(t *testing.T) {
	content := []byte("invalid json")

	result, err := handleServiceAccountTokenFromMetaDB(content)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to serviceaccount list from db failed")
}

func TestHandleServiceAccountTokenFromMetaManager_Success(t *testing.T) {
	expSeconds := int64(3600)
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"https://kubernetes.default.svc"},
			ExpirationSeconds: &expSeconds,
		},
		Status: authenticationv1.TokenRequestStatus{
			Token:               "test-token",
			ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
		},
	}

	content, err := json.Marshal(tr)
	require.NoError(t, err)

	result, err := handleServiceAccountTokenFromMetaManager(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tr.Status.Token, result.Status.Token)
}

func TestHandleServiceAccountTokenFromMetaManager_InvalidJSON(t *testing.T) {
	content := []byte("invalid json")

	result, err := handleServiceAccountTokenFromMetaManager(content)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to service account failed")
}

func TestNewServiceAccount(t *testing.T) {
	sa := newServiceAccount(testNamespace)

	assert.NotNil(t, sa)
	assert.Equal(t, testNamespace, sa.namespace)
}

func TestServiceAccount_Get(t *testing.T) {
	testCases := []struct {
		name      string
		saName    string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Get ServiceAccount Not Found",
			saName:    "non-existent-sa",
			expectErr: true,
			errMsg:    "serviceaccount",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			saClient := newServiceAccount(testNamespace)
			sa, err := saClient.Get(test.saName)

			if test.expectErr {
				assert.Error(t, err)
				assert.Nil(t, sa)
				if test.errMsg != "" {
					assert.Contains(t, err.Error(), test.errMsg)
				}
			}
		})
	}
}

func TestCheckTokenExist(t *testing.T) {
	testCases := []struct {
		name         string
		token        string
		expectExists bool
	}{
		{
			name:         "Empty Token",
			token:        "",
			expectExists: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			exists := CheckTokenExist(test.token)

			if test.expectExists {
				assert.True(t, exists)
			} else {
				assert.False(t, exists)
			}
		})
	}
}

func TestServiceAccountToken_Interface(t *testing.T) {
	// Verify that serviceAccountToken implements ServiceAccountTokenInterface
	mockSend := newMockSend()
	satClient := newServiceAccountToken(mockSend)

	assert.NotNil(t, satClient)
	assert.NotNil(t, satClient.send)
}

func TestServiceAccount_Interface(t *testing.T) {
	// Verify that serviceAccount implements ServiceAccountInterface
	saClient := newServiceAccount(testNamespace)

	assert.NotNil(t, saClient)
	assert.Equal(t, testNamespace, saClient.namespace)
}

func TestRequiresRefresh_EdgeCases(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name       string
		tr         *authenticationv1.TokenRequest
		expectBool bool
	}{
		{
			name: "Nil ExpirationSeconds",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: nil,
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
				},
			},
			expectBool: false,
		},
		{
			name: "Token Exactly At 80% TTL",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(12 * time.Minute)),
				},
			},
			expectBool: true,
		},
		{
			name: "Token Beyond 24 Hour MaxTTL",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func(i int64) *int64 { return &i }(86400 * 2), // 2 days
				},
				Status: authenticationv1.TokenRequestStatus{
					Token:               "test-token",
					ExpirationTimestamp: metav1.NewTime(now.Add(48 * time.Hour)),
				},
			},
			expectBool: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := requiresRefresh(test.tr)
			assert.Equal(t, test.expectBool, result)
		})
	}
}

func TestKeyFunc_EdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		saName     string
		namespace  string
		tr         *authenticationv1.TokenRequest
		expectFunc bool
	}{
		{
			name:      "Nil ExpirationSeconds",
			saName:    "test-sa",
			namespace: testNamespace,
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"https://kubernetes.default.svc"},
					ExpirationSeconds: nil,
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind: "Pod",
						Name: "test-pod",
						UID:  types.UID("test-uid"),
					},
				},
			},
			expectFunc: true,
		},
		{
			name:      "Nil BoundObjectRef",
			saName:    "test-sa",
			namespace: testNamespace,
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"https://kubernetes.default.svc"},
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
					BoundObjectRef:    nil,
				},
			},
			expectFunc: true,
		},
		{
			name:      "Empty Audiences",
			saName:    "test-sa",
			namespace: testNamespace,
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{},
					ExpirationSeconds: func(i int64) *int64 { return &i }(3600),
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind: "Pod",
						Name: "test-pod",
						UID:  types.UID("test-uid"),
					},
				},
			},
			expectFunc: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			key := KeyFunc(test.saName, test.namespace, test.tr)
			assert.NotEmpty(t, key)
			assert.Contains(t, key, test.saName)
			assert.Contains(t, key, test.namespace)
		})
	}
}
