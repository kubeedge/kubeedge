package clouddatastream

import (
	"fmt"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

type APIServerConnection interface {
	fmt.Stringer
	SendConnection() (stream.EdgedConnection, error)
	WriteToTunnel(m *stream.Message) error
	WriteToAPIServer(p []byte) (n int, err error)
	SetMessageID(id uint64)
	GetMessageID() uint64
	Serve() error
	SetEdgePeerDone()
	EdgePeerDone() chan struct{}
}
