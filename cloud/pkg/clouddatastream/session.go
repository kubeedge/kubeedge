package clouddatastream

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/kubeedge/pkg/stream"
	"k8s.io/klog/v2"
)

type Session struct {
	sessionID     string
	tunnel        stream.SafeWriteTunneler
	tunnelClosed  bool
	apiServerConn map[uint64]APIServerConnection
	apiConnlock   *sync.RWMutex
}

func (s *Session) WriteMessageToTunnel(m *stream.Message) error {
	return s.tunnel.WriteMessage(m)
}

func (s *Session) Close() {
	s.apiConnlock.Lock()
	defer s.apiConnlock.Unlock()
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
			return
		}
	}
}

func (s *Session) ProxyTunnelMessageToApiserver(message *stream.Message) error {
	s.apiConnlock.RLock()
	defer s.apiConnlock.RUnlock()
	kubeCon, ok := s.apiServerConn[message.ConnectID]
	if !ok {
		return fmt.Errorf("can not find apiServer connection id %v in %v",
			message.ConnectID, s.String())
	}
	switch message.MessageType {
	case stream.MessageTypeRemoveConnect:
		klog.V(6).Infof("delete connection %v from %v", message.ConnectID, s.String())
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
		return nil, fmt.Errorf("the tunnel connection of %v has closed", s.String())
	}
	connection.SetMessageID(id)
	s.apiServerConn[id] = connection
	klog.Infof("Add a new apiserver connection %s in to %s", connection.String(), s.String())
	return connection, nil
}

func (s *Session) GetAPIServerConnection(id uint64) (APIServerConnection, error) {
	s.apiConnlock.RLock()
	defer s.apiConnlock.RUnlock()
	kubeCon, ok := s.apiServerConn[id]
	if !ok {
		return nil, fmt.Errorf("can not find apiServer connection id %v in %v",
			id, s.String())
	}
	return kubeCon, nil
}

func (s *Session) DeleteAPIServerConnection(con APIServerConnection) {
	s.apiConnlock.Lock()
	delete(s.apiServerConn, con.GetMessageID())
	// Other operations do not affect the release of the lock,
	// but will increase the lock holding time,
	// so the lock needs to be released in advance.
	s.apiConnlock.Unlock()
	klog.Infof("Delete a apiserver connection %s from %s", con.String(), s.String())
}

func (s *Session) IsTunnelClosed() bool {
	return s.tunnelClosed
}
