package client

import (
	"fmt"
	"io/ioutil"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// the client based on websocket
type WSClient struct {
	options Options
	exOpts  api.WSClientOption
	dialer  *websocket.Dialer
}

// new websocket client instance
func NewWSClient(options Options, exOpts interface{}) *WSClient {
	extendOption, ok := exOpts.(api.WSClientOption)
	if !ok {
		panic("bad websocket extend option")
	}

	return &WSClient{
		options: options,
		exOpts:  extendOption,
		dialer: &websocket.Dialer{
			TLSClientConfig:  options.TLSConfig,
			HandshakeTimeout: options.HandshakeTimeout,
		},
	}
}

// Connect try to connect remote server
func (c *WSClient) Connect() (conn.Connection, error) {
	wsConn, resp, err := c.dialer.Dial(c.options.Addr, c.exOpts.Header)
	if err == nil {
		log.LOGGER.Infof("dial %s successfully", c.options.Addr)

		// do user's processing on connection or response
		if c.exOpts.Callback != nil {
			c.exOpts.Callback(wsConn, resp)
		}
		return conn.NewConnection(&conn.ConnectionOptions{
			ConnType: api.ProtocolTypeWS,
			Base:     wsConn,
			Handler:  c.options.Handler,
		}), nil
	}

	// something wrong!!
	var respMsg string
	if resp != nil {
		body, errRead := ioutil.ReadAll(resp.Body)
		if errRead != nil {
			respMsg = fmt.Sprintf("response code: %d, response body: %s", resp.StatusCode, string(body))
		} else {
			respMsg = fmt.Sprintf("response code: %d", resp.StatusCode)
		}
		resp.Body.Close()
		return nil, err
	}
	log.LOGGER.Errorf("dial websocket error(%+v), response message: %s", err, respMsg)

	return nil, err
}
