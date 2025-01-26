package client

import (
    "crypto/tls"
    "net/http"
    "testing"
    "time"

    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
    
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestQuicClient(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    t.Run("TestNewQuicClient", func(t *testing.T) {
        opts := Options{
            Type: api.ProtocolTypeQuic,
            Addr: "localhost:8080",
            HandshakeTimeout: time.Second * 30,
            TLSConfig: &tls.Config{},
        }
        exOpts := api.QuicClientOption{
            Header: http.Header{"Test": []string{"value"}},
            MaxIncomingStreams: 100,
        }
        
        client := NewQuicClient(opts, exOpts)
        assert.NotNil(t, client)
        assert.Equal(t, opts, client.options)
        assert.Equal(t, exOpts, client.exOpts)
    })

    t.Run("TestQuicConfig", func(t *testing.T) {
        client := &QuicClient{
            options: Options{HandshakeTimeout: time.Second * 30},
        }
        config := client.getQuicConfig()
        
        assert.Equal(t, time.Second*30, config.HandshakeIdleTimeout)
        assert.Equal(t, time.Second*60, config.MaxIdleTimeout)
    })

    t.Run("TestQuicClientConnect_Error", func(t *testing.T) {
        client := &QuicClient{
            options: Options{
                Addr: "invalid-addr",
                TLSConfig: &tls.Config{},
                HandshakeTimeout: time.Second * 30,
            },
        }
        
        conn, err := client.Connect()
        assert.Error(t, err)
        assert.Nil(t, conn)
    })
}