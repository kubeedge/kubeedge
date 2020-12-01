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
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// Session indicates one tunnel connection (default websocket) from edgecore
// And multiple kube-apiserver initiated requests to this edgecore
type Session struct {
	// sessionID indicates the unique id of session
	sessionID string

	// tunnel indicates  a tunnel connection between edgecore and cloudcore
	// default is websocket
	tunnel stream.SafeWriteTunneler
	// tunnelClosed indicates whether tunnel closed
	tunnelClosed bool

	// apiServerConn indicates a connection request made by multiple apiserver to one edgecore
	apiServerConn map[uint64]APIServerConnection
	apiConnlock   *sync.Mutex
}

func (s *Session) WriteMessageToTunnel(m *stream.Message) error {
	return m.WriteTo(s.tunnel)
}

func (s *Session) Close() {
	for _, c := range s.apiServerConn {
		c.SetEdgePeerDone()
	}
	s.tunnel.Close()
	s.tunnelClosed = true
}

// Serve read tunnel message ,and write to specific apiserver connection
func (s *Session) Serve() {
	defer s.Close()

	for {
		t, r, err := s.tunnel.NextReader()
		if err != nil {
			klog.Errorf("get %v reader error %v", s.String(), err)
			return
		}
		if t != websocket.TextMessage {
			klog.Errorf("Websocket message type must be %v type", websocket.TextMessage)
			return
		}
		message, err := stream.ReadMessageFromTunnel(r)
		if err != nil {
			klog.Errorf("Read message from tunnel %v error %v", s.String(), err)
			return
		}

		if err := s.ProxyTunnelMessageToApiserver(message); err != nil {
			klog.Errorf("Proxy tunnel message [%s] to kube-apiserver error %v", message.String(), err)
			continue
		}
	}
}

func (s *Session) ProxyTunnelMessageToApiserver(message *stream.Message) error {
	kubeCon, ok := s.apiServerConn[message.ConnectID]
	if !ok {
		return fmt.Errorf("Can not find apiServer connection id %v in %v",
			message.ConnectID, s.String())
	}
	switch message.MessageType {
	case stream.MessageTypeRemoveConnect:
		kubeCon.SetEdgePeerDone()
	case stream.MessageTypeData:
		for i := 0; i < len(message.Data); {
			n, err := kubeCon.WriteToAPIServer(message.Data[i:])
			if err != nil {
				return err
			}
			i += n
		}
	default:
	}
	return nil
}

func (s *Session) String() string {
	return fmt.Sprintf("Tunnel session [%v]", s.sessionID)
}

func (s *Session) AddAPIServerConnection(ss *StreamServer, connection APIServerConnection) (APIServerConnection, error) {
	id := atomic.AddUint64(&(ss.nextMessageID), 1)
	s.apiConnlock.Lock()
	defer s.apiConnlock.Unlock()
	if s.tunnelClosed {
		return nil, fmt.Errorf("The tunnel connection of %v has closed", s.String())
	}
	connection.SetMessageID(id)
	s.apiServerConn[id] = connection
	klog.Infof("Add a new apiserver connection %s in to %s", connection.String(), s.String())
	return connection, nil
}

func (s *Session) DeleteAPIServerConnection(con APIServerConnection) {
	s.apiConnlock.Lock()
	defer s.apiConnlock.Unlock()
	delete(s.apiServerConn, con.GetMessageID())
	klog.Infof("Delete a apiserver connection %s from %s", con.String(), s.String())
}
