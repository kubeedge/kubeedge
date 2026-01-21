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
	stderrors "errors"
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

const (
	testNodeName = "test-node"
	testPodCIDR  = "10.0.0.0/24"
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

	inputNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
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
				content, err := json.Marshal(nodeResp)
				assert.NoError(err)
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
				content, err := json.Marshal(nodeResp)
				assert.NoError(err)
				resp.Content = content
				return resp, nil
			},
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "SendSync Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, stderrors.New("sendSync error")
			},
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "GetContentData Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNode, testNodeName), message.GetResource())
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
				if test.name != "Error response" {
					assert.Nil(createdNode)
				}
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedNode, createdNode)
			}
		})
	}
}

func TestNode_Update(t *testing.T) {
	assert := assert.New(t)

	inputNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: api.NodeSpec{
			PodCIDR: testPodCIDR,
		},
	}

	testCases := []struct {
		name      string
		respFunc  func(*model.Message) (*model.Message, error)
		expectErr bool
	}{
		{
			name: "Successful Update",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				return resp, nil
			},
			expectErr: false,
		},
		{
			name: "SendSync Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, stderrors.New("sendSync error")
			},
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNode, testNodeName), message.GetResource())
				assert.Equal(model.UpdateOperation, message.GetOperation())

				content, err := message.GetContentData()
				assert.NoError(err)
				var node api.Node
				err = json.Unmarshal(content, &node)
				assert.NoError(err)
				assert.Equal(inputNode, &node)

				return test.respFunc(message)
			}

			nodeClient := newNodes(namespace, mockSend)

			err := nodeClient.Update(inputNode)

			if test.expectErr {
				assert.Error(err)
				assert.Contains(err.Error(), "update node failed")
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestNode_Delete(t *testing.T) {
	assert := assert.New(t)

	nodeClient := newNodes(namespace, nil)
	err := nodeClient.Delete(testNodeName)
	assert.NoError(err, "Delete method should always return nil")
}

func TestNode_Patch(t *testing.T) {
	assert := assert.New(t)

	patchData := []byte(`{"metadata":{"labels":{"test":"label"}}}`)

	expectedNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
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
				content, err := json.Marshal(nodeResp)
				assert.NoError(err)
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
				content, err := json.Marshal(nodeResp)
				assert.NoError(err)
				resp.Content = content
				return resp, nil
			},
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "SendSync Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, stderrors.New("sendSync error")
			},
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "GetContentData Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNodePatch, testNodeName), message.GetResource())
				assert.Equal(model.PatchOperation, message.GetOperation())

				content, err := message.GetContentData()
				assert.NoError(err)
				assert.Equal(string(patchData), string(content))

				return test.respFunc(message)
			}

			nodeClient := newNodes(namespace, mockSend)

			patchedNode, err := nodeClient.Patch(testNodeName, patchData)

			if test.expectErr {
				assert.Error(err)
				if test.name != "Error response" {
					assert.Nil(patchedNode)
				}
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedNode, patchedNode)
			}
		})
	}
}

func TestNode_Get(t *testing.T) {
	assert := assert.New(t)

	expectedNode := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: api.NodeSpec{
			PodCIDR: testPodCIDR,
		},
	}
	nodeJSON, _ := json.Marshal(expectedNode)

	testCases := []struct {
		name         string
		respFunc     func(*model.Message) (*model.Message, error)
		metaDBNode   bool
		expectedNode *api.Node
		expectErr    bool
	}{
		{
			name: "Get Node from MetaManager",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Source = "other-module"
				resp.Content = nodeJSON
				return resp, nil
			},
			metaDBNode:   false,
			expectedNode: expectedNode,
			expectErr:    false,
		},
		{
			name: "Get Node from MetaDB",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				nodeList := []string{string(nodeJSON)}
				listJSON, _ := json.Marshal(nodeList)
				resp.Content = listJSON
				return resp, nil
			},
			metaDBNode:   true,
			expectedNode: expectedNode,
			expectErr:    false,
		},
		{
			name: "SendSync Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				return nil, stderrors.New("sendSync error")
			},
			metaDBNode:   false,
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "GetContentData Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = 123
				return resp, nil
			},
			metaDBNode:   false,
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "MetaDB Node Unmarshal Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Router.Operation = model.ResponseOperation
				resp.Router.Source = modules.MetaManagerModuleName
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			metaDBNode:   true,
			expectedNode: nil,
			expectErr:    true,
		},
		{
			name: "MetaManager Node Unmarshal Error",
			respFunc: func(message *model.Message) (*model.Message, error) {
				resp := model.NewMessage(message.GetID())
				resp.Content = []byte(`{"invalid": json}`)
				return resp, nil
			},
			metaDBNode:   false,
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
				assert.Equal(fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeNode, testNodeName), message.GetResource())
				assert.Equal(model.QueryOperation, message.GetOperation())

				return test.respFunc(message)
			}

			nodeClient := newNodes(namespace, mockSend)

			node, err := nodeClient.Get(testNodeName)

			if test.expectErr {
				assert.Error(err)
				assert.Nil(node)
			} else {
				assert.NoError(err)
				assert.Equal(test.expectedNode, node)
			}
		})
	}
}

func TestHandleNodeFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: api.NodeSpec{
			PodCIDR: testPodCIDR,
		},
	}
	nodeJSON, err := json.Marshal(node)
	assert.NoError(err)
	validList := []string{string(nodeJSON)}
	validContent, err := json.Marshal(validList)
	assert.NoError(err)

	result, err := handleNodeFromMetaDB(validContent)
	assert.NoError(err)
	assert.Equal(node, result)

	emptyList := []string{}
	emptyContent, err := json.Marshal(emptyList)
	assert.NoError(err)

	result, err = handleNodeFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "node length from meta db is 0")

	invalidList := []string{`{"invalid": json}`}
	invalidContent, err := json.Marshal(invalidList)
	assert.NoError(err)

	result, err = handleNodeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node from db failed")

	multipleList := []string{string(nodeJSON), string(nodeJSON)}
	multipleContent, err := json.Marshal(multipleList)
	assert.NoError(err)

	result, err = handleNodeFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "node length from meta db is 2")

	invalidContent = []byte(`{"not": "a list"}`)

	result, err = handleNodeFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node list from db failed")
}

func TestHandleNodeFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: api.NodeSpec{
			PodCIDR: testPodCIDR,
		},
	}
	content, err := json.Marshal(node)
	assert.NoError(err)

	result, err := handleNodeFromMetaManager(content)
	assert.NoError(err)
	assert.Equal(node, result)

	emptyContent := []byte(`{}`)

	result, err = handleNodeFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.Equal(&api.Node{}, result)

	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleNodeFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node failed")
}

func TestHandleNodeResp(t *testing.T) {
	assert := assert.New(t)

	node := &api.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
	}
	nodeResp := NodeResp{
		Object: node,
	}
	content, err := json.Marshal(nodeResp)
	assert.NoError(err)

	result, err := handleNodeResp(content)
	assert.NoError(err)
	assert.Equal(node, result)

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
	content, err = json.Marshal(nodeResp)
	assert.NoError(err)

	result, err = handleNodeResp(content)
	assert.Error(err)
	assert.Nil(result)
	assert.Equal(&statusErr, err)

	invalidContent := []byte(`{"invalid": json}`)

	result, err = handleNodeResp(invalidContent)
	assert.Error(err)
	assert.Nil(result)
	assert.Contains(err.Error(), "unmarshal message to node failed")
}
