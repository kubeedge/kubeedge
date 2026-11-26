/*
Copyright 2026 The KubeEdge Authors.

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

package mux

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// mockResponseWriter implements ResponseWriter interface for testing
type mockResponseWriter struct {
	lastMsg     *model.Message
	lastContent interface{}
	lastError   string
}

func (m *mockResponseWriter) WriteResponse(msg *model.Message, content interface{}) {
	m.lastMsg = msg
	m.lastContent = content
}

func (m *mockResponseWriter) WriteError(msg *model.Message, err string) {
	m.lastMsg = msg
	m.lastError = err
}

func TestNewMessageMux(t *testing.T) {
	mux := NewMessageMux()
	assert.NotNil(t, mux)
	assert.Empty(t, mux.muxEntry)
}

func TestMessagePattern_Match(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		op       string
		msg      *model.Message
		want     bool
	}{
		{
			name:     "match exact resource and op",
			resource: "node/node1",
			op:       "update",
			msg: &model.Message{
				Router: model.MessageRoute{
					Resource:  "node/node1",
					Operation: "update",
				},
			},
			want: true,
		},
		{
			name:     "mismatch resource",
			resource: "node/node1",
			op:       "update",
			msg: &model.Message{
				Router: model.MessageRoute{
					Resource:  "node/node2",
					Operation: "update",
				},
			},
			want: false,
		},
		{
			name:     "mismatch operation",
			resource: "node/node1",
			op:       "update",
			msg: &model.Message{
				Router: model.MessageRoute{
					Resource:  "node/node1",
					Operation: "delete",
				},
			},
			want: false,
		},
		{
			name:     "wildcard operation",
			resource: "node/node1",
			op:       "*",
			msg: &model.Message{
				Router: model.MessageRoute{
					Resource:  "node/node1",
					Operation: "any",
				},
			},
			want: true,
		},
		{
			name:     "parameterized resource",
			resource: "node/{nodeID}",
			op:       "update",
			msg: &model.Message{
				Router: model.MessageRoute{
					Resource:  "node/123",
					Operation: "update",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := NewPattern(tt.resource).Op(tt.op)
			assert.Equal(t, tt.want, pattern.Match(tt.msg))
		})
	}
}

func TestMessageMux_Dispatch(t *testing.T) {
	mux := NewMessageMux()
	handled := false
	var capturedParams map[string]string

	// Register a handler
	pattern := NewPattern("node/{nodeID}").Op("update")
	mux.Entry(pattern, func(container *MessageContainer, writer ResponseWriter) {
		handled = true
		capturedParams = container.parameters
		writer.WriteResponse(container.Message, "ok")
	})

	// Create a request that matches
	msg := &model.Message{
		Router: model.MessageRoute{
			Resource:  "node/123",
			Operation: "update",
		},
	}
	req := &MessageRequest{Message: msg}
	writer := &mockResponseWriter{}

	// Dispatch
	mux.ServeConn(req, writer)

	assert.True(t, handled)
	assert.Equal(t, "123", capturedParams["nodeID"])
	assert.Equal(t, "ok", writer.lastContent)
}

func TestMessageMux_Dispatch_NoMatch(t *testing.T) {
	mux := NewMessageMux()
	handled := false

	// Register a handler
	pattern := NewPattern("node/{nodeID}").Op("update")
	mux.Entry(pattern, func(container *MessageContainer, writer ResponseWriter) {
		handled = true
	})

	// Create a request that DOES NOT match
	msg := &model.Message{
		Router: model.MessageRoute{
			Resource:  "pod/123",
			Operation: "update",
		},
	}
	req := &MessageRequest{Message: msg}
	writer := &mockResponseWriter{}

	// Dispatch
	mux.ServeConn(req, writer)

	assert.False(t, handled)
}

func TestEntry_Global(t *testing.T) {
	// Test the global Entry function
	// We need to be careful as MuxDefault is a global variable
	
	pattern := NewPattern("global/test").Op("get")
	Entry(pattern, func(container *MessageContainer, writer ResponseWriter) {})

	assert.NotEmpty(t, MuxDefault.muxEntry)
	
	// Check if the last entry matches what we added
	lastEntry := MuxDefault.muxEntry[len(MuxDefault.muxEntry)-1]
	assert.Equal(t, "global/test", lastEntry.pattern.resource)
	assert.Equal(t, "get", lastEntry.pattern.operation)
}
