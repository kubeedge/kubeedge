/*
Copyright 2019 The Kubernetes Authors.

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

package agent

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/agent/metrics"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"
	"sigs.k8s.io/apiserver-network-proxy/proto/header"
)

const dialTimeout = 5 * time.Second
const xfrChannelSize = 150

// connContext tracks a connection from agent to node network.
type connContext struct {
	conn      net.Conn
	connID    int64
	cleanFunc func()
	dataCh    chan []byte
	cleanOnce sync.Once
	warnChLim bool
}

func (c *connContext) cleanup() {
	c.cleanOnce.Do(c.cleanFunc)
}

func (c *connContext) send(msg []byte) {
	// TODO (cheftako@): Get perf test working and compare this solution with a lock based solution.
	defer func() {
		// Handles the race condition where we write to a closed channel
		if err := recover(); err != nil {
			klog.InfoS("Recovered from attempt to write to closed channel")
		}
	}()
	if c.warnChLim && len(c.dataCh) >= xfrChannelSize {
		klog.V(2).InfoS("Data channel on agent is full", "connectionID", c.connID)
	}

	c.dataCh <- msg
}

type connectionManager struct {
	mu          sync.RWMutex
	connections map[int64]*connContext
}

func (cm *connectionManager) Add(connID int64, ctx *connContext) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[connID] = ctx
}

func (cm *connectionManager) Get(connID int64) (*connContext, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ctx, ok := cm.connections[connID]
	return ctx, ok
}

func (cm *connectionManager) Delete(connID int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, connID)
}

func (cm *connectionManager) List() []*connContext {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	connContexts := make([]*connContext, 0, len(cm.connections))
	for _, connCtx := range cm.connections {
		connContexts = append(connContexts, connCtx)
	}
	return connContexts
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[int64]*connContext),
	}
}

// Identifiers stores agent identifiers that will be used by the server when
// choosing agents
type Identifiers struct {
	IPv4         []string
	IPv6         []string
	Host         []string
	CIDR         []string
	DefaultRoute bool
}

type IdentifierType string

const (
	IPv4         IdentifierType = "ipv4"
	IPv6         IdentifierType = "ipv6"
	Host         IdentifierType = "host"
	CIDR         IdentifierType = "cidr"
	UID          IdentifierType = "uid"
	DefaultRoute IdentifierType = "default-route"
)

// GenAgentIdentifiers generates an Identifiers based on the input string, the
// input string should be a comma-seprated list with each item in the format
// of <IdentifierType>=<address>
func GenAgentIdentifiers(addrs string) (Identifiers, error) {
	var agentIDs Identifiers
	decoded, err := url.ParseQuery(addrs)
	if err != nil {
		return agentIDs, fmt.Errorf("fail to parse url encoded string: %v", err)
	}
	for idType, ids := range decoded {
		switch IdentifierType(idType) {
		case IPv4:
			agentIDs.IPv4 = append(agentIDs.IPv4, ids...)
		case IPv6:
			agentIDs.IPv6 = append(agentIDs.IPv6, ids...)
		case Host:
			agentIDs.Host = append(agentIDs.Host, ids...)
		case CIDR:
			agentIDs.CIDR = append(agentIDs.CIDR, ids...)
		case DefaultRoute:
			defaultRouteIdentifier, err := strconv.ParseBool(ids[0])
			if err == nil && defaultRouteIdentifier {
				agentIDs.DefaultRoute = true
			}
		default:
			return agentIDs, fmt.Errorf("Unknown address type: %s", idType)
		}
	}
	return agentIDs, nil
}

// Client runs on the node network side. It connects to proxy server and establishes
// a stream connection from which it sends and receives network traffic.
type Client struct {
	nextConnID int64

	connManager *connectionManager

	cs *ClientSet // the clientset that includes this AgentClient.

	stream           agent.AgentService_ConnectClient
	agentID          string
	agentIdentifiers string
	serverID         string // the id of the proxy server this client connects to.

	// connect opts
	address string
	opts    []grpc.DialOption
	conn    *grpc.ClientConn
	stopCh  chan struct{}
	// locks
	sendLock      sync.Mutex
	recvLock      sync.Mutex
	probeInterval time.Duration // interval between probe pings

	// file path contains service account token.
	// token's value is auto-rotated by kubernetes, based on projected volume configuration.
	serviceAccountTokenPath string

	warnOnChannelLimit bool
}

func newAgentClient(address, agentID, agentIdentifiers string, cs *ClientSet, opts ...grpc.DialOption) (*Client, int, error) {
	a := &Client{
		cs:                      cs,
		address:                 address,
		agentID:                 agentID,
		agentIdentifiers:        agentIdentifiers,
		opts:                    opts,
		probeInterval:           cs.probeInterval,
		stopCh:                  make(chan struct{}),
		serviceAccountTokenPath: cs.serviceAccountTokenPath,
		connManager:             newConnectionManager(),
		warnOnChannelLimit:      cs.warnOnChannelLimit,
	}
	serverCount, err := a.Connect()
	if err != nil {
		return nil, 0, err
	}
	return a, serverCount, nil
}

// Connect makes the grpc dial to the proxy server. It returns the serverID
// it connects to.
func (a *Client) Connect() (int, error) {
	conn, err := grpc.Dial(a.address, a.opts...)
	if err != nil {
		return 0, err
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(),
		header.AgentID, a.agentID,
		header.AgentIdentifiers, a.agentIdentifiers)
	if a.serviceAccountTokenPath != "" {
		if ctx, err = a.initializeAuthContext(ctx); err != nil {
			err := conn.Close()
			if err != nil {
				klog.ErrorS(err, "failed to close connection")
			}
			return 0, err
		}
	}
	stream, err := agent.NewAgentServiceClient(conn).Connect(ctx)
	if err != nil {
		conn.Close() /* #nosec G104 */
		return 0, err
	}
	serverID, err := serverID(stream)
	if err != nil {
		conn.Close() /* #nosec G104 */
		return 0, err
	}
	serverCount, err := serverCount(stream)
	if err != nil {
		conn.Close() /* #nosec G104 */
		return 0, err
	}
	a.conn = conn
	a.stream = stream
	a.serverID = serverID
	klog.V(2).InfoS("Connect to", "server", serverID)
	return serverCount, nil
}

