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

	"github.com/kubeedge/beehive/pkg/core/model"
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

func TestMessageMuxDispatch(t *testing.T) {
	messageMux := NewMessageMux()
	msg := model.NewMessage("").SetResourceOperation("/nodes/node-1/pods/pod-1", "get")
	req := &MessageRequest{Message: msg}

	var handled bool
	messageMux.Entry(NewPattern("/nodes/{node}/pods/{pod}").Op("get"), func(container *MessageContainer, writer ResponseWriter) {
		handled = true
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
