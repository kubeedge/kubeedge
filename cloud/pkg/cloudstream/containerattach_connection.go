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
	"time"

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

// SendConnectionWithRetry attempts to send connection with retry mechanism
func (ah *ContainerAttachConnection) SendConnectionWithRetry(maxRetries int, retryDelay time.Duration) (stream.EdgedConnection, error) {
	var connector stream.EdgedConnection
	var err error

	for i := 0; i < maxRetries; i++ {
		connector, err = ah.SendConnection()
		if err == nil {
			return connector, nil
		}

		klog.Warningf("%s failed to send connection (attempt %d/%d): %v", ah.String(), i+1, maxRetries, err)
		
		if i < maxRetries-1 {
			select {
			case <-ah.ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry")
			case <-ah.EdgePeerDone():
				return nil, fmt.Errorf("edge peer done during retry")
			case <-time.After(retryDelay):
				// Continue to next retry
			}
		}
	}

	return nil, fmt.Errorf("failed to send connection after %d attempts: %v", maxRetries, err)
}

func (ah *ContainerAttachConnection) Serve() error {
	defer func() {
		close(ah.closeChan)
		klog.V(6).Infof("%s stop successfully", ah.String())
	}()

	// first send connect message with retry mechanism
	connector, err := ah.SendConnectionWithRetry(3, time.Second*2)
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
			
			// Wait before retry
			select {
			case <-ah.ctx.Done():
				return
			case <-ah.EdgePeerDone():
				return
			case <-time.After(time.Second * time.Duration(retry+1)):
				// Continue to next retry
			}
		}
		klog.Errorf("max retry count reached when send %s message to edge", msg.MessageType)
	}

	var data [256]byte
	readTimeout := time.Second * 30
	
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
		
		// Set read timeout to detect stale connections
		if err := ah.Conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			klog.Warningf("%s failed to set read deadline: %v", ah.String(), err)
		}
		
		n, err := ah.Conn.Read(data[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				klog.V(6).Infof("%s read EOF from client", ah.String())
				sendCloseMessage()
				return nil
			}
			
			// Check for timeout error
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				klog.V(6).Infof("%s read timeout, continuing", ah.String())
				continue
			}
			
			klog.Errorf("%s failed to read from client: %v", ah.String(), err)
			sendCloseMessage()
			return fmt.Errorf("read from client failed: %v", err)
		}
		
		if n <= 0 {
			continue
		}
		
		msg := stream.NewMessage(connector.GetMessageID(), stream.MessageTypeData, data[:n])
		if err := ah.WriteToTunnel(msg); err != nil {
			klog.Errorf("%s failed to write to tunnel server, err: %v", ah.String(), err)
			sendCloseMessage()
			return fmt.Errorf("write to tunnel failed: %v", err)
		}
	}
}

// IsConnectionActive checks if the connection is still active
func (ah *ContainerAttachConnection) IsConnectionActive() bool {
	select {
	case <-ah.closeChan:
		return false
	case <-ah.edgePeerStop:
		return false
	default:
		return true
	}
}

// GetConnectionStats returns basic connection statistics
func (ah *ContainerAttachConnection) GetConnectionStats() map[string]interface{} {
	return map[string]interface{}{
		"message_id":   ah.MessageID,
		"active":       ah.IsConnectionActive(),
		"session_type": "container_attach",
	}
}

var _ APIServerConnection = &ContainerAttachConnection{}