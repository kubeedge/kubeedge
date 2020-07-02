package cloudstream

import (
	"context"
	"fmt"
	"net"

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
	panic("implement me")
}

func (c *ContainerExecConnection) Serve() error {
	panic("implement me")
}

var _ APIServerConnection = &ContainerExecConnection{}
