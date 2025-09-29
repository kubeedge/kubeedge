package util

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/streamruleendpoint/dao"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

type TunnelSession struct {
	Tunnel        stream.SafeWriteTunneler
	closeLock     sync.Mutex
	closed        bool
	localCons     map[uint64]stream.EdgedConnection
	localConsLock sync.RWMutex
}

func NewTunnelSession(c *websocket.Conn) *TunnelSession {
	return &TunnelSession{
		closeLock:     sync.Mutex{},
		localConsLock: sync.RWMutex{},
		Tunnel:        stream.NewDefaultTunnel(c),
		localCons:     make(map[uint64]stream.EdgedConnection, 128),
	}
}

func (s *TunnelSession) startPing(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := s.Tunnel.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				klog.Errorf("Write Ping Message error %v", err)
				return
			}
		}
	}
}

func (s *TunnelSession) Serve() {
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		s.Close()
		klog.Info("Close tunnel session successfully")
	}()

	go s.startPing(ctx)

	for {
		_, r, err := s.Tunnel.NextReader()
		if err != nil {
			klog.Errorf("Read Message error %v", err)
			return
		}

		mess, err := stream.ReadMessageFromTunnel(r)
		if err != nil {
			klog.Errorf("Get tunnel Message error %v", err)
			return
		}

		if mess.MessageType == stream.MessageTypeCloseConnect {
			klog.Errorf("close tunnel stream connection, error:%s", string(mess.Data))
			return
		}

		if (mess.MessageType < stream.MessageTypeData) || (mess.MessageType >= stream.MessageTypeAttachConnect) {
			go s.ServeConnection(mess)
		}
		s.WriteToLocalConnection(mess)
	}
}

func (s *TunnelSession) serveVideoConnection(m *stream.Message) error {
	videoCon := &stream.EdgedVideoConnection{
		ReadChan: make(chan *stream.Message, 128),
		Stop:     make(chan struct{}, 2),
	}

	if err := json.Unmarshal(m.Data, videoCon); err != nil {
		klog.Errorf("unmarshal connector data error %v", err)
		return err
	}
	last := path.Base(videoCon.URL.Path)
	EpUrl, err := dao.GetEpUrlsByKey(last)
	if err != nil {
		klog.Errorf("Get url by endpoint %s error %v", last, err)
		return err
	}
	videoCon.ResourceUrl = EpUrl.URL
	s.AddLocalConnection(m.ConnectID, videoCon)
	return videoCon.Serve(s.Tunnel)
}

func (s *TunnelSession) ServeConnection(m *stream.Message) {
	switch m.MessageType {
	case stream.MessageTypeVideoConnect:
		if err := s.serveVideoConnection(m); err != nil {
			klog.Errorf("Serve Video connection error %s", m.String())
		}

	default:
		panic(fmt.Sprintf("Wrong message type %v", m.MessageType))
	}
	s.DeleteLocalConnection(m.ConnectID)
	klog.V(6).Infof("Delete local connection MessageID %v Type %s", m.ConnectID, m.MessageType.String())
}

func (s *TunnelSession) WriteToLocalConnection(m *stream.Message) {
	if con, ok := s.GetLocalConnection(m.ConnectID); ok {
		con.CacheTunnelMessage(m)
	}
}

func (s *TunnelSession) Close() {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	if !s.closed {
		s.Tunnel.Close()
	}
	s.closed = true
}

func (s *TunnelSession) AddLocalConnection(id uint64, con stream.EdgedConnection) {
	s.localConsLock.Lock()
	defer s.localConsLock.Unlock()
	s.localCons[id] = con
}

func (s *TunnelSession) GetLocalConnection(id uint64) (stream.EdgedConnection, bool) {
	s.localConsLock.RLock()
	defer s.localConsLock.RUnlock()
	con, ok := s.localCons[id]
	return con, ok
}

func (s *TunnelSession) DeleteLocalConnection(id uint64) {
	s.localConsLock.Lock()
	defer s.localConsLock.Unlock()
	con, ok := s.localCons[id]
	if !ok {
		return
	}
	con.CleanChannel()
	con.CloseReadChannel()
	delete(s.localCons, id)
}
