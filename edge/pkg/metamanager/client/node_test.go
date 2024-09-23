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
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestNewNodes(t *testing.T) {
	assert := assert.New(t)

	s := newSend()
	nodesClient := newNodes(namespace, s)

	assert.NotNil(nodesClient)
	assert.Equal(namespace, nodesClient.namespace)
	assert.IsType(&send{}, nodesClient.send)
}

func TestNode_Create(t *testing.T) {
	assert := assert.New(t)

	nodeName := "test-node"
	inputNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}

	testCases := []struct {
		name         string
		respFunc     func(*model.Message) (*model.Message, error)
		expectedNode *api.Node
		expectErr    bool
	}{
		{
			name: "Successful Create",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				nodeResp := NodeResp{
					Object: inputNode,
					Err:    apierrors.StatusError{},
				}
				content, _ := json.Marshal(nodeResp)
				resp.Content = content
				return resp, nil
			},
			expectedNode: inputNode,
			expectErr:    false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				nodeResp := NodeResp{
					Object: nil,
					Err: apierrors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Test error",
							Reason:  metav1.StatusReasonInternalError,
							Code:    500,
						},
					},
				}
				content, _ := json.Marshal(nodeResp)
				resp.Content = content
				return resp, nil
			},
			expectedNode: nil,
			expectErr:    true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNode, nodeName), message.GetResource())
				assert.Equal(model.InsertOperation, message.GetOperation())

				content, err := message.GetContentData()
				assert.NoError(err)
				var node api.Node
				err = json.Unmarshal(content, &node)
				assert.NoError(err)
				assert.Equal(inputNode, &node)

				return test.respFunc(message)
			}

			nodeClient := newNodes(namespace, mockSend)

			createdNode, err := nodeClient.Create(inputNode)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(createdNode)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedNode, createdNode)
			}
		})
	}
}

func TestNode_Patch(t *testing.T) {
	assert := assert.New(t)

	nodeName := "test-node"
	patchData := []byte(`{"metadata":{"labels":{"test":"label"}}}`)

	expectedNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"test": "label",
			},
		},
	}

	testCases := []struct {
		name         string
		respFunc     func(*model.Message) (*model.Message, error)
		expectedNode *api.Node
		expectErr    bool
	}{
		{
			name: "Successful Patch",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				nodeResp := NodeResp{
					Object: expectedNode,
					Err:    apierrors.StatusError{},
				}
				content, _ := json.Marshal(nodeResp)
				resp.Content = content
				return resp, nil
			},
			expectedNode: expectedNode,
			expectErr:    false,
		},
		{
			name: "Error response",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				nodeResp := NodeResp{
					Object: nil,
					Err: apierrors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Test error msg",
							Reason:  metav1.StatusReasonInternalError,
							Code:    500,
						},
					},
				}
				content, _ := json.Marshal(nodeResp)
				resp.Content = content
				return resp, nil
			},
			expectedNode: nil,
			expectErr:    true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockSend := &mockSendInterface{}
			mockSend.sendSyncFunc = func(message *model.Message) (*model.Message, error) {
				assert.Equal(modules.MetaGroup, message.GetGroup())
				assert.Equal(modules.EdgedModuleName, message.GetSource())
				assert.NotEmpty(message.GetID())
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNodePatch, nodeName), message.GetResource())
				assert.Equal(model.PatchOperation, message.GetOperation())

				content, err := message.GetContentData()
				assert.NoError(err)
				assert.Equal(string(patchData), string(content))

				return test.respFunc(message)
			}

			nodeClient := newNodes(namespace, mockSend)

			patchedNode, err := nodeClient.Patch(nodeName, patchData)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(patchedNode)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedNode, patchedNode)
			}
		})
	}
}

func TestHandleNodeFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid node JSON in list
	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: api.NodeSpec{
			PodCIDR: "10.0.0.0/24",
		},
	}
	nodeJSON, _ := json.Marshal(node)
	validList := []string{string(nodeJSON)}
	validContent, _ := json.Marshal(validList)

	result, err := handleNodeFromMetaDB(validContent)
	assert.NoError(err)
	assert.Equal(node, result)

	// Test case 2: Empty list
	emptyList := []string{}
	emptyContent, _ := json.Marshal(emptyList)

	result, err = handleNodeFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "node length from meta db is 0")

	// Test case 3: Invalid JSON in list
	invalidList := []string{`{"invalid": json}`}
	invalidContent, _ := json.Marshal(invalidList)

	result, err = handleNodeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node from db failed")
}

func TestHandleNodeFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid node JSON
	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: api.NodeSpec{
			PodCIDR: "10.0.0.0/24",
		},
	}
	content, _ := json.Marshal(node)

	result, err := handleNodeFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(node, result)

	// Test case 2: Empty JSON
	emptyContent := []byte(`{}`)

	result, err = handleNodeFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.Node{}, result)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleNodeFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node failed")
}

func TestHandleNodeResp(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid node response
	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}
	nodeResp := NodeResp{
		Object: node,
	}
	content, _ := json.Marshal(nodeResp)

	result, err := handleNodeResp(content)
	assert.NoError(err)
	assert.Equal(node, result)

	// Test case 2: Response with error
	statusErr := errors.StatusError{
		ErrStatus: metav1.Status{
			Message: "Test error",
			Reason:  metav1.StatusReasonNotFound,
			Code:    404,
		},
	}
	nodeResp = NodeResp{
		Err: statusErr,
	}
	content, _ = json.Marshal(nodeResp)

	result, err = handleNodeResp(content)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(&statusErr, err)

	// Test case 3: Invalid JSON
	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleNodeResp(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node failed")
}
