package comm

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCommConstants(t *testing.T) {
    t.Run("TestControlMessageTypes", func(t *testing.T) {
        assert.Equal(t, "header", ControlTypeHeader)
        assert.Equal(t, "config", ControlTypeConfig)
        assert.Equal(t, "ping", ControlTypePing)
        assert.Equal(t, "pong", ControlTypePong)
    })

    t.Run("TestControlActions", func(t *testing.T) {
        assert.Equal(t, "/control/header", ControlActionHeader)
        assert.Equal(t, "/control/config", ControlActionConfig)
        assert.Equal(t, "/control/ping", ControlActionPing)
        assert.Equal(t, "/control/pong", ControlActionPong)
    })

    t.Run("TestResponseTypes", func(t *testing.T) {
        assert.Equal(t, "ack", RespTypeAck)
        assert.Equal(t, "nack", RespTypeNack)
    })

    t.Run("TestSizeLimits", func(t *testing.T) {
        assert.Equal(t, 100, MessageFiFoSizeMax)
        assert.Equal(t, 1<<20, MaxReadLength) // 1 MiB
    })
}

func TestErrorCodes(t *testing.T) {
    t.Run("TestStatusCodes", func(t *testing.T) {
        assert.Equal(t, 0, StatusCodeNoError)
        assert.Equal(t, 1, StatusCodeFreeStream)
    })
}