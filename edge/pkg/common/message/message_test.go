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

package message_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
)

func TestBuildMsg(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		name      string
		parentID  string
		group     string
		resource  string
		source    string
		operation string
		content   interface{}
		result    model.Message
	}{
		{
			name:      "Non Empty group, resource, operation and content",
			parentID:  "parent1",
			group:     "resource",
			source:    "edgehub",
			operation: "publish",
			resource:  "node/connection",
			content:   "This is a content",
			result: model.Message{
				Router: model.MessageRoute{
					Group:     "resource",
					Resource:  "node/connection",
					Operation: "publish",
					Source:    "edgehub",
				},
				Content: "This is a content",
			},
		},
		{
			name:      "Empty parentID and content",
			parentID:  "",
			group:     "twin",
			source:    "edgehub",
			operation: "subscribe",
			resource:  "node/connection",
			content:   "",
			result: model.Message{
				Header: model.MessageHeader{
					ParentID: "",
				},
				Router: model.MessageRoute{
					Group:     "twin",
					Resource:  "node/connection",
					Operation: "subscribe",
					Source:    "edgehub",
				},
				Content: "",
			},
		},
		{
			name:      "Empty group , parentID , source , resource , operation and content",
			parentID:  "",
			group:     "",
			source:    "",
			operation: "",
			content:   "",
			resource:  "",
			result: model.Message{
				Header: model.MessageHeader{
					ParentID: "",
				},
				Router: model.MessageRoute{
					Group:     "",
					Resource:  "",
					Operation: "",
					Source:    "",
				},
				Content: "",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(_ *testing.T) {
			result := message.BuildMsg(test.group, test.parentID, test.source, test.resource, test.operation, test.content)
			assert.NotNil(result)
			assert.Equal(test.result.Router.Group, result.GetGroup())
			assert.Equal(test.result.Router.Resource, result.GetResource())
			assert.Equal(test.result.Router.Operation, result.GetOperation())
			assert.Equal(test.result.Router.Source, result.GetSource())
			assert.Equal(test.result.Content, result.GetContent())
		})
	}
}