// Close closes the underlying connection.
func (a *Client) Close() {
	if a.conn == nil {
		klog.Errorln("Unexpected empty AgentClient.conn")
	}
	err := a.conn.Close()
	if err != nil {
		klog.ErrorS(err, "failed to close underlying connection")
	}
	close(a.stopCh)
}

func (a *Client) Send(pkt *client.Packet) error {
	a.sendLock.Lock()
	defer a.sendLock.Unlock()

	err := a.stream.Send(pkt)
	if err != nil && err != io.EOF {
		metrics.Metrics.ObserveFailure(metrics.DirectionToServer)
		a.cs.RemoveClient(a.serverID)
	}
	return err
}

func (a *Client) Recv() (*client.Packet, error) {
	a.recvLock.Lock()
	defer a.recvLock.Unlock()

	pkt, err := a.stream.Recv()
	if err != nil && err != io.EOF {
		metrics.Metrics.ObserveFailure(metrics.DirectionFromServer)
	}
	return pkt, err
}

func serverCount(stream agent.AgentService_ConnectClient) (int, error) {
	md, err := stream.Header()
	if err != nil {
		return 0, err
	}
	scounts := md.Get(header.ServerCount)
	if len(scounts) != 1 {
		return 0, fmt.Errorf("expected one server count, got %d", len(scounts))
	}
	scount := scounts[0]
	return strconv.Atoi(scount)
}

func serverID(stream agent.AgentService_ConnectClient) (string, error) {
	// TODO: this is a blocking call. Add a timeout?
	md, err := stream.Header()
	if err != nil {
		return "", err
	}
	sids := md.Get(header.ServerID)
	if len(sids) != 1 {
		return "", fmt.Errorf("expected one server ID in the context, got %v", sids)
	}
	return sids[0], nil
}

