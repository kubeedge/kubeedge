package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/quic-go/quic-go"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/lane"
)

// the client based on quic
type QuicClient struct {
	options  Options
	exOpts   api.QuicClientOption
	ctrlLane lane.Lane
	laneLock sync.Mutex
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
		HandshakeIdleTimeout: c.options.HandshakeTimeout,
		MaxIdleTimeout: c.options.HandshakeTimeout * 2,
	}
}

// the basic lan for connection control
// never be closed
func (c *QuicClient) getControlLane(s quic.Connection) error {
	c.laneLock.Lock()
	defer c.laneLock.Unlock()

	if c.ctrlLane != nil {
		return nil
	}

	stream, err := s.OpenStream()
	if err != nil {
		klog.Errorf("open control stream error(%+v)", err)
		return fmt.Errorf("open control stream")
	}

	c.ctrlLane = lane.NewLane(api.ProtocolTypeQuic, stream)
	return nil
}

// send the headers
// TODO: add timeout?
func (c *QuicClient) sendHeader() error {
	msg := model.NewMessage("").
		BuildRouter("", "", comm.ControlTypeHeader, comm.ControlTypeHeader).
		FillBody(c.exOpts.Header)
	err := c.ctrlLane.WriteMessage(msg)
	if err != nil {
		klog.Errorf("failed to write message, error: %+v", err)
		return err
	}

	// receive the response
	// ignore the response
	// TODO: check the response content
	var response model.Message
	err = c.ctrlLane.ReadMessage(&response)
	if err != nil {
		klog.Errorf("failed to read message, error: %+v", err)
		return err
	}
	klog.Infof("get response: %+v", response)
	return nil
}

// try to dial server and get connection interface for operations
func (c *QuicClient) Connect() (conn.Connection, error) {
	ctx := context.Background()
	quicConfig := c.getQuicConfig()
	session, err := quic.DialAddr(
        ctx,
        c.options.Addr, 
        c.options.TLSConfig, 
        quicConfig,
    )
	if err != nil {
		klog.Errorf("failed dial addr %s, error:%+v", c.options.Addr, err)
		return nil, err
	}

	// get control lan
	err = c.getControlLane(session)
	if err != nil {
		session.CloseWithError(0, "failed to get control lane")
		return nil, err
	}

	// send headers
	err = c.sendHeader()
	if err != nil {
		klog.Warningf("failed to send headers, error: %+v", err)
	}

	klog.Info("connect remote peer successfully")
	return conn.NewConnection(&conn.ConnectionOptions{
		ConnType: api.ProtocolTypeQuic,
		ConnUse:  c.options.ConnUse,
		Base:     session,
		CtrlLane: c.ctrlLane,
		Consumer: c.options.Consumer,
		Handler:  c.options.Handler,
		State: &conn.ConnectionState{
			State:            api.StatConnected,
			Headers:          c.exOpts.Header,
			PeerCertificates: session.ConnectionState().TLS.PeerCertificates,
		},
		AutoRoute: c.options.AutoRoute,
	}), nil
}
