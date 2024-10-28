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
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewLeases(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	leases := newLeases(namespace, s)

	assert.NotNil(leases)
	assert.Equal(namespace, leases.namespace)
	assert.IsType(&send{}, leases.send)
}

func TestLeases_Create(t *testing.T) {
	assert := assert.New(t)

	leaseName := "test-lease"
	inputLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: leaseName,
		},
	}

	testCases := []struct {
		name          string
		respFunc      func(*model.Message) (*model.Message, error)
		expectedLease *coordinationv1.Lease
		expectErr     bool
	}{
		{
			name: "Successful Create",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				leaseResp := LeaseResp{
					Object: inputLease,
					Err:    apierrors.StatusError{},
				}
				content, _ := json.Marshal(leaseResp)
				resp.Content = content
				return resp, nil
			},
			expectedLease: inputLease,
			expectErr:     false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				leaseResp := LeaseResp{
					Object: nil,
					Err: apierrors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Test error",
							Reason:  metav1.StatusReasonInternalError,
							Code:    500,
						},
					},
				}
				content, _ := json.Marshal(leaseResp)
				resp.Content = content
				return resp, nil
			},
			expectedLease: nil,
			expectErr:     true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeLease, leaseName), message.GetResource())
				assert.Equal(model.InsertOperation, message.GetOperation())

				content, err := message.GetContentData()
				assert.NoError(err)
				var lease coordinationv1.Lease
				err = json.Unmarshal(content, &lease)
				assert.NoError(err)
				assert.Equal(inputLease, &lease)

				return test.respFunc(message)
			}

			leaseClient := newLeases(namespace, mockSend)

			createdLease, err := leaseClient.Create(inputLease)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(createdLease)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedLease, createdLease)
			}
		})
	}
}

func TestLeases_Get(t *testing.T) {
	assert := assert.New(t)

	leaseName := "test-lease"
	expectedLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: leaseName,
		},
	}

	testCases := []struct {
		name          string
		respFunc      func(*model.Message) (*model.Message, error)
		expectedLease *coordinationv1.Lease
		expectErr     bool
	}{
		{
			name: "Successful Get",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				leaseResp := LeaseResp{
					Object: expectedLease,
					Err:    apierrors.StatusError{},
				}
				content, _ := json.Marshal(leaseResp)
				resp.Content = content
				return resp, nil
			},
			expectedLease: expectedLease,
			expectErr:     false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				leaseResp := LeaseResp{
					Object: nil,
					Err: apierrors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Test error",
							Reason:  metav1.StatusReasonInternalError,
							Code:    500,
						},
					},
				}
				content, _ := json.Marshal(leaseResp)
				resp.Content = content
				return resp, nil
			},
			expectedLease: nil,
			expectErr:     true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeLease, leaseName), message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			leaseClient := newLeases(namespace, mockSend)

			lease, err := leaseClient.Get(leaseName)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(lease)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedLease, lease)
			}
		})
	}
}

func TestHandleLeaseResp(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Successful response
	expectedLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-lease",
		},
	}

	successResp := LeaseResp{
		Object: expectedLease,
		Err:    apierrors.StatusError{},
	}

	successContent, _ := json.Marshal(successResp)

	lease, err := handleLeaseResp(successContent)
	assert.NoError(err)
	assert.Equal(expectedLease, lease)

	// Test case 2: Error response
	errorResp := LeaseResp{
		Object: nil,
		Err: apierrors.StatusError{
			ErrStatus: metav1.Status{
				Message: "Test error",
				Code:    400,
			},
		},
	}

	errorContent, _ := json.Marshal(errorResp)

	lease, err = handleLeaseResp(errorContent)
	assert.Error(err)
	assert.Nil(lease)
	assert.Equal("Test error", err.Error())

	// Test case 3: Invalid JSON
	invalidContent := []byte("invalid json")

	lease, err = handleLeaseResp(invalidContent)
	assert.Error(err)
	assert.Nil(lease)
	assert.Contains(err.Error(), "unmarshal message to lease failed")
}
