package cloudstream

import (
	"context"
	"fmt"
	"net"

	"github.com/kubeedge/kubeedge/common/constants"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/stream"

	"github.com/emicklei/go-restful"
)

// ContainerExecConnection indicates the container exec request initiated by kube-apiserver
type ContainerExecConnection struct {
	// MessageID indicate the unique id to create his message
	MessageID    uint64
	ctx          context.Context
	r            *restful.Request
	Conn         net.Conn
	session      *Session
	edgePeerStop chan struct{}
}

func (c *ContainerExecConnection) String() string {
	return fmt.Sprintf("APIServer_ExecConnection MessageID %v", c.MessageID)
}

func (c *ContainerExecConnection) WriteToAPIServer(p []byte) (n int, err error) {
	return c.Conn.Write(p)
}

func (c *ContainerExecConnection) SetMessageID(id uint64) {
	c.MessageID = id
}

func (c *ContainerExecConnection) GetMessageID() uint64 {
	return c.MessageID
}

func (c *ContainerExecConnection) SetEdgePeerDone() {
	close(c.edgePeerStop)
}

func (c *ContainerExecConnection) EdgePeerDone() <-chan struct{} {
	return c.edgePeerStop
}

func (c *ContainerExecConnection) WriteToTunnel(m *stream.Message) error {
	return c.session.WriteMessageToTunnel(m)
}

func (c *ContainerExecConnection) SendConnection() (stream.EdgedConnection, error) {
	connector := &stream.EdgedExecConnection{
		MessID: c.MessageID,
		Method: c.r.Request.Method,
		URL:    *c.r.Request.URL,
		Header: c.r.Request.Header,
	}
	connector.URL.Scheme = "http"
	connector.URL.Host = net.JoinHostPort("127.0.0.1", fmt.Sprintf("%v", constants.ServerPort))
	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}
	if err := c.WriteToTunnel(m); err != nil {
		klog.Errorf("%s write %s error %v", c.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (c *ContainerExecConnection) Serve() error {
	defer func() {
		klog.Infof("%s end successful", c.String())
	}()

	// first send connect message
	if _, err := c.SendConnection(); err != nil {
		klog.Errorf("%s send %s info error %v", c.String(), stream.MessageTypeExecConnect, err)
		return err
	}

	for {
		select {
		case <-c.ctx.Done():
			// if apiserver request end, send close message to edge
			msg := stream.NewMessage(c.MessageID, stream.MessageTypeRemoveConnect, nil)
			for retry := 0; retry < 3; retry++ {
				if err := c.WriteToTunnel(msg); err != nil {
					klog.Warningf("%v send %s message to edge error %v", c, msg.MessageType, err)
				} else {
					break
				}
			}
			klog.Infof("%s send close message to edge successfully", c.String())
			return nil
		case <-c.EdgePeerDone():
			klog.Infof("%s find edge peer done, so stop this connection", c.String())
			return nil
		}
	}
}

var _ APIServerConnection = &ContainerExecConnection{}
