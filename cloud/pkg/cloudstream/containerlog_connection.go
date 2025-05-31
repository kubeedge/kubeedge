/*
Copyright 2020 The KubeEdge Authors.

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
	"io"
	"net"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

// ContainerLogsConnection indicates the containerlogs request initiated by kube-apiserver
type ContainerLogsConnection struct {
	// MessageID indicate the unique id to create his message
	MessageID    uint64
	ctx          context.Context
	r            *restful.Request
	flush        io.Writer
	session      *Session
	edgePeerStop chan struct{}
	closeChan    chan bool
}

func (l *ContainerLogsConnection) String() string {
	return fmt.Sprintf("APIServer_LogsConnection MessageID %v", l.MessageID)
}

func (l *ContainerLogsConnection) WriteToAPIServer(p []byte) (n int, err error) {
	return l.flush.Write(p)
}

func (l *ContainerLogsConnection) SetMessageID(id uint64) {
	l.MessageID = id
}

func (l *ContainerLogsConnection) GetMessageID() uint64 {
	return l.MessageID
}

func (l *ContainerLogsConnection) SetEdgePeerDone() {
	select {
	case <-l.closeChan:
		return
	case l.EdgePeerDone() <- struct{}{}:
		klog.V(6).Infof("success send channel deleting connection with messageID %v", l.MessageID)
	}
}

func (l *ContainerLogsConnection) EdgePeerDone() chan struct{} {
	return l.edgePeerStop
}

func (l *ContainerLogsConnection) WriteToTunnel(m *stream.Message) error {
	return l.session.WriteMessageToTunnel(m)
}

func (l *ContainerLogsConnection) SendConnection() (stream.EdgedConnection, error) {
	connector := &stream.EdgedLogsConnection{
		BaseEdgedConnection: stream.BaseEdgedConnection{
			MessID: l.MessageID,
			URL:    *l.r.Request.URL,
			Header: l.r.Request.Header,
		},
	}
	connector.URL.Scheme = httpScheme
	connector.URL.Host = net.JoinHostPort(defaultServerHost, fmt.Sprintf("%v", constants.ServerPort))
	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}
	if err := l.WriteToTunnel(m); err != nil {
		klog.Errorf("%s write %s error %v", l.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (l *ContainerLogsConnection) Serve() error {
	defer func() {
		close(l.closeChan)
		klog.Infof("%s end successful", l.String())
	}()

	// first send connect message
	if _, err := l.SendConnection(); err != nil {
		klog.Errorf("%s send %s info error %v", l.String(), stream.MessageTypeLogsConnect, err)
		return err
	}

	for {
		select {
		case <-l.ctx.Done():
			// if apiserver request end, send close message to edge
			msg := stream.NewMessage(l.MessageID, stream.MessageTypeRemoveConnect, nil)
			for retry := 0; retry < 3; retry++ {
				if err := l.WriteToTunnel(msg); err != nil {
					klog.Warningf("%v send %s message to edge error %v", l, msg.MessageType, err)
				} else {
					break
				}
			}
			klog.Infof("%s send close message to edge successfully", l.String())
			return nil
		case <-l.EdgePeerDone():
			klog.V(6).Infof("%s find edge peer done, so stop this connection", l.String())
			return fmt.Errorf("%s find edge peer done, so stop this connection", l.String())
		}
	}
}

var _ APIServerConnection = &ContainerLogsConnection{}
