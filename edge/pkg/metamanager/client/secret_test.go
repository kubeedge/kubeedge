package client

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// MockMetaService is a mock implementation of MetaServiceInterface
type MockMetaService struct {
	QueryMetaFunc func(key, value string) (*[]string, error)
}

func (m *MockMetaService) QueryMeta(key, value string) (*[]string, error) {
	if m.QueryMetaFunc != nil {
		return m.QueryMetaFunc(key, value)
	}
	return &[]string{}, nil
}

func TestNewSecrets(t *testing.T) {
	mockSend := &mockSendInterface{}

	s := newSecrets(testNamespace, mockSend)

	assert.NotNil(t, s)
	assert.Equal(t, testNamespace, s.namespace)
	assert.Equal(t, mockSend, s.send)
}

func TestSecretsCreate(t *testing.T) {
	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)

	result, err := s.Create(&api.Secret{})

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestSecretsUpdate(t *testing.T) {
	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)

	err := s.Update(&api.Secret{})

	assert.NoError(t, err)
}

func TestSecretsDelete(t *testing.T) {
	mockSend := &mockSendInterface{}
	mockMeta := &MockMetaService{}
	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)

	err := s.Delete("test-secret")

	assert.NoError(t, err)
}

func TestSecretsGet_FromMetaDB_Success(t *testing.T) {
	mockSend := &mockSendInterface{}

	testSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Type: api.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	secretData, _ := json.Marshal(testSecret)
	secretList := []string{string(secretData)}

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return &secretList, nil
		},
	}

	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)
	secret, err := s.Get("test-secret")

	assert.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "test-secret", secret.Name)
	assert.Equal(t, testNamespace, secret.Namespace)
}

func TestSecretsGet_FromMetaDB_QueryError(t *testing.T) {
	mockSend := &mockSendInterface{}

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return nil, fmt.Errorf("query failed")
		},
	}

	mockSend.sendSyncFunc = func(msg *model.Message) (*model.Message, error) {
		return nil, fmt.Errorf("remote get failed")
	}

	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)
	secret, err := s.Get("test-secret")

	assert.Error(t, err)
	assert.Nil(t, secret)
	assert.Contains(t, err.Error(), "remote get failed")
}

func TestSecretsGet_FromMetaDB_Empty_FallbackToRemote(t *testing.T) {
	testSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Type: api.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
		},
	}

	secretData, _ := json.Marshal(testSecret)

	mockMeta := &MockMetaService{
		QueryMetaFunc: func(key, value string) (*[]string, error) {
			return &[]string{}, nil
		},
	}

	mockSend := &mockSendInterface{
		sendSyncFunc: func(msg *model.Message) (*model.Message, error) {
			mockMsg := &model.Message{
				Content: secretData,
			}
			return mockMsg, nil
		},
	}

	s := NewSecretsWithMetaService(testNamespace, mockSend, mockMeta)
	secret, err := s.Get("test-secret")

	assert.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "test-secret", secret.Name)
}

func TestHandleSecretFromMetaDB_Success(t *testing.T) {
	testSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Type: api.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	secretData, err := json.Marshal(testSecret)
	assert.NoError(t, err)

	lists := []string{string(secretData)}
	result, err := handleSecretFromMetaDB(&lists)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testSecret.Name, result.Name)
	assert.Equal(t, testSecret.Namespace, result.Namespace)
	assert.Equal(t, testSecret.Data, result.Data)
}

func TestHandleSecretFromMetaDB_EmptyList(t *testing.T) {
	lists := []string{}
	result, err := handleSecretFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "secret length from meta db is 0")
}

func TestHandleSecretFromMetaDB_MultipleItems(t *testing.T) {
	lists := []string{"secret1", "secret2"}
	result, err := handleSecretFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "secret length from meta db is 2")
}

func TestHandleSecretFromMetaDB_UnmarshalError(t *testing.T) {
	lists := []string{"invalid-json"}
	result, err := handleSecretFromMetaDB(&lists)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to secret from db failed")
}

func TestHandleSecretFromMetaManager_Success(t *testing.T) {
	testSecret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: testNamespace,
		},
		Type: api.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret"),
		},
	}

	content, err := json.Marshal(testSecret)
	assert.NoError(t, err)

	result, err := handleSecretFromMetaManager(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testSecret.Name, result.Name)
	assert.Equal(t, testSecret.Type, result.Type)
	assert.Equal(t, len(testSecret.Data), len(result.Data))
}

func TestHandleSecretFromMetaManager_UnmarshalError(t *testing.T) {
	invalidContent := []byte("invalid-json")

	result, err := handleSecretFromMetaManager(invalidContent)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unmarshal message to secret failed")
}

func TestHandleSecretFromMetaManager_EmptyContent(t *testing.T) {
	content := []byte("{}")

	result, err := handleSecretFromMetaManager(content)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Name)
}
