package broker

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/beehive/pkg/core/socket/synckeeper"
	"github.com/kubeedge/beehive/pkg/core/socket/wrapper"
)

const (
	syncMessageTimeoutDefault = 10 * time.Second
)

// RemoteBroker remote broker
type RemoteBroker struct {
	keeper *synckeeper.Keeper
}

// ConnectOptions connect options
type ConnectOptions struct {
	Address     string
	MessageType string
	BufferSize  int
	Cert        tls.Certificate

	// for websocket/http
	RequestHeader http.Header
}

// ConnectFunc connect func
type ConnectFunc func(ConnectOptions) (interface{}, error)

// NewRemoteBroker new remote broker
func NewRemoteBroker() *RemoteBroker {
	return &RemoteBroker{
		keeper: synckeeper.NewKeeper(),
	}
}

// Connect connect
func (broker *RemoteBroker) Connect(opts ConnectOptions, connect ConnectFunc) wrapper.Conn {
	conn, err := connect(opts)
	if err != nil {
		klog.Errorf("failed to connect, address: %s; error:%+v", opts.Address, err)
		return nil
	}
	return wrapper.NewWrapper(opts.MessageType, conn, opts.BufferSize)
}

// Send send
func (broker *RemoteBroker) Send(conn wrapper.Conn, message model.Message) error {
	//log.LOGGER.Infof("connection: %+v message: %+v", conn, message)
	err := conn.WriteJSON(&message)
	if err != nil {
		klog.Errorf("failed to write with error %+v", err)
		return fmt.Errorf("failed to write, error: %+v", err)
	}
	return nil
}

// Receive receive
func (broker *RemoteBroker) Receive(conn wrapper.Conn) (model.Message, error) {
	var message model.Message
	for {
		err := conn.SetReadDeadline(time.Time{})
		err = conn.ReadJSON(&message)
		if err != nil {
			klog.Errorf("failed to read, error:%+v", err)
			return model.Message{}, fmt.Errorf("failed to read, error: %+v", err)
		}

		isResponse := broker.keeper.IsSyncResponse(message.GetParentID())
		if !isResponse {
			return message, nil
		}

		err = broker.keeper.SendToKeepChannel(message)
	}
}

// SendSyncInternal sync mode
func (broker *RemoteBroker) SendSyncInternal(conn wrapper.Conn, message model.Message, timeout time.Duration) (model.Message, error) {
	if timeout <= 0 {
		timeout = syncMessageTimeoutDefault
	}

	// make sure to set sync flag
	message.Header.Sync = true

	err := conn.WriteJSON(&message)
	if err != nil {
		klog.Errorf("failed to write with error %+v", err)
		return model.Message{}, fmt.Errorf("failed to write, error: %+v", err)
	}

	deadline := time.Now().Add(timeout)
	err = conn.SetReadDeadline(deadline)
	var response model.Message
	err = conn.ReadJSON(&response)
	if err != nil {
		klog.Errorf("failed to read with error %+v", err)
		return model.Message{}, fmt.Errorf("failed to read, error: %+v", err)
	}

	return response, nil
}

// SendSync sync mode
func (broker *RemoteBroker) SendSync(conn wrapper.Conn, message model.Message, timeout time.Duration) (model.Message, error) {
	if timeout <= 0 {
		timeout = syncMessageTimeoutDefault
	}

	deadline := time.Now().Add(timeout)

	// make sure to set sync flag
	message.Header.Sync = true

	err := conn.WriteJSON(&message)
	if err != nil {
		klog.Errorf("failed to write with error %+v", err)
		return model.Message{}, fmt.Errorf("failed to write, error: %+v", err)
	}

	tempChannel := broker.keeper.AddKeepChannel(message.GetID())
	sendTimer := time.NewTimer(time.Until(deadline))
	select {
	case response := <-tempChannel:
		sendTimer.Stop()
		broker.keeper.DeleteKeepChannel(response.GetParentID())
		return response, nil
	case <-sendTimer.C:
		klog.Warningf("timeout to receive response for message: %s", message.String())
		broker.keeper.DeleteKeepChannel(message.GetID())
		return model.Message{}, fmt.Errorf("timeout to receive response for message: %s", message.String())
	}
}