func (a *Client) initializeAuthContext(ctx context.Context) (context.Context, error) {
	var err error
	var b []byte

	// load current service account's token value
	if b, err = ioutil.ReadFile(a.serviceAccountTokenPath); err != nil {
		klog.ErrorS(err, "Failed to read token", "path", a.serviceAccountTokenPath)
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, header.AuthenticationTokenContextKey, header.AuthenticationTokenContextSchemePrefix+string(b))

	return ctx, nil
}

// Connect connects to proxy server to establish a gRPC stream,
// on which the proxied traffic is multiplexed through the stream
// and piped to the local connection. It register itself as a
// backend from proxy server, so proxy server will route traffic
// to this agent.
//
// The caller needs to call Serve to start serving proxy requests
// coming from proxy server.

// Serve starts to serve proxied requests from proxy server over the
// gRPC stream. Successful Connect is required before Serve. The
// The requests include things like opening a connection to a server,
// streaming data and close the connection.
func (a *Client) Serve() {
	defer a.cs.RemoveClient(a.serverID)
	defer func() {
		// close all of conns with remote when Client exits
		for _, connCtx := range a.connManager.List() {
			connCtx.cleanup()
		}
		klog.V(2).InfoS("cleanup all of conn contexts when client exits", "agentID", a.agentID)
	}()

	klog.V(2).InfoS("Start serving", "serverID", a.serverID)
	go a.probe()
	for {
		select {
		case <-a.stopCh:
			klog.V(2).Infoln("stop agent client.")
			return
		default:
		}

		pkt, err := a.Recv()
		if err != nil {
			if err == io.EOF {
				klog.V(2).Infoln("received EOF, exit")
				return
			}
			klog.ErrorS(err, "could not read stream")
			return
		}

		klog.V(5).InfoS("[tracing] recv packet", "type", pkt.Type)

		if pkt == nil {
			klog.V(3).Infoln("empty packet received")
			continue
		}

		switch pkt.Type {
		case client.PacketType_DIAL_REQ:
			klog.V(4).Infoln("received DIAL_REQ")
			resp := &client.Packet{
				Type:    client.PacketType_DIAL_RSP,
				Payload: &client.Packet_DialResponse{DialResponse: &client.DialResponse{}},
			}

			dialReq := pkt.GetDialRequest()
			resp.GetDialResponse().Random = dialReq.Random

			start := time.Now()
			conn, err := net.DialTimeout(dialReq.Protocol, dialReq.Address, dialTimeout)
			if err != nil {
				resp.GetDialResponse().Error = err.Error()
				if err := a.Send(resp); err != nil {
					klog.ErrorS(err, "could not send stream")
				}
				continue
			}
			metrics.Metrics.ObserveDialLatency(time.Since(start))

			connID := atomic.AddInt64(&a.nextConnID, 1)
			dataCh := make(chan []byte, xfrChannelSize)
			ctx := &connContext{
				conn:   conn,
				connID: connID,
				dataCh: dataCh,
				cleanFunc: func() {
					klog.V(4).InfoS("close connection", "connectionID", connID)
					resp := &client.Packet{
						Type:    client.PacketType_CLOSE_RSP,
						Payload: &client.Packet_CloseResponse{CloseResponse: &client.CloseResponse{}},
					}
					resp.GetCloseResponse().ConnectID = connID

					close(dataCh)
					a.connManager.Delete(connID)

					err := conn.Close()
					if err != nil {
						resp.GetCloseResponse().Error = err.Error()
					}

					if err := a.Send(resp); err != nil {
						klog.ErrorS(err, "close response failure")
					}
				},
				warnChLim: a.warnOnChannelLimit,
			}
			a.connManager.Add(connID, ctx)

			resp.GetDialResponse().ConnectID = connID
			if err := a.Send(resp); err != nil {
				klog.ErrorS(err, "stream send failure")
				continue
			}

			go a.remoteToProxy(connID, ctx)
			go a.proxyToRemote(connID, ctx)

		case client.PacketType_DATA:
			data := pkt.GetData()
			klog.V(4).InfoS("received DATA", "connectionID", data.ConnectID)

			ctx, ok := a.connManager.Get(data.ConnectID)
			if ok {
				ctx.send(data.Data)
			}

		case client.PacketType_CLOSE_REQ:
			closeReq := pkt.GetCloseRequest()
			connID := closeReq.ConnectID

			klog.V(4).InfoS("received CLOSE_REQ", "connectionID", connID)

			ctx, ok := a.connManager.Get(connID)
			if ok {
				ctx.cleanup()
			} else {
				klog.V(4).InfoS("Failed to find connection context for close", "connectionID", connID)
				resp := &client.Packet{
					Type:    client.PacketType_CLOSE_RSP,
					Payload: &client.Packet_CloseResponse{CloseResponse: &client.CloseResponse{}},
				}
				resp.GetCloseResponse().ConnectID = connID
				resp.GetCloseResponse().Error = "Unknown connectID"
				if err := a.Send(resp); err != nil {
					klog.ErrorS(err, "close response send failure", err)
					continue
				}
			}

		default:
			klog.V(2).InfoS("unrecognized packet", "type", pkt)
		}
	}
}

