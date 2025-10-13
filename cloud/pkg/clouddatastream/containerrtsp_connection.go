package clouddatastream

import (
	"context"
	"fmt"
	"sync"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/kubeedge/kubeedge/pkg/stream"
	"k8s.io/klog/v2"
)

type ContainerRTSPConnection struct {
	// MessageID indicate the unique id to create his message
	MessageID    uint64
	ctx          context.Context
	req          *restful.Request
	wsConns      []*websocket.Conn
	session      *Session
	edgePeerStop chan struct{}
	emptyChan    chan struct{}
	closeChan    chan bool
	mu           sync.Mutex
}

func (r *ContainerRTSPConnection) GetMessageID() uint64 {
	return r.MessageID
}

func (r *ContainerRTSPConnection) SetEdgePeerDone() {
	select {
	case <-r.closeChan:
		return
	case r.EdgePeerDone() <- struct{}{}:
		klog.V(6).Infof("success send channel deleting connection with messageID %v", r.MessageID)
	}
}

func (r *ContainerRTSPConnection) EdgePeerDone() chan struct{} {
	return r.edgePeerStop
}

func (r *ContainerRTSPConnection) AddWSConn(conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wsConns = append(r.wsConns, conn)
}

func (r *ContainerRTSPConnection) WriteToAPIServer(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Iterate over all WebSocket clients
	for i := 0; i < len(r.wsConns); {
		ws := r.wsConns[i]
		err := ws.WriteMessage(websocket.BinaryMessage, p)
		if err != nil {
			klog.Errorf("write to wsConn error: %v, removing it", err)
			ws.Close()

			// Remove Failed Connections
			r.wsConns = append(r.wsConns[:i], r.wsConns[i+1:]...)
			continue
		}
		i++
	}

	if len(r.wsConns) == 0 {
		select {
		case r.emptyChan <- struct{}{}:
		default:
		}
	}

	return len(p), nil
}

func (r *ContainerRTSPConnection) WriteToTunnel(m *stream.Message) error {
	return r.session.WriteMessageToTunnel(m)
}

func (r *ContainerRTSPConnection) SetMessageID(id uint64) {
	r.MessageID = id
}

func (r *ContainerRTSPConnection) String() string {
	return fmt.Sprintf("APIServer_RTSPConnection MessageID %v", r.MessageID)
}

func (r *ContainerRTSPConnection) SendConnection() (stream.EdgedConnection, error) {
	connector := &stream.EdgedVideoConnection{
		MessID: r.MessageID,
		URL:    *r.req.Request.URL,
		Header: r.req.Request.Header,
	}
	m, err := connector.CreateConnectMessage()
	if err != nil {
		return nil, err
	}

	if err := r.WriteToTunnel(m); err != nil {
		klog.Errorf("%s write %s error %v", r.String(), connector.String(), err)
		return nil, err
	}
	return connector, nil
}

func (r *ContainerRTSPConnection) Serve() error {
	defer func() {
		close(r.closeChan)
		klog.Infof("%s end successful", r.String())
	}()

	if _, err := r.SendConnection(); err != nil {
		klog.Errorf("%s send %s info error %v", r.String(), stream.MessageTypeVideoConnect, err)
		return err
	}

	for {
		select {
		case <-r.ctx.Done():
			// if apiserver request end, send close message to edge
			msg := stream.NewMessage(r.MessageID, stream.MessageTypeRemoveConnect, nil)
			for retry := 0; retry < 3; retry++ {
				if err := r.WriteToTunnel(msg); err != nil {
					klog.Warningf("%v send %s message to edge error %v", r, msg.MessageType, err)
				} else {
					break
				}
			}
			klog.Infof("%s send close message to edge successfully", r.String())
			return nil
		case <-r.EdgePeerDone():
			klog.V(6).Infof("%s find edge peer done, so stop this connection", r.String())
			return fmt.Errorf("%s find edge peer done, so stop this connection", r.String())
		}
	}
}

var _ APIServerConnection = &ContainerRTSPConnection{}
