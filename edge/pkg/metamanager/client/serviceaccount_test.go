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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

const testNamespace = "test-namespace"

type fakeSend struct {
	syncHandler func(*model.Message) (*model.Message, error)
	sendHandler func(*model.Message)
}

func (f *fakeSend) SendSync(message *model.Message) (*model.Message, error) {
	if f.syncHandler != nil {
		return f.syncHandler(message)
	}
	return &model.Message{}, nil
}

func (f *fakeSend) Send(message *model.Message) {
	if f.sendHandler != nil {
		f.sendHandler(message)
	}
}

func newFakeSend() *fakeSend {
	return &fakeSend{}
}

func TestNewServiceAccountToken(t *testing.T) {
	assert := assert.New(t)

	sendInterface := newFakeSend()

	sat := newServiceAccountToken(sendInterface)
	assert.NotNil(sat)
	assert.NotNil(sat.send)
	assert.Equal(sendInterface, sat.send)
}

func TestRequiresRefresh(t *testing.T) {
	assert := assert.New(t)

	now := time.Now()
	testcases := []struct {
		name     string
		tr       *authenticationv1.TokenRequest
		expected bool
	}{
		{
			name: "Not expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
				},
			},
			expected: false,
		},
		{
			name: "Expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
				},
			},
			expected: true,
		},
		{
			name: "Near expiration (within 20% of TTL)",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Minute * 10)),
				},
			},
			expected: true,
		},
		{
			name: "Beyond max TTL",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(24 * 3600 * 2); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour * 48)),
					Token:               "test-token",
				},
			},
			expected: false,
		},
		{
			name: "Nil ExpirationSeconds",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: nil,
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
				},
			},
			expected: false,
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			result := requiresRefresh(test.tr)
			assert.Equal(test.expected, result)
		})
	}
}

func TestKeyFunc(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		name      string
		saName    string
		namespace string
		tr        *authenticationv1.TokenRequest
		expected  string
	}{
		{
			name:      "Basic TokenRequest",
			saName:    "test-sa",
			namespace: "default",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"audience1", "audience2"},
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind: "Pod",
						Name: "test-pod",
						UID:  "12345",
					},
				},
			},
			expected: `"test-sa"/"default"/[]string{"audience1", "audience2"}/3600/v1.BoundObjectReference{Kind:"Pod", APIVersion:"", Name:"test-pod", UID:"12345"}`,
		},
		{
			name:      "TokenRequest with nil ExpirationSeconds",
			saName:    "test-sa",
			namespace: "kube-system",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"audience3"},
					ExpirationSeconds: nil,
					BoundObjectRef:    nil,
				},
			},
			expected: `"test-sa"/"kube-system"/[]string{"audience3"}/0/v1.BoundObjectReference{Kind:"", APIVersion:"", Name:"", UID:""}`,
		},
		{
			name:      "TokenRequest with empty fields",
			saName:    "test-sa",
			namespace: "default",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{},
			},
			expected: `"test-sa"/"default"/[]string(nil)/0/v1.BoundObjectReference{Kind:"", APIVersion:"", Name:"", UID:""}`,
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			result := KeyFunc(test.saName, test.namespace, test.tr)
			assert.Equal(test.expected, result)
		})
	}
}

func TestHandleServiceAccountTokenFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid TokenRequest JSON in array
	validTokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-token-request",
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{"audience1", "audience2"},
		},
		Status: authenticationv1.TokenRequestStatus{
			Token: "test-token",
		},
	}
	tokenRequestJSON, err := json.Marshal(validTokenRequest)
	if err != nil {
		t.Fatalf("Failed to marshal valid token request: %v", err)
	}
	validContent, err := json.Marshal([]string{string(tokenRequestJSON)})
	if err != nil {
		t.Fatalf("Failed to marshal valid content: %v", err)
	}

	tokenRequest, err := handleServiceAccountTokenFromMetaDB(validContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Equal(validTokenRequest.Name, tokenRequest.Name)
	assert.Equal(validTokenRequest.Spec.Audiences, tokenRequest.Spec.Audiences)
	assert.Equal(validTokenRequest.Status.Token, tokenRequest.Status.Token)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to serviceaccount list from db failed")

	// Test case 3: Empty array
	emptyContent, err := json.Marshal([]string{})
	if err != nil {
		t.Fatalf("Failed to marshal empty content: %v", err)
	}

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "serviceaccount length from meta db is 0")

	// Test case 4: Array with multiple elements
	multipleContent, err := json.Marshal([]string{"{}", "{}"})
	if err != nil {
		t.Fatalf("Failed to marshal multiple content: %v", err)
	}

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "serviceaccount length from meta db is 2")

	// Test case 5: Invalid TokenRequest JSON in the array
	invalidTokenRequestContent, err := json.Marshal([]string{"{invalid json}"})
	if err != nil {
		t.Fatalf("Failed to marshal invalid token request content: %v", err)
	}

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(invalidTokenRequestContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to serviceaccount token from db failed")
}

func TestHandleServiceAccountTokenFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid TokenRequest JSON
	validTokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-token-request",
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{"audience1", "audience2"},
		},
		Status: authenticationv1.TokenRequestStatus{
			Token: "test-token",
		},
	}

	validContent, err := json.Marshal(validTokenRequest)
	if err != nil {
		t.Fatalf("Failed to marshal valid token request: %v", err)
	}

	tokenRequest, err := handleServiceAccountTokenFromMetaManager(validContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Equal(validTokenRequest.Name, tokenRequest.Name)
	assert.Equal(validTokenRequest.Spec.Audiences, tokenRequest.Spec.Audiences)
	assert.Equal(validTokenRequest.Status.Token, tokenRequest.Status.Token)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	tokenRequest, err = handleServiceAccountTokenFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to service account failed")

	// Test case 3: Empty JSON object
	emptyContent := []byte("{}")

	tokenRequest, err = handleServiceAccountTokenFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Empty(tokenRequest.Name)
	assert.Empty(tokenRequest.Spec.Audiences)
	assert.Empty(tokenRequest.Status.Token)
}

func TestNewServiceAccount(t *testing.T) {
	namespace := testNamespace

	sa := newServiceAccount(namespace)

	assert.NotNil(t, sa)
	assert.IsType(t, &serviceAccount{}, sa)
	assert.Equal(t, namespace, sa.namespace)
}

func TestDeleteServiceAccountToken(t *testing.T) {
	assert := assert.New(t)

	podUID := types.UID("pod-123")

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	metaList := []dao.Meta{
		{
			Key: "key1",
			Value: func() string {
				tr := authenticationv1.TokenRequest{
					Spec: authenticationv1.TokenRequestSpec{
						BoundObjectRef: &authenticationv1.BoundObjectReference{
							UID: podUID,
						},
					},
				}
				data, err := json.Marshal(tr)
				if err != nil {
					t.Fatalf("Failed to marshal token request 1: %v", err)
				}
				return string(data)
			}(),
		},
		{
			Key: "key2",
			Value: func() string {
				tr := authenticationv1.TokenRequest{
					Spec: authenticationv1.TokenRequestSpec{
						BoundObjectRef: &authenticationv1.BoundObjectReference{
							UID: "other-pod",
						},
					},
				}
				data, err := json.Marshal(tr)
				if err != nil {
					t.Fatalf("Failed to marshal token request 2: %v", err)
				}
				return string(data)
			}(),
		},
	}

	patches.ApplyFunc(dao.QueryAllMeta, func(key, value string) (*[]dao.Meta, error) {
		assert.Equal("type", key)
		assert.Equal(model.ResourceTypeServiceAccountToken, value)
		return &metaList, nil
	})

	var deletedKey string
	patches.ApplyFunc(dao.DeleteMetaByKey, func(key string) error {
		deletedKey = key
		return nil
	})

	sat := &serviceAccountToken{}
	sat.DeleteServiceAccountToken(podUID)

	assert.Equal("key1", deletedKey)

	patches.Reset()
	patches.ApplyFunc(dao.QueryAllMeta, func(key, value string) (*[]dao.Meta, error) {
		return nil, errors.New("query meta error")
	})

	sat.DeleteServiceAccountToken(podUID)

	patches.Reset()
	metaList = []dao.Meta{
		{
			Key:   "key1",
			Value: "invalid json",
		},
	}
	patches.ApplyFunc(dao.QueryAllMeta, func(key, value string) (*[]dao.Meta, error) {
		return &metaList, nil
	})

	sat.DeleteServiceAccountToken(podUID)

	patches.Reset()
	metaList = []dao.Meta{
		{
			Key: "key1",
			Value: func() string {
				tr := authenticationv1.TokenRequest{
					Spec: authenticationv1.TokenRequestSpec{
						BoundObjectRef: &authenticationv1.BoundObjectReference{
							UID: podUID,
						},
					},
				}
				data, err := json.Marshal(tr)
				if err != nil {
					t.Fatalf("Failed to marshal token request 3: %v", err)
				}
				return string(data)
			}(),
		},
	}
	patches.ApplyFunc(dao.QueryAllMeta, func(key, value string) (*[]dao.Meta, error) {
		return &metaList, nil
	})
	patches.ApplyFunc(dao.DeleteMetaByKey, func(key string) error {
		return fmt.Errorf("failed to delete meta by key %s: delete operation failed", key)
	})

	sat.DeleteServiceAccountToken(podUID)
}

