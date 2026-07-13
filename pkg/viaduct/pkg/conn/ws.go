package conn

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/fifo"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/keeper"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/lane"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/mux"
)

type WSConnection struct {
	WriteDeadline        time.Time
	ReadDeadline         time.Time
	handler              mux.Handler
	wsConn               *websocket.Conn
	state                *ConnectionState
	syncKeeper           *keeper.SyncKeeper
	connUse              api.UseType
	consumer             io.Writer
	autoRoute            bool
	messageFifo          *fifo.MessageFifo
	locker               sync.Mutex
	OnReadTransportErr   func(nodeID, projectID string)
	readDeadlineInterval time.Duration
}

func NewWSConn(options *ConnectionOptions) *WSConnection {
	return &WSConnection{
		wsConn:               options.Base.(*websocket.Conn),
		handler:              options.Handler,
		syncKeeper:           keeper.NewSyncKeeper(),
		state:                options.State,
		connUse:              options.ConnUse,
		autoRoute:            options.AutoRoute,
		messageFifo:          fifo.NewMessageFifo(),
		OnReadTransportErr:   options.OnReadTransportErr,
		readDeadlineInterval: options.ReadDeadlineInterval,
	}
}

// ServeConn start to receive message from connection
func (conn *WSConnection) ServeConn() {
	switch conn.connUse {
	case api.UseTypeMessage:
		go conn.handleMessage()
	case api.UseTypeStream:
		go conn.handleRawData()
	case api.UseTypeShare:
		klog.Error("don't support share in websocket")
	}
}

func (conn *WSConnection) filterControlMessage(msg *model.Message) bool {
	// check control message
	operation := msg.GetOperation()
	if operation != comm.ControlTypeConfig &&
		operation != comm.ControlTypePing &&
		operation != comm.ControlTypePong {
		return false
	}

	// feedback the response
	resp := msg.NewRespByMessage(msg, comm.RespTypeAck)
	conn.locker.Lock()
	err := lane.NewLane(api.ProtocolTypeWS, conn.wsConn).WriteMessage(resp)
	conn.locker.Unlock()
	if err != nil {
		klog.Errorf("failed to send response back, error:%+v", err)
	}
	return true
}

func (conn *WSConnection) handleRawData() {
	if conn.consumer == nil {
		klog.Warning("bad consumer for raw data")
		return
	}

	if !conn.autoRoute {
		return
	}

	// TODO: support control message processing in raw data mode
	_, err := io.Copy(conn.consumer, lane.NewLane(api.ProtocolTypeQuic, conn.wsConn))
	if err != nil {
		klog.Errorf("failed to copy data, error: %+v", err)
		conn.state.State = api.StatDisconnected
		conn.wsConn.Close()
		return
	}
}

// pingLoop keeps guaranteed inbound traffic flowing on an otherwise idle
// connection by sending WebSocket protocol pings every half of
// readDeadlineInterval. The pongs sent back by the peer (gorilla answers
// pings automatically as long as its read loop is running) reset the read
// deadline via the pong handler registered in handleMessage, so the deadline
// only expires when the connection is genuinely stalled.
func (conn *WSConnection) pingLoop(stop <-chan struct{}) {
	period := conn.readDeadlineInterval / 2
	if period <= 0 {
		// In-repo configuration is int32 seconds (>= 1s), but the field
		// takes a raw Duration: guard the integer division against sub-2ns
		// values so NewTicker cannot panic.
		period = time.Millisecond
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			// WriteControl is safe for concurrent use with the data-plane
			// writes going through conn.locker.
			if err := conn.wsConn.WriteControl(websocket.PingMessage, nil, time.Now().Add(period)); err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					// The write lock was held past the deadline (e.g. a
					// large message on a slow link); the connection may
					// well be healthy. Keep pinging — liveness is judged
					// by the read deadline, not here.
					continue
				}
				// If handleMessage already tore the connection down (stop is
				// closed), this error is a consequence of that shutdown, not a
				// new failure: exit quietly without a misleading log or a
				// double close.
				select {
				case <-stop:
					return
				default:
				}
				// The connection is gone (closed or broken); close it so the
				// read loop unblocks immediately instead of waiting for the
				// read deadline to expire.
				klog.V(2).Infof("ping loop stopped: %v", err)
				_ = conn.wsConn.Close()
				return
			}
		}
	}
}