func (a *Client) remoteToProxy(connID int64, ctx *connContext) {
	defer func() {
		if panicInfo := recover(); panicInfo != nil {
			klog.V(2).InfoS("Exiting remoteToProxy with recovery", "panicInfo", panicInfo, "connectionID", connID)
		} else {
			klog.V(2).InfoS("Exiting remoteToProxy", "connectionID", connID)
		}
	}()
	defer ctx.cleanup()

	var buf [1 << 12]byte
	resp := &client.Packet{
		Type: client.PacketType_DATA,
	}

	for {
		n, err := ctx.conn.Read(buf[:])
		klog.V(5).InfoS("received data from remote", "bytes", n, "connectionID", connID)

		if err == io.EOF {
			klog.V(2).InfoS("connection EOF", "connectionID", connID)
			return
		} else if err != nil {
			// Normal when receive a CLOSE_REQ
			klog.ErrorS(err, "connection read failure", "connectionID", connID)
			return
		} else {
			resp.Payload = &client.Packet_Data{Data: &client.Data{
				Data:      buf[:n],
				ConnectID: connID,
			}}
			if err := a.Send(resp); err != nil {
				klog.ErrorS(err, "stream send failure", "connectionID", connID)
			}
		}
	}
}

func (a *Client) proxyToRemote(connID int64, ctx *connContext) {
	defer func() {
		if panicInfo := recover(); panicInfo != nil {
			klog.V(2).InfoS("Exiting proxyToRemote with recovery", "panicInfo", panicInfo, "connectionID", connID)
		} else {
			klog.V(2).InfoS("Exiting proxyToRemote", "connectionID", connID)
		}
	}()
	defer ctx.cleanup()

	for d := range ctx.dataCh {
		pos := 0
		for {
			n, err := ctx.conn.Write(d[pos:])
			if err == nil {
				klog.V(4).InfoS("write to remote", "connectionID", connID, "lastData", n, "dataSize", len(d))
				break
			} else if n > 0 {
				// https://golang.org/pkg/io/#Writer specifies return non nil error if n < len(d)
				klog.ErrorS(err, "write to remote with failure", "connectionID", connID, "lastData", n)
				pos += n
			} else {
				klog.ErrorS(err, "conn write failure", "connectionID", connID)
				return
			}
		}
	}
}

func (a *Client) probe() {
	for {
		select {
		case <-a.stopCh:
			return
		case <-time.After(a.probeInterval):
			if a.conn == nil {
				continue
			}
			// health check
			if a.conn.GetState() == connectivity.Ready {
				continue
			}
		}
		klog.V(1).InfoS("Removing client used for server connection", "state", a.conn.GetState(), "serverID", a.serverID)
		a.cs.RemoveClient(a.serverID)
		return
	}
}