func TestGetTokenLocally(t *testing.T) {
	assert := assert.New(t)

	name := "sa-name"
	namespace := "default"
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
		},
	}
	key := KeyFunc(name, namespace, tr)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	validTR := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
		},
		Status: authenticationv1.TokenRequestStatus{
			ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Hour)),
		},
	}
	validTRBytes, err := json.Marshal(validTR)
	if err != nil {
		t.Fatalf("Failed to marshal valid token request: %v", err)
	}

	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		assert.Equal("key", k)
		assert.Equal(key, v)
		return &[]string{string(validTRBytes)}, nil
	})

	result, err := getTokenLocally(name, namespace, tr)
	assert.NoError(err)
	assert.NotNil(result)

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		return nil, errors.New("query meta error")
	})

	result, err = getTokenLocally(name, namespace, tr)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "query meta error")

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		return &[]string{}, nil
	})

	result, err = getTokenLocally(name, namespace, tr)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "query meta")
	assert.Contains(err.Error(), "length error")

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		return &[]string{"invalid-json"}, nil
	})

	result, err = getTokenLocally(name, namespace, tr)
	assert.Error(err)
	assert.Nil(result)

	patches.Reset()
	expiredTR := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
		},
		Status: authenticationv1.TokenRequestStatus{
			ExpirationTimestamp: metav1.NewTime(time.Now().Add(-time.Hour)),
		},
	}
	expiredTRBytes, err := json.Marshal(expiredTR)
	if err != nil {
		t.Fatalf("Failed to marshal expired token request: %v", err)
	}

	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		return &[]string{string(expiredTRBytes)}, nil
	})

	patches.ApplyFunc(dao.DeleteMetaByKey, func(k string) error {
		assert.Equal(key, k)
		return nil
	})

	result, err = getTokenLocally(name, namespace, tr)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "token expired")

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(k, v string) (*[]string, error) {
		return &[]string{string(expiredTRBytes)}, nil
	})

	patches.ApplyFunc(dao.DeleteMetaByKey, func(k string) error {
		return fmt.Errorf("failed to delete meta by key %s: deletion operation failed", k)
	})

	result, err = getTokenLocally(name, namespace, tr)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "failed to delete meta")
}