func (conn *WSConnection) handleMessage() {
	if conn.readDeadlineInterval > 0 {
		// A read deadline alone would tear down idle-but-healthy
		// connections: nothing guarantees cloud-to-edge traffic within the
		// interval (EdgeHub's keepalive is one-way; CloudHub sends no
		// response to it). pingLoop provides that guarantee at the
		// WebSocket protocol level, and every pong extends the deadline
		// here, so expiry now means the peer stopped answering entirely.
		conn.wsConn.SetPongHandler(func(string) error {
			return conn.wsConn.SetReadDeadline(time.Now().Add(conn.readDeadlineInterval))
		})
		stopPing := make(chan struct{})
		defer close(stopPing)
		go conn.pingLoop(stopPing)
	}
	for {
		// Arm/refresh the read deadline so a half-open TCP connection
		// surfaces as a read error within readDeadlineInterval instead of
		// waiting for kernel TCP retransmission (tcp_retries2, ~15min on Linux).
		if conn.readDeadlineInterval > 0 {
			_ = conn.wsConn.SetReadDeadline(time.Now().Add(conn.readDeadlineInterval))
		}
		msg := &model.Message{}
		err := lane.NewLane(api.ProtocolTypeWS, conn.wsConn).ReadMessage(msg)
		if err != nil {
			// With pingLoop keeping pongs flowing, a deadline expiry means
			// no inbound traffic at all for readDeadlineInterval: the
			// connection is treated as half-open. This is the designed
			// detection path, so keep it quiet; EdgeHub already logs the
			// resulting reconnect at Warning level. Genuine transport
			// errors still surface as Errorf.
			var netErr net.Error
			switch {
			case errors.Is(err, io.EOF):
				// silent: legacy behavior
			case errors.As(err, &netErr) && netErr.Timeout():
				klog.V(2).Infof("read deadline reached, will reconnect: %v", err)
			default:
				klog.Errorf("failed to read message, error: %+v", err)
			}
			conn.state.State = api.StatDisconnected
			_ = conn.wsConn.Close()
			// Close the FIFO so that callers blocked on ReadMessage()/Get()
			// (e.g. EdgeHub's routeToEdge) observe the error immediately and
			// can trigger a reconnect, instead of waiting for the next
			// keepalive write to fail (up to one Heartbeat period).
			conn.messageFifo.Close()

			if conn.OnReadTransportErr != nil {
				conn.OnReadTransportErr(conn.state.Headers.Get("node_id"),
					conn.state.Headers.Get("project_id"))
			}

			return
		}

		// filter control message
		if filtered := conn.filterControlMessage(msg); filtered {
			continue
		}

		// to check whether the message is a response or not
		if matched := conn.syncKeeper.MatchAndNotify(*msg); matched {
			continue
		}

		// put the messages into fifo and wait for reading
		if !conn.autoRoute {
			conn.messageFifo.Put(msg)
			continue
		}

		if conn.handler == nil {
			// use default mux
			conn.handler = mux.MuxDefault
		}
		conn.handler.ServeConn(&mux.MessageRequest{
			Header:           conn.state.Headers,
			PeerCertificates: conn.state.PeerCertificates,
			Message:          msg,
		}, &responseWriter{
			Type: api.ProtocolTypeWS,
			Van:  conn.wsConn,
		})
	}
}

func (conn *WSConnection) SetReadDeadline(t time.Time) error {
	conn.ReadDeadline = t
	return conn.wsConn.SetReadDeadline(t)
}

func (conn *WSConnection) SetWriteDeadline(t time.Time) error {
	conn.WriteDeadline = t
	return conn.wsConn.SetWriteDeadline(t)
}

func (conn *WSConnection) Read(raw []byte) (int, error) {
	return lane.NewLane(api.ProtocolTypeWS, conn.wsConn).Read(raw)
}

func (conn *WSConnection) Write(raw []byte) (int, error) {
	return lane.NewLane(api.ProtocolTypeWS, conn.wsConn).Write(raw)
}

func (conn *WSConnection) WriteMessageAsync(msg *model.Message) error {
	lane := lane.NewLane(api.ProtocolTypeWS, conn.wsConn)
	_ = lane.SetWriteDeadline(conn.WriteDeadline)
	msg.Header.Sync = false
	conn.locker.Lock()
	defer conn.locker.Unlock()
	return lane.WriteMessage(msg)
}

func (conn *WSConnection) WriteMessageSync(msg *model.Message) (*model.Message, error) {
	lane := lane.NewLane(api.ProtocolTypeWS, conn.wsConn)
	// send msg
	_ = lane.SetWriteDeadline(conn.WriteDeadline)
	msg.Header.Sync = true
	conn.locker.Lock()
	err := lane.WriteMessage(msg)
	conn.locker.Unlock()
	if err != nil {
		klog.Errorf("write message error(%+v)", err)
		return nil, err
	}
	//receive response
	response, err := conn.syncKeeper.WaitResponse(msg, conn.WriteDeadline)
	return &response, err
}

func (conn *WSConnection) ReadMessage(msg *model.Message) error {
	return conn.messageFifo.Get(msg)
}

func (conn *WSConnection) RemoteAddr() net.Addr {
	return conn.wsConn.RemoteAddr()
}

func (conn *WSConnection) LocalAddr() net.Addr {
	return conn.wsConn.LocalAddr()
}

func (conn *WSConnection) Close() error {
	conn.messageFifo.Close()
	return conn.wsConn.Close()
}

// get connection state
// TODO:
func (conn *WSConnection) ConnectionState() ConnectionState {
	return *conn.state
}
