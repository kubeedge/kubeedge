package mux

import (
    "testing"
)

func TestMessageMuxEntry(t *testing.T) {
    t.Run("new_entry", func(t *testing.T) {
        pattern := &MessagePattern{resource: "test"}
        handler := func(msg *MessageContainer, w ResponseWriter) {}
        
        entry := NewEntry(pattern, handler)
        if entry.pattern != pattern {
            t.Errorf("pattern mismatch: got %v, want %v", entry.pattern, pattern)
        }
        if entry.handleFunc == nil {
            t.Error("handler function should not be nil")
        }
    })

    t.Run("pattern_setter", func(t *testing.T) {
        entry := &MessageMuxEntry{}
        pattern := &MessagePattern{resource: "test"}
        
        result := entry.Pattern(pattern)
        if result != entry {
            t.Error("Pattern() should return entry for chaining")
        }
        if entry.pattern != pattern {
            t.Errorf("pattern not set correctly: got %v, want %v", entry.pattern, pattern)
        }
    })

    t.Run("handle_setter", func(t *testing.T) {
        entry := &MessageMuxEntry{}
        handler := func(msg *MessageContainer, w ResponseWriter) {}
        
        result := entry.Handle(handler)
        if result != entry {
            t.Error("Handle() should return entry for chaining")
        }
        if entry.handleFunc == nil {
            t.Error("handler function should not be nil")
        }
    })

    t.Run("chaining_operations", func(t *testing.T) {
        pattern1 := &MessagePattern{resource: "test1"}
        pattern2 := &MessagePattern{resource: "test2"}
        handler := func(msg *MessageContainer, w ResponseWriter) {}
        
        entry := NewEntry(pattern1, handler).Pattern(pattern2)
        if entry.pattern != pattern2 {
            t.Error("chained pattern update failed")
        }
    })
}