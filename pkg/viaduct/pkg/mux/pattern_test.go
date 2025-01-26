package mux

import (
    "testing"
    "github.com/kubeedge/beehive/pkg/core/model"
)

func TestPattern(t *testing.T) {
    t.Run("creation", func(t *testing.T) {
        testCases := []struct {
            name     string
            resource string
            wantNil  bool
        }{
            {"valid_path", "/devices/light", false},
            {"valid_var_path", "/devices/{id}", false},
            {"empty_path", "", false},
        }

        for _, tc := range testCases {
            t.Run(tc.name, func(t *testing.T) {
                pattern := NewPattern(tc.resource)
                if pattern == nil {
                    t.Fatal("pattern should not be nil")
                }
            })
        }
    })

    t.Run("resource_operation", func(t *testing.T) {
        pattern := NewPattern("/devices/{id}")
        pattern.resource = "/devices/{id}"
        pattern.Op("get")

        msg := &model.Message{
            Router: model.MessageRoute{
                Resource:  "/devices/123",
                Operation: "get",
            },
        }

        if !pattern.Match(msg) {
            t.Error("pattern should match message")
        }
    })

    t.Run("operation_matching", func(t *testing.T) {
        testCases := []struct {
            name        string
            resource    string
            op         string
            msgResource string
            msgOp      string
            shouldMatch bool
        }{
            {"exact_match", "/test", "get", "/test", "get", true},
            {"wildcard_op", "/test", "*", "/test", "any", true},
            {"no_match_op", "/test", "get", "/test", "post", false},
            {"no_match_resource", "/test", "get", "/other", "get", false},
        }

        for _, tc := range testCases {
            t.Run(tc.name, func(t *testing.T) {
                pattern := NewPattern(tc.resource)
                pattern.Op(tc.op)

                msg := &model.Message{
                    Router: model.MessageRoute{
                        Resource:  tc.msgResource,
                        Operation: tc.msgOp,
                    },
                }

                if got := pattern.Match(msg); got != tc.shouldMatch {
                    t.Errorf("Match() = %v, want %v", got, tc.shouldMatch)
                }
            })
        }
    })
}