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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/certificates/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewCertificateSigningRequests(t *testing.T) {
	assert := assert.New(t)

	sendInterface := newSend()

	csr := newCertificateSigningRequests(sendInterface)
	assert.NotNil(csr)

	assert.NotNil(csr.send)
	assert.Equal(sendInterface, csr.send)
}

// mockSendInterface is a mock implementation of SendInterface used by multiple test files in this package
type mockSendInterface struct {
	sendSyncFunc func(*model.Message) (*model.Message, error)
	sendFunc     func(*model.Message)
}

func (s *mockSendInterface) SendSync(message *model.Message) (*model.Message, error) {
	return s.sendSyncFunc(message)
}

func (s *mockSendInterface) Send(message *model.Message) {
	if s.sendFunc != nil {
		s.sendFunc(message)
	}
}

func TestCertificateSigningRequests_Create(t *testing.T) {
	assert := assert.New(t)

	csrName := "test-csr"
	inputCSR := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: csrName,
		},
		Spec: v1.CertificateSigningRequestSpec{
			Request:    []byte("test-csr-data"),
			SignerName: "kubernetes.io/kube-apiserver-client",
			Usages:     []v1.KeyUsage{v1.UsageClientAuth},
		},
	}

	testCases := []struct {
		name        string
		respFunc    func(*model.Message) (*model.Message, error)
		expectedCSR *v1.CertificateSigningRequest
		expectErr   bool
	}{
		{
			name: "Successful Create",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				csrResp := CertificateSigningRequestResp{
					Object: inputCSR,
					Err:    apierrors.StatusError{},
				}
				content, _ := json.Marshal(csrResp)
				resp.Content = content
				return resp, nil
			},
			expectedCSR: inputCSR,
			expectErr:   false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				csrResp := CertificateSigningRequestResp{
					Object: nil,
					Err: apierrors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Test error",
							Reason:  metav1.StatusReasonInternalError,
							Code:    500,
						},
					},
				}
				content, _ := json.Marshal(csrResp)
				resp.Content = content
				return resp, nil
			},
			expectedCSR: nil,
			expectErr:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal("default/certificatesigningrequest/test-csr", message.GetResource())

				content, err := message.GetContentData()
				assert.NoError(err)
				var csr v1.CertificateSigningRequest
				err = json.Unmarshal(content, &csr)
				assert.NoError(err)
				assert.Equal(inputCSR, &csr)

				return test.respFunc(message)
			}

			csrClient := newCertificateSigningRequests(mockSend)

			createdCSR, err := csrClient.Create(inputCSR)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(createdCSR)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedCSR, createdCSR)
			}
		})
	}
}

func TestCertificateSigningRequests_Get(t *testing.T) {
	assert := assert.New(t)

	csrName := "test-csr"
	expectedCSR := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: csrName,
		},
		Spec: v1.CertificateSigningRequestSpec{
			Request: []byte("test-csr-data"),
		},
	}

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		stdResult *v1.CertificateSigningRequest
		expectErr bool
	}{
		{
			name: "Get using MetaManager",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = expectedCSR
				return resp, nil
			},
			stdResult: expectedCSR,
			expectErr: false,
		},
		{
			name: "Get using MetaDB",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Source = modules.MetaManagerModuleName
				resp.Router.Operation = model.ResponseOperation
				csrJSON, _ := json.Marshal(expectedCSR)
				resp.Content = []string{string(csrJSON)}
				return resp, nil
			},
			stdResult: expectedCSR,
			expectErr: false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, fmt.Errorf("test error")
			},
			stdResult: nil,
			expectErr: true,
		},
		{
			name: "Invalid MetaDB response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Source = modules.MetaManagerModuleName
				resp.Router.Operation = model.ResponseOperation
				resp.Content = []string{"{invalid json}"}
				return resp, nil
			},
			stdResult: nil,
			expectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal("default/certificatesigningrequest/test-csr", message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			csrClient := newCertificateSigningRequests(mockSend)

			csr, err := csrClient.Get(csrName)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(csr)
			} else {
				assert.NoError(err)
				assert.Equal(test.stdResult, csr)
			}
		})
	}
}

func TestHandleCertificateSigningRequestFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid CSR JSON in array
	validCSR := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr",
		},
		Spec: v1.CertificateSigningRequestSpec{
			Request: []byte("test-csr-data"),
			Usages:  []v1.KeyUsage{v1.UsageDigitalSignature, v1.UsageKeyEncipherment},
		},
	}
	csrJSON, _ := json.Marshal(validCSR)
	validContent, _ := json.Marshal([]string{string(csrJSON)})

	csr, err := handleCertificateSigningRequestFromMetaDB(validContent)
	assert.NoError(err)
	assert.NotNil(csr)
	assert.Equal(validCSR.Name, csr.Name)
	assert.Equal(validCSR.Spec.Request, csr.Spec.Request)
	assert.Equal(validCSR.Spec.Usages, csr.Spec.Usages)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	csr, err = handleCertificateSigningRequestFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "unmarshal message to csr list from db failed")

	// Test case 3: Empty array
	emptyContent, _ := json.Marshal([]string{})

	csr, err = handleCertificateSigningRequestFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "csr length from meta db is 0")

	// Test case 4: Array with multiple elements
	multipleContent, _ := json.Marshal([]string{"{}", "{}"})

	csr, err = handleCertificateSigningRequestFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "csr length from meta db is 2")

	// Test case 5: Invalid CSR JSON in the array
	invalidCSRContent, _ := json.Marshal([]string{"{invalid json}"})

	csr, err = handleCertificateSigningRequestFromMetaDB(invalidCSRContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "unmarshal message to csr from db failed")
}

func TestHandleCertificateSigningRequestFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid CSR JSON
	validCSR := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr",
		},
		Spec: v1.CertificateSigningRequestSpec{
			Request: []byte("test-csr-data"),
			Usages:  []v1.KeyUsage{v1.UsageDigitalSignature, v1.UsageKeyEncipherment},
		},
	}

	validContent, _ := json.Marshal(validCSR)

	csr, err := handleCertificateSigningRequestFromMetaManager(validContent)
	assert.NoError(err)
	assert.NotNil(csr)
	assert.Equal(validCSR.Name, csr.Name)
	assert.Equal(validCSR.Spec.Request, csr.Spec.Request)
	assert.Equal(validCSR.Spec.Usages, csr.Spec.Usages)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	csr, err = handleCertificateSigningRequestFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "unmarshal message to csr failed")

	// Test case 3: Empty JSON object
	emptyContent := []byte("{}")

	csr, err = handleCertificateSigningRequestFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.NotNil(csr)
	assert.Empty(csr.Name)
	assert.Empty(csr.Spec.Request)

	// Test case 4: Partial CSR JSON
	partialCSR := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "partial-csr",
		},
	}

	partialContent, _ := json.Marshal(partialCSR)

	csr, err = handleCertificateSigningRequestFromMetaManager(partialContent)
	assert.NoError(err)
	assert.NotNil(csr, "Should return a non-nil csr even for partial CSR JSON")
	assert.Equal("partial-csr", csr.Name)
	assert.Empty(csr.Spec.Request)
}

func TestHandleCertificatesSigningRequestResp(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Successful response
	testCSR := &v1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-csr",
		},
		Spec: v1.CertificateSigningRequestSpec{
			Request: []byte("test-csr-data"),
		},
	}

	successResp := CertificateSigningRequestResp{
		Object: testCSR,
		Err:    apierrors.StatusError{},
	}

	successContent, _ := json.Marshal(successResp)

	csr, err := handleCertificatesSigningRequestResp(successContent)
	assert.NoError(err)
	assert.Equal(testCSR, csr)

	// Test case 2: Error response
	errorResp := CertificateSigningRequestResp{
		Object: nil,
		Err: apierrors.StatusError{
			ErrStatus: metav1.Status{
				Message: "Test error",
				Code:    400,
			},
		},
	}

	errorContent, _ := json.Marshal(errorResp)

	csr, err = handleCertificatesSigningRequestResp(errorContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Equal("Test error", err.Error())

	// Test case 3: Invalid JSON
	invalidContent := []byte("invalid json")

	csr, err = handleCertificatesSigningRequestResp(invalidContent)
	assert.Error(err)
	assert.Nil(csr)
	assert.Contains(err.Error(), "unmarshal message to CertificateSigningRequestResp failed")
}
