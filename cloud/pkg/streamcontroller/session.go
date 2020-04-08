package streamcontroller

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"k8s.io/klog"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

// Session 代表了tunnel链接对应的多个来着apiserver的连接
type Session struct {
	sync.Mutex
	nextID    uint64 // 唯一代表的此session 的id，用来区分message, 从0 开始，来一个apiserver 的连接，则+1
	sessionID string // 代表这个session的唯一id，

	// server 和 特定agent 的tunnel通道
	tunnelCon *websocket.Conn
	closed    bool // websocket tunnelCon 是否关闭

	// 来自于apiserver 的连接 map,来一个apiserver 的连接，nextID 就+1,并使用+1的ID作为Key
	//apiServerConn map[uint64]*ApiServerConnection
	apiServerConn map[uint64]ApiServerConnection
}

func (s *Session) WriteMessageToTunnel(m *stream.Message) error {
	return m.WriteTo(s.tunnelCon)
}

func (s *Session) Close() {
	s.Lock()
	s.tunnelCon.Close()
	for _, c := range s.apiServerConn {
		c.Close()
	}
	s.closed = true
	s.Unlock()
}

// Serve read tunnelCon message ,and write to specific apiserver connection
func (s *Session) Serve() {
	defer s.Close()

	for {
		t, r, err := s.tunnelCon.NextReader()
		if err != nil {
			klog.Errorf("get %v reader error %v", s, err)
			return
		}
		if t != websocket.TextMessage {
			klog.Errorf("wrong websocket message type")
			return
		}
		if err := s.ProxyTunnelToApiserver(r); err != nil {
			klog.Errorf("get tunnelCon message error %v", err)
			continue
		}
	}
}

func (s *Session) ProxyTunnelToApiserver(r io.Reader) error {
	message, err := stream.TunnelMessage(r)
	if err != nil {
		return err
	}
	// no need lock
	con, ok := s.apiServerConn[message.ConnectID]
	if !ok {
		return fmt.Errorf("can not find this connection %v of %v",
			message.ConnectID, s)
	}
	for i := 0; i < len(message.Data); {
		n, err := con.Write(message.Data[i:])
		if err != nil {
			return err
		}
		i += n
	}
	return nil
}

func (s *Session) String() string {
	return fmt.Sprintf("session key %v", s.sessionID)
}

func (s *Session) AddAPIServerConnection(connection ApiServerConnection) (ApiServerConnection, error) {
	id := atomic.AddUint64(&(s.nextID), 1)
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return nil, fmt.Errorf("%v tunnelCon closed", s)
	}
	connection.SetID(id)
	s.apiServerConn[id] = connection
	return connection, nil
}
