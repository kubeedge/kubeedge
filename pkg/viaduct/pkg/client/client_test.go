package client

import (
    "testing"
    "time"
    "crypto/tls"
    
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestClient(t *testing.T) {
    t.Run("TestClientCreation", func(t *testing.T) {
        opts := Options{
            Type: api.ProtocolTypeQuic,
            Addr: "localhost:8080",
            ConnUse: api.UseTypeStream,
            TLSConfig: &tls.Config{},
            HandshakeTimeout: time.Second * 30,
        }
        client := &Client{Options: opts}
        assert.NotNil(t, client)
    })

    t.Run("TestProtocolSelection", func(t *testing.T) {
        client := &Client{
            Options: Options{Type: api.ProtocolTypeQuic},
            ExOpts: api.QuicClientOption{},
        }
        conn, err := client.Connect()
        assert.Error(t, err)
        assert.Nil(t, conn)
    })
}