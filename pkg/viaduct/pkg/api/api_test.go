package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/gorilla/websocket"
)

// validate critical constants so that they are not changed
func TestProtocolTypes (t *testing.T) {
	t.Run("Protocol Constants", func(t *testing.T) {
		assert.Equal(t, "quic", ProtocolTypeQuic)
		assert.Equal(t, "websocket", ProtocolTypeWS)
	})

	t.Run("Connection Stats", func(t *testing.T){
		assert.Equal(t, "connected", StatConnected)
		assert.Equal(t, "disconnected", StatDisconnected)
	})
	t.Run("UseType Constants", func(t *testing.T) {
        assert.Equal(t, UseType("msg"), UseTypeMessage)
        assert.Equal(t, UseType("str"), UseTypeStream)
        assert.Equal(t, UseType("shr"), UseTypeShare)
        assert.Equal(t, 3, UseLen)
    })
}


func TestQuicOption(t *testing.T) {
	t.Run("Quic Client Option", func(t *testing.T) {
		header := http.Header{"Test": []string{"value"}}
        opt := QuicClientOption{
            Header: header,
            MaxIncomingStreams: 100,
        }
        assert.Equal(t, header, opt.Header)
        assert.Equal(t, 100, opt.MaxIncomingStreams)
	})

	t.Run("Quic Server Option", func(t *testing.T) {
		opt := QuicServerOption{
			MaxIncomingStreams: 100,
		}
		assert.Equal(t, 100, opt.MaxIncomingStreams)
	})
}

func TestWebSocketOptions(t *testing.T) {
    t.Run("WSClientOption", func(t *testing.T) {
        header := http.Header{"Test": []string{"value"}}
        var called bool
        callback := func(conn *websocket.Conn, resp *http.Response) {
            called = true
        }
        
        opt := WSClientOption{
            Header: header,
            Callback: callback,
        }
        
        assert.Equal(t, header, opt.Header)
        assert.NotNil(t, opt.Callback)
        
        opt.Callback(nil, nil)
        assert.True(t, called)
    })

	t.Run("WSServerOption", func(t *testing.T) {
        var filtered bool
        filter := func(w http.ResponseWriter, r *http.Request) bool {
            filtered = true
            return true
        }
        
        opt := WSServerOption{
            Path: "/test",
            Filter: filter,
        }
        
        assert.Equal(t, "/test", opt.Path)
        assert.NotNil(t, opt.Filter)
        
        result := opt.Filter(nil, nil)
        assert.True(t, filtered)
        assert.True(t, result)
    })
}