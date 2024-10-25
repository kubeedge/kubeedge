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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	edgeapi "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewPodStatus(t *testing.T) {
	assert := assert.New(t)

	s := newSend()

	ps := newPodStatus(namespace, s)

	assert.NotNil(ps)
	assert.Equal(namespace, ps.namespace)
	assert.IsType(&send{}, ps.send)
}

func TestPodStatus_Update(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		rsName         string
		podStatusReq   edgeapi.PodStatusRequest
		sendSyncResult *model.Message
		sendSyncError  error
		expectedError  error
	}{
		{
			name:   "Successful Update",
			rsName: "test-rs",
			podStatusReq: edgeapi.PodStatusRequest{
				UID: "test-uid",
			},
			sendSyncResult: &model.Message{
				Content: constants.MessageSuccessfulContent,
			},
			sendSyncError: nil,
			expectedError: nil,
		},
		{
			name:   "SendSync Error",
			rsName: "test-rs",
			podStatusReq: edgeapi.PodStatusRequest{
				UID: "test-uid",
			},
			sendSyncResult: nil,
			sendSyncError:  errors.New("send sync error"),
			expectedError:  errors.New("update podstatus failed, err: send sync error"),
		},
		{
			name:   "Unsuccessful Update",
			rsName: "test-rs",
			podStatusReq: edgeapi.PodStatusRequest{
				UID: "test-uid",
			},
			sendSyncResult: &model.Message{
				Content: errors.New("update failed"),
			},
			sendSyncError: nil,
			expectedError: errors.New("update failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSend := &mockSendInterface{
				sendSyncFunc: func(message *model.Message) (*model.Message, error) {
					assert.Equal(modules.MetaGroup, message.GetGroup())
					assert.Equal(modules.EdgedModuleName, message.GetSource())
					assert.Equal(namespace+"/"+model.ResourceTypePodStatus+"/"+tc.rsName, message.GetResource())
					assert.Equal(model.UpdateOperation, message.GetOperation())

					return tc.sendSyncResult, tc.sendSyncError
				},
			}

			ps := newPodStatus(namespace, mockSend)
			err := ps.Update(tc.rsName, tc.podStatusReq)

			if tc.expectedError != nil {
				assert.EqualError(err, tc.expectedError.Error())
			} else {
				assert.NoError(err)
			}
		})
	}
}
