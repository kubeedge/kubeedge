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
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/emicklei/go-restful"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
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
	connector := &stream.EdgedAttachConnection{
		MessID: ah.MessageID,
		Method: ah.r.Request.Method,
		URL:    *ah.r.Request.URL,
		Header: ah.r.Request.Header,
	}
	connector.URL.Scheme = httpScheme
	connector.URL.Host = net.JoinHostPort(defaultServerHost, fmt.Sprintf("%v", constants.ServerPort))
	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}
	if err := ah.WriteToTunnel(m); err != nil {
		klog.Errorf("%s failed to create attach connection: %s, err: %v", ah.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (ah *ContainerAttachConnection) Serve() error {
	defer func() {
		close(ah.closeChan)
		klog.V(6).Infof("%s stop successfully", ah.String())
	}()

	// first send connect message
	connector, err := ah.SendConnection()
	if err != nil {
		klog.Errorf("%s send %s info error %v", ah.String(), stream.MessageTypeAttachConnect, err)
		return err
	}

	sendCloseMessage := func() {
		msg := stream.NewMessage(ah.MessageID, stream.MessageTypeRemoveConnect, nil)
		for retry := 0; retry < 3; retry++ {
			if err := ah.WriteToTunnel(msg); err == nil {
				klog.V(6).Infof("%s send close message to edge successfully", ah.String())
				return
			}
			klog.Warningf("%v failed send %s message to edge, err: %v", ah, msg.MessageType, err)
		}
		klog.Errorf("max retry count reached when send %s message to edge", msg.MessageType)
	}

	var data [256]byte
	for {
		select {
		case <-ah.ctx.Done():
			// if apiserver request end, send close message to edge
			sendCloseMessage()
			return nil
		case <-ah.EdgePeerDone():
			klog.V(6).Infof("%s find edge peer done, so stop this connection", ah.String())
			return fmt.Errorf("%s find edge peer done, so stop this connection", ah.String())
		default:
		}
		func() {
			n, err := ah.Conn.Read(data[:])
			if err != nil {
				if !errors.Is(err, io.EOF) {
					klog.Errorf("%s failed to read from client: %v", ah.String(), err)
					return
				}
				klog.V(6).Infof("%s read EOF from client", ah.String())
				sendCloseMessage()
				return
			}
			if n <= 0 {
				return
			}
			msg := stream.NewMessage(connector.GetMessageID(), stream.MessageTypeData, data[:n])
			if err := ah.WriteToTunnel(msg); err != nil {
				klog.Errorf("%s failed to write to tunnel server, err: %v", ah.String(), err)
				return
			}
		}()
	}
}

var _ APIServerConnection = &ContainerAttachConnection{}
