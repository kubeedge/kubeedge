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

package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSecrets(t *testing.T) {
	assert := assert.New(t)

	namespace := "test-namespace"
	s := newSend()

	secret := newSecrets(namespace, s)

	assert.NotNil(secret)
	assert.Equal(namespace, secret.namespace)
	assert.IsType(&send{}, secret.send)
}

func TestHandleSecretFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid Secret JSON in a single-element list
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}
	secretJSON, _ := json.Marshal(secret)
	content := []string{string(secretJSON)}

	result, err := handleSecretFromMetaDB(&content)
	assert.NoError(err)
	assert.Equal(secret, result)

	// Test case 2: Empty list
	var emptyList []string

	result, err = handleSecretFromMetaDB(&emptyList)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "secret length from meta db is 0")

	// Test case 3: List with multiple elements
	multipleSecrets := []string{string(secretJSON), string(secretJSON)}

	result, err = handleSecretFromMetaDB(&multipleSecrets)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "secret length from meta db is 2")

	// Test case 4: Invalid JSON in the list
	invalidJSON := []string{"{invalid json}"}

	result, err = handleSecretFromMetaDB(&invalidJSON)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to secret from db failed")
}

func TestHandleSecretFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid Secret JSON
	secret := &api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("password123"),
		},
		Type: api.SecretTypeOpaque,
	}
	content, _ := json.Marshal(secret)

	result, err := handleSecretFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(secret, result)

	// Test case 2: Empty JSON
	emptyContent := []byte("{}")

	result, err = handleSecretFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.Secret{}, result)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleSecretFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to secret failed")

	// Test case 4: Partial Secret JSON
	partialSecret := []byte(`{"metadata": {"name": "partial-secret"}}`)

	result, err = handleSecretFromMetaManager(partialSecret)
	assert.NoError(err)
	assert.Equal("partial-secret", result.Name)
	assert.Nil(result.Data)
}
