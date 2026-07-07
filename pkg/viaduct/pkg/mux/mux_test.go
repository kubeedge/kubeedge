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
	"crypto/x509"
	"errors"
	"net/http"
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/filter"
)

type fakeResponseWriter struct{}

func (fakeResponseWriter) WriteResponse(*model.Message, interface{}) {}

func (fakeResponseWriter) WriteError(*model.Message, string) {}

func TestMessageExpressionGetExpression(t *testing.T) {
	expression := NewExpression().GetExpression("/nodes/{node}/pods/{pod}")
	if expression == nil {
		t.Fatal("expected expression, got nil")
	}

	if expression.VarCount != 2 {
		t.Fatalf("expected 2 variables, got %d", expression.VarCount)
	}

	expectedVarNames := []string{"node", "pod"}
	if len(expression.VarNames) != len(expectedVarNames) {
		t.Fatalf("expected variable names %v, got %v", expectedVarNames, expression.VarNames)
	}
	for i, expected := range expectedVarNames {
		if expression.VarNames[i] != expected {
			t.Fatalf("expected variable names %v, got %v", expectedVarNames, expression.VarNames)
		}
	}

	expectedVarIndexes := []int{1, 2}
	for i, expected := range expectedVarIndexes {
		if expression.VarIndexes[i] != expected {
			t.Fatalf("expected variable indexes %v, got %v", expectedVarIndexes, expression.VarIndexes)
		}
	}

	if !expression.Matcher.MatchString("/nodes/node-1/pods/pod-1") {
		t.Fatal("expected expression to match resource")
	}
	if expression.Matcher.MatchString("/nodes/node-1") {
		t.Fatal("expected expression not to match incomplete resource")
	}
}

func TestMessagePatternMatch(t *testing.T) {
	msg := model.NewMessage("").SetResourceOperation("/nodes/node-1/pods/pod-1", "get")

	if !NewPattern("/nodes/{node}/pods/{pod}").Op("get").Match(msg) {
		t.Fatal("expected pattern to match resource and operation")
	}

	if NewPattern("/nodes/{node}/pods/{pod}").Op("update").Match(msg) {
		t.Fatal("expected pattern not to match different operation")
	}

	if !NewPattern("/nodes/{node}/pods/{pod}").Op("*").Match(msg) {
		t.Fatal("expected wildcard operation to match")
	}
}

func TestMessagePatternResRebuildsExpression(t *testing.T) {
	pattern := NewPattern("/old").Res("/new").Op("get")
	if pattern == nil {
		t.Fatal("expected pattern, got nil")
	}

	oldMsg := model.NewMessage("").SetResourceOperation("/old", "get")
	if pattern.Match(oldMsg) {
		t.Fatal("expected Res to stop matching the original resource")
	}

	newMsg := model.NewMessage("").SetResourceOperation("/new", "get")
	if !pattern.Match(newMsg) {
		t.Fatal("expected Res to match the updated resource")
	}
}

func TestMessagePatternMatchBranches(t *testing.T) {
	tests := []struct {
		name     string
		pattern  *MessagePattern
		resource string
		want     bool
	}{
		{
			name:     "resource wildcard",
			pattern:  NewPattern("*").Op("get"),
			resource: "/nodes/node-1/pods/pod-1",
			want:     true,
		},
		{
			name:     "wildcard parameter",
			pattern:  NewPattern("/nodes/{path:*}").Op("get"),
			resource: "/nodes/node-1/pods/pod-1",
			want:     true,
		},
		{
			name:     "valid custom regular expression",
			pattern:  NewPattern("/nodes/{node:node-[0-9]+}").Op("get"),
			resource: "/nodes/node-1",
			want:     true,
		},
		{
			name:     "custom regular expression mismatch",
			pattern:  NewPattern("/nodes/{node:node-[0-9]+}").Op("get"),
			resource: "/nodes/edge-1",
			want:     false,
		},
		{
			name:     "trailing sub-resource",
			pattern:  NewPattern("/nodes/{node}").Op("get"),
			resource: "/nodes/node-1/pods/pod-1",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pattern == nil {
				t.Fatal("expected pattern, got nil")
			}

			msg := model.NewMessage("").SetResourceOperation(tt.resource, "get")
			if got := tt.pattern.Match(msg); got != tt.want {
				t.Fatalf("expected match %t, got %t", tt.want, got)
			}
		})
	}
}

func TestMessagePatternInvalidRegex(t *testing.T) {
	if pattern := NewPattern("/nodes/{node:[}"); pattern != nil {
		t.Fatalf("expected invalid regex to return nil, got %#v", pattern)
	}
}

