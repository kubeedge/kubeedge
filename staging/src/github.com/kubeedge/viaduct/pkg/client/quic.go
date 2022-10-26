package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/comm"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/lane"
)

// QuicClient the client based on quic
type QuicClient struct {
	options  Options
	exOpts   api.QuicClientOption
	ctrlLane lane.Lane
	laneLock sync.Mutex
}

// NewQuicClient new a quic client instance
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
		// keep the session by default
		KeepAlive: true,
	}
}

// the basic lan for connection control
// never be closed
func (c *QuicClient) getControlLane(s quic.Session) error {
	c.laneLock.Lock()
	defer c.laneLock.Unlock()

	if c.ctrlLane != nil {
		return nil
	}

	stream, err := s.OpenStreamSync(context.Background())
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
	var response model.Message
	err = c.ctrlLane.ReadMessage(&response)
	if err != nil {
		klog.Errorf("failed to read message, error: %+v", err)
		return err
	}
	result, ok := response.GetContent().([]byte)
	if !ok {
		klog.Errorf("invalid header response content: %T", response.GetContent())
		return fmt.Errorf("invalid header response content: %T", response.GetContent())
	}
	var resultRsp comm.ResponseContent
	err = json.Unmarshal(result, &resultRsp)
	if err != nil {
		klog.Errorf("send header response unmarshal error: %v", err)
		return fmt.Errorf("send header response unmarshal error: %v", err)
	}
	if resultRsp.Type == comm.RespTypeNack {
		klog.Errorf("send header response error: %v", resultRsp)
		return fmt.Errorf("send header response error: %v", resultRsp)
	}
	klog.Infof("get response: %+v", response)
	return nil
}

// Connect try to dial server and get connection interface for operations
func (c *QuicClient) Connect() (conn.Connection, error) {
	quicConfig := c.getQuicConfig()
	session, err := quic.DialAddr(c.options.Addr, c.options.TLSConfig, quicConfig)
	if err != nil {
		klog.Errorf("failed dial addr %s, error:%+v", c.options.Addr, err)
		return nil, err
	}

	// get control lan
	err = c.getControlLane(session)
	if err != nil {
		closeErr := session.CloseWithError(quic.ApplicationErrorCode(comm.StatusCodeNoError), "")
		if closeErr != nil {
			klog.Errorf("failed to close session, error:%+v", closeErr)
		}
		return nil, err
	}

	// send headers
	err = c.sendHeader()
	if err != nil {
		closeErr := session.CloseWithError(quic.ApplicationErrorCode(comm.StatusCodeNoError), "")
		if closeErr != nil {
			klog.Errorf("failed to close session, error:%+v", closeErr)
		}
		return nil, err
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
			State:   api.StatConnected,
			Headers: c.exOpts.Header,
		},
		AutoRoute: c.options.AutoRoute,
	}), nil
}
