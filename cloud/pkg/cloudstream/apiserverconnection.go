/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
