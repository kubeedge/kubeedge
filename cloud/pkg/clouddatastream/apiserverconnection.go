/*
Copyright 2025 The KubeEdge Authors.

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
