package mux

import (
    "testing"
    "net/http"
    "github.com/kubeedge/beehive/pkg/core/model"
)

type mockResponseWriter struct {
    responseMsg *model.Message
    errorMsg    string
}

func (w *mockResponseWriter) WriteResponse(msg *model.Message, content interface{}) {
    w.responseMsg = msg
}

func (w *mockResponseWriter) WriteError(msg *model.Message, err string) {
    w.errorMsg = err
}

func TestMessageMux(t *testing.T) {
    t.Run("register_and_dispatch", func(t *testing.T) {
        mux := NewMessageMux()
        writer := &mockResponseWriter{}
        handled := false

        // Register handler
        pattern := NewPattern("/devices/{id}")
        mux.Entry(pattern, func(c *MessageContainer, w ResponseWriter) {
            handled = true
            if c.Parameter("id") != "123" {
                t.Errorf("parameter id = %s, want 123", c.Parameter("id"))
            }
        })

        // Create test request
        req := &MessageRequest{
            Message: &model.Message{
                Router: model.MessageRoute{
                    Resource: "/devices/123",
                },
            },
            Header: make(http.Header),
        }

        // Dispatch request
        err := mux.dispatch(req, writer)
        if err != nil {
            t.Errorf("dispatch error: %v", err)
        }
        if !handled {
            t.Error("handler was not called")
        }
    })

    t.Run("parameter_extraction", func(t *testing.T) {
        mux := NewMessageMux()
        expr := NewExpression().GetExpression("/devices/{id}/status/{type}")
        params := mux.extractParameters(expr, "/devices/123/status/temp")

        expected := map[string]string{
            "id":   "123",
            "type": "temp",
        }

        for k, v := range expected {
            if params[k] != v {
                t.Errorf("param[%s] = %s, want %s", k, params[k], v)
            }
        }
    })

    t.Run("no_matching_entry", func(t *testing.T) {
        mux := NewMessageMux()
        writer := &mockResponseWriter{}
        req := &MessageRequest{
            Message: &model.Message{
                Router: model.MessageRoute{
                    Resource: "/unknown/path",
                },
            },
        }

        err := mux.dispatch(req, writer)
        if err == nil {
            t.Error("expected error for non-matching path")
        }
    })
}