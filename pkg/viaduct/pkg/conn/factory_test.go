package conn

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestFactory_NewConnection(t *testing.T) {
    t.Run("Nil_Options", func(t *testing.T) {
        conn := NewConnection(nil)
        assert.Nil(t, conn, "Should return nil for nil options")
    })

    t.Run("Invalid_Protocol", func(t *testing.T) {
        opts := &ConnectionOptions{
            ConnType: "invalid",
        }
        conn := NewConnection(opts)
        assert.Nil(t, conn, "Should return nil for invalid protocol")
    })

    t.Run("Empty_Protocol", func(t *testing.T) {
        opts := &ConnectionOptions{}
        conn := NewConnection(opts)
        assert.Nil(t, conn, "Should return nil for empty protocol")
    })
}