package client

import (
    "crypto/tls"
    "net/http"
    "testing"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestWSClient(t *testing.T) {
    t.Run("TestNewWSClient", func(t *testing.T) {
        opts := Options{
            Type: api.ProtocolTypeWS,
            Addr: "ws://localhost:8080",
            HandshakeTimeout: time.Second * 30,
            TLSConfig: &tls.Config{},
        }
        
        exOpts := api.WSClientOption{
            Header: http.Header{"Test": []string{"value"}},
        }
        
        client := NewWSClient(opts, exOpts)
        assert.NotNil(t, client)
        assert.Equal(t, opts, client.options)
        assert.Equal(t, exOpts, client.exOpts)
        assert.NotNil(t, client.dialer)
        assert.Equal(t, opts.TLSConfig, client.dialer.TLSClientConfig)
        assert.Equal(t, opts.HandshakeTimeout, client.dialer.HandshakeTimeout)
    })

    t.Run("TestWSClientConnect_Error", func(t *testing.T) {
        client := &WSClient{
            options: Options{
                Addr: "ws://invalid-addr",
                ConnUse: api.UseTypeStream,
            },
            exOpts: api.WSClientOption{
                Header: http.Header{},
            },
            dialer: &websocket.Dialer{},
        }
        
        conn, err := client.Connect()
        assert.Error(t, err)
        assert.Nil(t, conn)
    })

    t.Run("TestWSClientCallback", func(t *testing.T) {
        callbackCalled := false
        callback := func(conn *websocket.Conn, resp *http.Response) {
            callbackCalled = true
        }
        
        client := &WSClient{
            options: Options{
                Addr: "ws://localhost:8080",
            },
            exOpts: api.WSClientOption{
                Header: http.Header{},
                Callback: callback,
            },
        }
        
        assert.NotNil(t, client.exOpts.Callback)
        client.exOpts.Callback(nil, nil)
        assert.True(t, callbackCalled)
    })
}