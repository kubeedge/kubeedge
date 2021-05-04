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

package edgestream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// TunnelSession
type TunnelSession struct {
	Tunnel        stream.SafeWriteTunneler
	closeLock     sync.Mutex
	closed        bool // tunnel whether closed
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

func (s *TunnelSession) serveLogsConnection(m *stream.Message) error {
	logCon := &stream.EdgedLogsConnection{
		ReadChan: make(chan *stream.Message, 128),
	}
	if err := json.Unmarshal(m.Data, logCon); err != nil {
		klog.Errorf("unmarshal connector data error %v", err)
		return err
	}

	s.AddLocalConnection(m.ConnectID, logCon)
	return logCon.Serve(s.Tunnel)
}

func (s *TunnelSession) serveContainerExecConnection(m *stream.Message) error {
	execCon := &stream.EdgedExecConnection{
		ReadChan: make(chan *stream.Message, 128),
	}
	if err := json.Unmarshal(m.Data, execCon); err != nil {
		klog.Errorf("unmarshal connector data error %v", err)
		return err
	}

	s.AddLocalConnection(m.ConnectID, execCon)
	klog.V(6).Infof("Get Exec Connection info: %++v", *execCon)
	return execCon.Serve(s.Tunnel)
}

func (s *TunnelSession) serveMetricsConnection(m *stream.Message) error {
	metricsCon := &stream.EdgedMetricsConnection{
		ReadChan: make(chan *stream.Message, 128),
	}
	if err := json.Unmarshal(m.Data, metricsCon); err != nil {
		klog.Errorf("unmarshal connector data error %v", err)
		return err
	}

	s.AddLocalConnection(m.ConnectID, metricsCon)
	return metricsCon.Serve(s.Tunnel)
}

func (s *TunnelSession) ServeConnection(m *stream.Message) {
	switch m.MessageType {
	case stream.MessageTypeLogsConnect:
		if err := s.serveLogsConnection(m); err != nil {
			klog.Errorf("Serve Logs connection error %s", m.String())
		}
	case stream.MessageTypeExecConnect:
		if err := s.serveContainerExecConnection(m); err != nil {
			klog.Errorf("Serve Container Exec connection error %s", m.String())
		}
	case stream.MessageTypeMetricConnect:
		if err := s.serveMetricsConnection(m); err != nil {
			klog.Errorf("Serve Metrics connection error %s", m.String())
		}
	default:
		panic(fmt.Sprintf("Wrong message type %v", m.MessageType))
	}

	s.DeleteLocalConnection(m.ConnectID)
	klog.V(6).Infof("Delete local connection MessageID %v Type %s", m.ConnectID, m.MessageType.String())
}

func (s *TunnelSession) Close() {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	if !s.closed {
		s.Tunnel.Close()
	}
	s.closed = true
}

func (s *TunnelSession) WriteToLocalConnection(m *stream.Message) {
	if con, ok := s.GetLocalConnection(m.ConnectID); ok {
		con.CacheTunnelMessage(m)
	}
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
		s.Close()
		klog.Info("Close tunnel session successfully")
	}()

	go s.startPing(ctx)

	for {
		_, r, err := s.Tunnel.NextReader()
		if err != nil {
			klog.Errorf("Read Message error %v", err)
			return err
		}

		mess, err := stream.ReadMessageFromTunnel(r)
		if err != nil {
			klog.Errorf("Get tunnel Message error %v", err)
			return err
		}

		if mess.MessageType < stream.MessageTypeData {
			go s.ServeConnection(mess)
		}
		s.WriteToLocalConnection(mess)
	}
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
	delete(s.localCons, id)
}
