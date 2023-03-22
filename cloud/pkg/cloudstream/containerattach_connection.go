/*
Copyright 2023 The KubeEdge Authors.

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
	"context"
	"fmt"
	"net"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// ContainerAttachConnection indicates the container attach request initiated by kube-apiserver
type ContainerAttachConnection struct {
	MessageID    uint64
	ctx          context.Context
	r            *restful.Request
	Conn         net.Conn
	session      *Session
	edgePeerStop chan struct{}
	closeChan    chan bool
}

func (ah *ContainerAttachConnection) String() string {
	return fmt.Sprintf("APIServer_AttachConnection MessageID %v", ah.MessageID)
}

func (ah *ContainerAttachConnection) WriteToAPIServer(p []byte) (n int, err error) {
	return ah.Conn.Write(p)
}

func (ah *ContainerAttachConnection) SetMessageID(id uint64) {
	ah.MessageID = id
}

func (ah *ContainerAttachConnection) GetMessageID() uint64 {
	return ah.MessageID
}

func (ah *ContainerAttachConnection) SetEdgePeerDone() {
	select {
	case <-ah.closeChan:
		return
	case ah.EdgePeerDone() <- struct{}{}:
		klog.V(6).Infof("success send channel deleting connection with messageID %v", ah.MessageID)
	}
}

func (ah *ContainerAttachConnection) EdgePeerDone() chan struct{} {
	return ah.edgePeerStop
}

func (ah *ContainerAttachConnection) WriteToTunnel(m *stream.Message) error {
	return ah.session.WriteMessageToTunnel(m)
}

func (ah *ContainerAttachConnection) SendConnection() (stream.EdgedConnection, error) {
	// todo: new edged connection and create connect message
	return nil, nil
}

func (ah *ContainerAttachConnection) Serve() error {
	// todo: send to connection, read response date and write to tunnel
	return nil
}

var _ APIServerConnection = &ContainerAttachConnection{}
