package client

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/lucas-clemente/quic-go"
)

// the client based on quic
type QuicClient struct {
	options Options
	exOpts  api.QuicClientOption
}

// new a quic client instance
func NewQuicClient(opts Options, exOpts interface{}) *QuicClient {
	extendOptions, ok := exOpts.(api.QuicClientOption)
	if !ok {
		panic("bad extend options type")
	}

	return &QuicClient{
		options: opts,
		exOpts:  extendOptions,
	}
}

// get quic config
// TODO: add additional options
func (c *QuicClient) getQuicConfig() *quic.Config {
	return &quic.Config{
		HandshakeTimeout: c.options.HandshakeTimeout,
		// keep the session by default
		KeepAlive: true,
	}
}

// try to dial server and get connection interface for operations
func (c *QuicClient) Connect() (conn.Connection, error) {
	quicConfig := c.getQuicConfig()
	session, err := quic.DialAddr(c.options.Addr, c.options.TLSConfig, quicConfig)
	if err != nil {
		log.LOGGER.Errorf("failed dial addr %s, error:%+v", c.options.Addr, err)
		return nil, err
	}

	return conn.NewConnection(&conn.ConnectionOptions{
		ConnType: api.ProtocolTypeQuic,
		Base:     session,
		Handler:  c.options.Handler,
	}), nil
}
