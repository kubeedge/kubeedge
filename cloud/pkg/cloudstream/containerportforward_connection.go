package cloudstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

// ContainerPortForwardConnection indicates the container port-forward request initiated by kube-apiserver
type ContainerPortForwardConnection struct {
	MessageID    uint64
	ctx          context.Context
	r            *restful.Request
	Conn         net.Conn
	session      *Session
	edgePeerStop chan struct{}
	closeChan    chan bool
}

func (c *ContainerPortForwardConnection) String() string {
	return fmt.Sprintf("APIServer_PortForwardConnection MessageID %v", c.MessageID)
}

func (c *ContainerPortForwardConnection) WriteToAPIServer(p []byte) (n int, err error) {
	return c.Conn.Write(p)
}

func (c *ContainerPortForwardConnection) SetMessageID(id uint64) {
	c.MessageID = id
}

func (c *ContainerPortForwardConnection) GetMessageID() uint64 {
	return c.MessageID
}

func (c *ContainerPortForwardConnection) SetEdgePeerDone() {
	select {
	case <-c.closeChan:
		return
	case c.EdgePeerDone() <- struct{}{}:
		klog.V(6).Infof("success send channel deleting connection with messageID %v", c.MessageID)
	}
}

func (c *ContainerPortForwardConnection) EdgePeerDone() chan struct{} {
	return c.edgePeerStop
}

func (c *ContainerPortForwardConnection) WriteToTunnel(m *stream.Message) error {
	return c.session.WriteMessageToTunnel(m)
}

func (c *ContainerPortForwardConnection) SendConnection() (stream.EdgedConnection, error) {
	connector := &stream.EdgedPortForwardConnection{
		MessID: c.MessageID,
		Method: c.r.Request.Method,
		URL:    *c.r.Request.URL,
		Header: c.r.Request.Header,
	}
	connector.URL.Scheme = httpScheme
	connector.URL.Host = net.JoinHostPort(defaultServerHost, fmt.Sprintf("%v", constants.ServerPort))

	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}
	if err := c.WriteToTunnel(m); err != nil {
		klog.Errorf("%s failed to create portForward connection: %s, err: %v", c.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (c *ContainerPortForwardConnection) Serve() error {
	defer func() {
		close(c.closeChan)
		klog.V(6).Infof("%s stop successfully", c.String())
	}()

	connector, err := c.SendConnection()
	if err != nil {
		klog.Errorf("%s send %s info error %v", c.String(), stream.MessageTypePortForwardConnect, err)
		return err
	}

	sendCloseMessage := func() {
		msg := stream.NewMessage(c.MessageID, stream.MessageTypeRemoveConnect, nil)
		for retry := 0; retry < 3; retry++ {
			if err := c.WriteToTunnel(msg); err == nil {
				klog.V(6).Infof("%s send close message to edge successfully", c.String())
				return
			}
			klog.Warningf("%v failed send %s message to edge, err: %v", c, msg.MessageType, err)
		}
		klog.Errorf("max retry count reached when send %s message to edge", msg.MessageType)
	}

	var data [256]byte
	for {
		select {
		case <-c.ctx.Done():
			sendCloseMessage()
			return nil
		case <-c.EdgePeerDone():
			klog.V(6).Infof("%s find edge peer done, so stop this connection", c.String())
			return fmt.Errorf("%s find edge peer done, so stop this connection", c.String())
		default:
		}

		n, err := c.Conn.Read(data[:])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				klog.Errorf("%s failed to read from client: %v", c.String(), err)
				return err
			}
			klog.V(6).Infof("%s read EOF from client", c.String())
			sendCloseMessage()
			return nil
		}
		if n <= 0 {
			continue
		}

		msg := stream.NewMessage(connector.GetMessageID(), stream.MessageTypeData, data[:n])
		if err := c.WriteToTunnel(msg); err != nil {
			klog.Errorf("%s failed to write to tunnel server, err: %v", c.String(), err)
			return err
		}
	}
}

var _ APIServerConnection = &ContainerPortForwardConnection{}