func TestGetServiceAccountToken(t *testing.T) {
	assert := assert.New(t)

	name := "sa-name"
	namespace := "default"
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
		},
	}

	fakeSender := newFakeSend()
	sat := &serviceAccountToken{send: fakeSender}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	localToken := &authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "local-token",
		},
	}

	patches.ApplyFunc(getTokenLocally, func(n, ns string, t *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		assert.Equal(name, n)
		assert.Equal(namespace, ns)
		assert.Equal(tr, t)
		return localToken, nil
	})

	result, err := sat.GetServiceAccountToken(namespace, name, tr)
	assert.NoError(err)
	assert.Equal(localToken, result)

	// Test remote token fetching
	patches.Reset()
	patches.ApplyFunc(getTokenLocally, func(n, ns string, t *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		return nil, errors.New("local token not found")
	})

	responseMsg := model.NewMessage("")
	remoteToken := &authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "remote-token",
		},
	}
	remoteTokenBytes, err := json.Marshal(remoteToken)
	if err != nil {
		t.Fatalf("Failed to marshal remote token: %v", err)
	}

	responseMsg.Content = remoteTokenBytes

	var capturedMessage *model.Message
	fakeSender.syncHandler = func(msg *model.Message) (*model.Message, error) {
		capturedMessage = msg
		return responseMsg, nil
	}

	result, err = sat.GetServiceAccountToken(namespace, name, tr)
	assert.NoError(err)
	assert.NotNil(result)
	assert.NotNil(capturedMessage)
	assert.Equal("remote-token", result.Status.Token)

	patches.Reset()
	patches.ApplyFunc(getTokenLocally, func(n, ns string, t *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
		return nil, errors.New("local token not found")
	})

	fakeSender.syncHandler = func(msg *model.Message) (*model.Message, error) {
		return nil, errors.New("sender error")
	}

	result, err = sat.GetServiceAccountToken(namespace, name, tr)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "sender error")
}

func TestServiceAccountGet(t *testing.T) {
	assert := assert.New(t)

	namespace := testNamespace
	name := "test-sa"
	sa := newServiceAccount(namespace)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		assert.Equal("type", key)
		assert.Equal(model.ResourceTypeSaAccess, value)

		mockData := `{"metadata":{"namespace":"test-namespace"},"spec":{"serviceAccount":{"metadata":{"name":"test-sa"}},"serviceAccountUID":"test-uid"}}`
		return &[]string{mockData}, nil
	})

	result, err := sa.Get(name)
	assert.NoError(err)
	assert.NotNil(result)
	assert.Equal(name, result.Name)
	assert.Equal(types.UID("test-uid"), result.UID)

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		return nil, errors.New("query metadata error")
	})

	result, err = sa.Get(name)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "query metadata error")

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		return &[]string{`{"metadata":{"namespace":"other-namespace"},"spec":{"serviceAccount":{"metadata":{"name":"other-name"}}}}`}, nil
	})

	result, err = sa.Get(name)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "not found")

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		return &[]string{"invalid-json"}, nil
	})

	result, err = sa.Get(name)
	assert.Error(err)
	assert.Nil(result)
}

func TestCheckTokenExist(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	assert.False(CheckTokenExist(""))

	tr1 := authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "test-token",
		},
	}
	tr1Bytes, err := json.Marshal(tr1)
	if err != nil {
		t.Fatalf("Failed to marshal token request 1: %v", err)
	}

	tr2 := authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "other-token",
		},
	}
	tr2Bytes, err := json.Marshal(tr2)
	if err != nil {
		t.Fatalf("Failed to marshal token request 2: %v", err)
	}

	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		assert.Equal("type", key)
		assert.Equal(model.ResourceTypeServiceAccountToken, value)
		return &[]string{string(tr1Bytes), string(tr2Bytes)}, nil
	})

	assert.True(CheckTokenExist("test-token"))
	assert.True(CheckTokenExist("other-token"))
	assert.False(CheckTokenExist("non-existent-token"))

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		return nil, errors.New("query metadata error")
	})

	assert.False(CheckTokenExist("test-token"))

	patches.Reset()
	patches.ApplyFunc(dao.QueryMeta, func(key, value string) (*[]string, error) {
		return &[]string{"invalid-json"}, nil
	})

	assert.False(CheckTokenExist("test-token"))
}
