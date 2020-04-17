package edgestream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

type TunnelSession struct {
	sync.Mutex
	Tunnel    *websocket.Conn
	closed    bool // tunnel whether closed
	LocalCons map[uint64]*LocalConnection
}

type LocalConnection struct {
	sync.Mutex
	ID     uint64
	con    *websocket.Conn
	closed bool // inditicate con whether closed
	tunnel *TunnelSession
}

func (l *LocalConnection) Server() error {
	defer l.Close()
	// read data from localconnection and then write to tunnel
	for {
		l.con.SetReadDeadline(time.Now().Add(time.Second * 5))
		_, data, err := l.con.ReadMessage()
		if err != nil {
			klog.Errorf("read local connection message error %v", err)
			return err
		}
		msg := stream.NewMessage(l.ID, stream.MessageTypeData, data)
		if err := msg.WriteTo(l.tunnel.Tunnel); err != nil {
			klog.Errorf("read localconnnection message to tunnel error %v", err)
			return err
		}
	}
}
func (l *LocalConnection) Close() {
	l.Lock()
	defer l.Unlock()
	if !l.closed {
		l.con.Close()
	}
	l.closed = true
}

func NewTunnelSession(c *websocket.Conn) *TunnelSession {
	return &TunnelSession{
		Tunnel:    c,
		LocalCons: make(map[uint64]*LocalConnection, 10),
	}
}

func (s *TunnelSession) serveLogsConnection(m *stream.Message) error {

	logCon := &stream.EdgeLogsConnector{}
	if err := json.Unmarshal(m.Data, logCon); err != nil {
		klog.Errorf("unmarshal connector data error %v", err)
		return err
	}

	return logCon.Serve(s.Tunnel)
	/*
		l := &LocalConnection{
			ID:     m.ConnectID,
			con:    localCon,
			tunnel: s,
		}
		s.LocalCons[l.ID] = l
		go l.Server()
		return nil
	*/
}

func (s *TunnelSession) ServeConnection(m *stream.Message) error {
	switch m.MessageType {
	case stream.MessageTypeLogsConnect:
		return s.serveLogsConnection(m)
	case stream.MessageTypeExecConnect:
		panic("TODO")
	default:
		panic(fmt.Errorf("Wrong message type %v", m.MessageType))
	}

	return nil
}

func (s *TunnelSession) Close() {
	s.Lock()
	defer s.Unlock()
	if !s.closed {
		s.Tunnel.Close()
	}
	for _, c := range s.LocalCons {
		c.Close()
	}
	s.closed = true
}

func (s *TunnelSession) WriteToLocal(m *stream.Message) error {
	if m.MessageType != stream.MessageTypeData {
		return nil
	}

	local, ok := s.LocalCons[m.ConnectID]
	if !ok {
		return fmt.Errorf("can not find this tunnel")
	}
	return local.con.WriteMessage(websocket.TextMessage, m.Data)
}

func (s *TunnelSession) startPing(ctx context.Context) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := s.Tunnel.WriteControl(websocket.PingMessage, []byte("ping you"), time.Now().Add(time.Second))
			if err != nil {
				klog.Errorf("write ping message error %v", err)
				return
			}
		}
	}

}

func (s *TunnelSession) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		klog.Info("prepare to close tunnel server ....")
		s.Close()
	}()

	go s.startPing(ctx)

	for {
		_, r, err := s.Tunnel.NextReader()
		if err != nil {
			klog.Errorf("Read Message error %v", err)
			return err
		}
		mess, err := stream.TunnelMessage(r)
		if err != nil {
			klog.Errorf("get tunnel Message error %v", err)
			return err
		}

		if mess.MessageType < stream.MessageTypeData {
			err := s.ServeConnection(mess)
			if err != nil {
				klog.Errorf("server local connection error %v", err)
				return err
			}
		}

		if mess.MessageType == stream.MessageTypeRemoveConnect {
			klog.Infof("receive remove client id %v", mess.ConnectID)
			if localCon, ok := s.LocalCons[mess.ConnectID]; ok {
				localCon.Close()
			}
			continue
		}
		// write data from tunnel to local connector
		if err := s.WriteToLocal(mess); err != nil {
			klog.Errorf("write to local connection error %v", err)
			return err
		}
	}
}