func TestMessagePatternMatchNilMessage(t *testing.T) {
	if NewPattern("/nodes/{node}").Op("get").Match(nil) {
		t.Fatal("expected nil message not to match")
	}
}

func TestMessageMuxExtractParameters(t *testing.T) {
	pattern := NewPattern("/nodes/{node}/pods/{pod}").Op("get")
	params := NewMessageMux().extractParameters(pattern.resExpr, "/nodes/node-1/pods/pod-1")

	if params["node"] != "node-1" {
		t.Fatalf("expected node parameter node-1, got %q", params["node"])
	}
	if params["pod"] != "pod-1" {
		t.Fatalf("expected pod parameter pod-1, got %q", params["pod"])
	}
}

func TestMessageMuxExtractParametersWithCustomRegexCaptureGroup(t *testing.T) {
	pattern := NewPattern("/nodes/{node:(foo|bar)}/pods/{pod}").Op("get")
	if pattern == nil {
		t.Fatal("expected pattern, got nil")
	}

	params := NewMessageMux().extractParameters(pattern.resExpr, "/nodes/foo/pods/pod-1")
	if params["node"] != "foo" {
		t.Fatalf("expected node parameter foo, got %q", params["node"])
	}
	if params["pod"] != "pod-1" {
		t.Fatalf("expected pod parameter pod-1, got %q", params["pod"])
	}
}

func TestMessageMuxDispatch(t *testing.T) {
	messageMux := NewMessageMux()
	msg := model.NewMessage("").SetResourceOperation("/nodes/node-1/pods/pod-1", "get")
	header := http.Header{}
	header.Set("node_id", "node-1")
	header.Set("project_id", "project-1")
	certs := []*x509.Certificate{{}}
	req := &MessageRequest{
		Message:          msg,
		Header:           header,
		PeerCertificates: certs,
	}

	var handled bool
	messageMux.Entry(NewPattern("/nodes/{node}/pods/{pod}").Op("get"), func(container *MessageContainer, writer ResponseWriter) {
		handled = true
		if container.Message != msg {
			t.Fatal("expected original message pointer to be propagated")
		}
		if container.Header.Get("node_id") != "node-1" {
			t.Fatalf("expected node_id header node-1, got %q", container.Header.Get("node_id"))
		}
		if container.Header.Get("project_id") != "project-1" {
			t.Fatalf("expected project_id header project-1, got %q", container.Header.Get("project_id"))
		}
		if len(container.PeerCertificates) != 1 || container.PeerCertificates[0] != certs[0] {
			t.Fatalf("expected peer certificates to be propagated, got %#v", container.PeerCertificates)
		}
		if container.Parameter("node") != "node-1" {
			t.Fatalf("expected node parameter node-1, got %q", container.Parameter("node"))
		}
		if container.Parameter("pod") != "pod-1" {
			t.Fatalf("expected pod parameter pod-1, got %q", container.Parameter("pod"))
		}
	})

	if err := messageMux.dispatch(req, fakeResponseWriter{}); err != nil {
		t.Fatalf("expected no dispatch error, got %v", err)
	}
	if !handled {
		t.Fatal("expected handler to be called")
	}
}

func TestMessageMuxDispatchReturnsErrorWithoutMatch(t *testing.T) {
	messageMux := NewMessageMux()
	msg := model.NewMessage("").SetResourceOperation("/nodes/node-1/pods/pod-1", "get")
	req := &MessageRequest{Message: msg}

	messageMux.Entry(NewPattern("/nodes/{node}/pods/{pod}").Op("update"), func(*MessageContainer, ResponseWriter) {
		t.Fatal("handler should not be called")
	})

	if err := messageMux.dispatch(req, fakeResponseWriter{}); err == nil {
		t.Fatal("expected dispatch error without matching entry")
	}
}

func TestMessageMuxServeConnFilterRejectsMessage(t *testing.T) {
	messageMux := NewMessageMux()
	messageFilter := &filter.MessageFilter{}
	messageFilter.AddFilterFunc(func(*model.Message) error {
		return errors.New("rejected")
	})
	messageMux.AddFilter(messageFilter)

	var handled bool
	messageMux.Entry(NewPattern("*").Op("*"), func(*MessageContainer, ResponseWriter) {
		handled = true
	})

	req := &MessageRequest{Message: model.NewMessage("").SetResourceOperation("/nodes/node-1", "get")}
	messageMux.ServeConn(req, fakeResponseWriter{})
	if handled {
		t.Fatal("expected filtered message not to be dispatched")
	}
}
