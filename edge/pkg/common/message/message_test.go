package message_test

import (
	"fmt"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/stretchr/testify/assert"
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
		hasError  bool
	}{
		{
			name:      "Valid Group Resource Operation",
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
			},
			hasError: false,
		},
		{
			name:      "",
			parentID:  "",
			group:     "",
			source:    "",
			operation: "",
			resource:  "",
			content:   "",
			result:    model.Message{},
			hasError:  true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {

			result := message.BuildMsg(test.group, test.parentID, test.source, test.resource, test.operation, test.content)

			fmt.Println("Result", result)
			assert.NotNil(result)

			content, ok := result.Content.(string)
			assert.True(ok, "Content should be of type string")

			// Directly compare the content if it's a string
			assert.Equal(test.content, content)

			assert.Equal(test.result.Router.Group, result.GetGroup())
			assert.Equal(test.result.Router.Resource, result.GetResource())
			assert.Equal(test.result.Router.Operation, result.GetOperation())
			assert.Equal(test.result.Router.Source, result.GetSource())

		})
	}
}
