package cloudstream

import (
	"fmt"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	httpScheme        = "http"
	defaultServerHost = "127.0.0.1"
)

// APIServerConnection indicates a connection request originally made by kube-apiserver to kubelet
// There are basically three types of connection requests : containersLogs, containerExec, Metric
// Cloudstream module first intercepts the connection request and then sends the request data through the tunnel (websocket) to edgestream module
type APIServerConnection interface {
	fmt.Stringer
	// SendConnection indicates send EdgedConnection to edge
	SendConnection() (stream.EdgedConnection, error)
	// WriteToTunnel indicates writing message to tunnel
	WriteToTunnel(m *stream.Message) error
	// WriteToAPIServer indicates writing data to apiserver response
	WriteToAPIServer(p []byte) (n int, err error)
	// SetMessageID indicates set messageid for it`s connection
	// Every APIServerConnection has his unique message id
	SetMessageID(id uint64)
	GetMessageID() uint64
	// Serve indicates handling his own logic
	Serve() error
	// SetEdgePeerDone indicates send specifical message to let edge peer exist
	SetEdgePeerDone()
	// EdgePeerDone indicates whether edge peer ends
	EdgePeerDone() <-chan struct{}
}
