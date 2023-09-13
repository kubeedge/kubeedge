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

package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	runpprof "runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	commonmetrics "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/common/metrics"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	pkgagent "sigs.k8s.io/apiserver-network-proxy/pkg/agent"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server/metrics"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"
	"sigs.k8s.io/apiserver-network-proxy/proto/header"
)

const xfrChannelSize = 10

type key int

type GrpcFrontend struct {
	stream    client.ProxyService_ProxyServer
	streamUID string
	sendLock  sync.Mutex
	recvLock  sync.Mutex
}

func (g *GrpcFrontend) Send(pkt *client.Packet) error {
	g.sendLock.Lock()
	defer g.sendLock.Unlock()

	const segment = commonmetrics.SegmentToClient
	metrics.Metrics.ObservePacket(segment, pkt.Type)
	err := g.stream.Send(pkt)
	if err != nil {
		metrics.Metrics.ObserveStreamError(segment, err, pkt.Type)
	}
	return err
}

func (g *GrpcFrontend) Recv() (*client.Packet, error) {
	g.recvLock.Lock()
	defer g.recvLock.Unlock()

	const segment = commonmetrics.SegmentFromClient
	pkt, err := g.stream.Recv()
	if err != nil {
		if err != io.EOF {
			metrics.Metrics.ObserveStreamErrorNoPacket(segment, err)
		}
		return nil, err
	}
	metrics.Metrics.ObservePacket(segment, pkt.Type)
	return pkt, nil
}

type ProxyClientConnection struct {
	Mode        string
	HTTP        io.ReadWriter
	frontend    *GrpcFrontend
	CloseHTTP   func() error
	connected   chan struct{}
	dialID      int64
	connectID   int64
	agentID     string
	start       time.Time
	backend     Backend
	dialAddress string // cached for logging
}

const (
	destHost key = iota
)

func (c *ProxyClientConnection) send(pkt *client.Packet) error {
	start := time.Now()
	defer metrics.Metrics.ObserveFrontendWriteLatency(time.Since(start))
	if c.Mode == "grpc" {
		return c.frontend.Send(pkt)
	}
	if c.Mode == "http-connect" {
		if pkt.Type == client.PacketType_CLOSE_RSP {
			return c.CloseHTTP()
		} else if pkt.Type == client.PacketType_DIAL_CLS {
			return c.CloseHTTP()
		} else if pkt.Type == client.PacketType_DATA {
			_, err := c.HTTP.Write(pkt.GetData().Data)
			return err
		} else if pkt.Type == client.PacketType_DIAL_RSP {
			if pkt.GetDialResponse().Error != "" {
				return c.CloseHTTP()
			}
			return nil
		} else {
			return fmt.Errorf("attempt to send via unrecognized connection type %v", pkt.Type)
		}
	}
	return fmt.Errorf("attempt to send via unrecognized connection mode %q", c.Mode)
}

func NewPendingDialManager() *PendingDialManager {
	return &PendingDialManager{
		pendingDial: make(map[int64]*ProxyClientConnection),
	}
}

type PendingDialManager struct {
	mu          sync.RWMutex
	pendingDial map[int64]*ProxyClientConnection
}

func (pm *PendingDialManager) Add(random int64, clientConn *ProxyClientConnection) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pendingDial[random] = clientConn
	metrics.Metrics.SetPendingDialCount(len(pm.pendingDial))
}

func (pm *PendingDialManager) Remove(random int64) *ProxyClientConnection {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pd := pm.pendingDial[random]
	delete(pm.pendingDial, random)
	metrics.Metrics.SetPendingDialCount(len(pm.pendingDial))
	return pd
}

// removeForStream removes and returns all pending ProxyClientConnection associated with a
// given Proxy gRPC connection (expected to be at most 1 while konnectivity-client API gives
// single-use tunnels).
func (pm *PendingDialManager) removeForStream(streamUID string) []*ProxyClientConnection {
	var ret []*ProxyClientConnection
	if streamUID == "" {
		return ret
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for dialID, frontend := range pm.pendingDial {
		if frontend.frontend == nil {
			continue
		}
		if frontend.frontend.streamUID == streamUID {
			delete(pm.pendingDial, dialID)
			ret = append(ret, frontend)
		}
	}
	return ret
}

// ProxyServer
type ProxyServer struct {
	// BackendManagers contains a list of BackendManagers
	BackendManagers []BackendManager

	// Readiness reports if the proxy server is ready, i.e., if the proxy
	// server has connections to proxy agents (backends). Note that the
	// proxy server does not check the healthiness of the connections,
	// though the proxy agents do, so this readiness check might report
	// ready but there is no healthy connection.
	Readiness ReadinessManager

	// fmu protects frontends.
	fmu sync.RWMutex
	// conn = Frontend[agentID][connID]
	frontends map[string]map[int64]*ProxyClientConnection

	PendingDial *PendingDialManager

	serverID    string // unique ID of this server
	serverCount int    // Number of proxy server instances, should be 1 unless it is a HA server.

	// agent authentication
	AgentAuthenticationOptions *AgentTokenAuthenticationOptions

	proxyStrategies []ProxyStrategy
}

// AgentTokenAuthenticationOptions contains list of parameters required for agent token based authentication
type AgentTokenAuthenticationOptions struct {
	Enabled                bool
	AgentNamespace         string
	AgentServiceAccount    string
	AuthenticationAudience string
	KubernetesClient       kubernetes.Interface
}

var _ agent.AgentServiceServer = &ProxyServer{}

var _ client.ProxyServiceServer = &ProxyServer{}

func genContext(proxyStrategies []ProxyStrategy, reqHost string) context.Context {
	ctx := context.Background()
	for _, ps := range proxyStrategies {
		switch ps {
		case ProxyStrategyDestHost:
			addr := util.RemovePortFromHost(reqHost)
			ctx = context.WithValue(ctx, destHost, addr)
		}
	}
	return ctx
}

func (s *ProxyServer) getBackend(reqHost string) (Backend, error) {
	ctx := genContext(s.proxyStrategies, reqHost)
	for _, bm := range s.BackendManagers {
		be, err := bm.Backend(ctx)
		if err == nil {
			return be, nil
		}
		if ignoreNotFound(err) != nil {
			// if can't find a backend through current BackendManager, move on
			// to the next one
			return nil, err
		}
	}
	return nil, &ErrNotFound{}
}

func (s *ProxyServer) addBackend(agentID string, conn agent.AgentService_ConnectServer) (backend Backend) {
	for i := 0; i < len(s.BackendManagers); i++ {
		switch s.BackendManagers[i].(type) {
		case *DestHostBackendManager:
			agentIdentifiers, err := getAgentIdentifiers(conn)
			if err != nil {
				klog.ErrorS(err, "fail to get the agent identifiers", "agentID", agentID)
				break
			}
			for _, ipv4 := range agentIdentifiers.IPv4 {
				klog.V(5).InfoS("Add the agent to DestHostBackendManager", "agent address", ipv4)
				s.BackendManagers[i].AddBackend(ipv4, pkgagent.IPv4, conn)
			}
			for _, ipv6 := range agentIdentifiers.IPv6 {
				klog.V(5).InfoS("Add the agent to DestHostBackendManager", "agent address", ipv6)
				s.BackendManagers[i].AddBackend(ipv6, pkgagent.IPv6, conn)
			}
			for _, host := range agentIdentifiers.Host {
				klog.V(5).InfoS("Add the agent to DestHostBackendManager", "agent address", host)
				s.BackendManagers[i].AddBackend(host, pkgagent.Host, conn)
			}
		case *DefaultRouteBackendManager:
			agentIdentifiers, err := getAgentIdentifiers(conn)
			if err != nil {
				klog.ErrorS(err, "fail to get the agent identifiers", "agentID", agentID)
				break
			}
			if agentIdentifiers.DefaultRoute {
				klog.V(5).InfoS("Add the agent to DefaultRouteBackendManager", "agentID", agentID)
				backend = s.BackendManagers[i].AddBackend(agentID, pkgagent.DefaultRoute, conn)
			}
		default:
			klog.V(5).InfoS("Add the agent to DefaultBackendManager", "agentID", agentID)
			backend = s.BackendManagers[i].AddBackend(agentID, pkgagent.UID, conn)
		}
	}
	return
}

func (s *ProxyServer) removeBackend(agentID string, conn agent.AgentService_ConnectServer) {
	for _, bm := range s.BackendManagers {
		switch bm.(type) {
		case *DestHostBackendManager:
			agentIdentifiers, err := getAgentIdentifiers(conn)
			if err != nil {
				klog.ErrorS(err, "fail to get the agent identifiers", "agentID", agentID)
				break
			}
			for _, ipv4 := range agentIdentifiers.IPv4 {
				klog.V(5).InfoS("Remove the agent from the DestHostBackendManager", "agentHost", ipv4)
				bm.RemoveBackend(ipv4, pkgagent.IPv4, conn)
			}
			for _, ipv6 := range agentIdentifiers.IPv6 {
				klog.V(5).InfoS("Remove the agent from the DestHostBackendManager", "agentHost", ipv6)
				bm.RemoveBackend(ipv6, pkgagent.IPv6, conn)
			}
			for _, host := range agentIdentifiers.Host {
				klog.V(5).InfoS("Remove the agent from the DestHostBackendManager", "agentHost", host)
				bm.RemoveBackend(host, pkgagent.Host, conn)
			}
		case *DefaultRouteBackendManager:
			agentIdentifiers, err := getAgentIdentifiers(conn)
			if err != nil {
				klog.ErrorS(err, "fail to get the agent identifiers", "agentID", agentID)
				break
			}
			if agentIdentifiers.DefaultRoute {
				klog.V(5).InfoS("Remove the agent from the DefaultRouteBackendManager", "agentID", agentID)
				bm.RemoveBackend(agentID, pkgagent.DefaultRoute, conn)
			}
		default:
			klog.V(5).InfoS("Remove the agent from the DefaultBackendManager", "agentID", agentID)
			bm.RemoveBackend(agentID, pkgagent.UID, conn)
		}
	}
}

func (s *ProxyServer) addFrontend(agentID string, connID int64, p *ProxyClientConnection) {
	s.fmu.Lock()
	defer s.fmu.Unlock()
	if _, ok := s.frontends[agentID]; !ok {
		s.frontends[agentID] = make(map[int64]*ProxyClientConnection)
	}
	s.frontends[agentID][connID] = p

	metrics.Metrics.SetEstablishedConnCount(s.getCount(s.frontends))
}

func (s *ProxyServer) removeFrontend(agentID string, connID int64) *ProxyClientConnection {
	var ret *ProxyClientConnection
	s.fmu.Lock()
	defer s.fmu.Unlock()
	conns, ok := s.frontends[agentID]
	if !ok {
		return nil
	}
	if ret, ok = conns[connID]; !ok {
		return nil
	}
	delete(s.frontends[agentID], connID)
	if len(s.frontends[agentID]) == 0 {
		delete(s.frontends, agentID)
	}
	metrics.Metrics.SetEstablishedConnCount(s.getCount(s.frontends))
	return ret
}

func (s *ProxyServer) getFrontend(agentID string, connID int64) (*ProxyClientConnection, error) {
	s.fmu.RLock()
	defer s.fmu.RUnlock()
	conns, ok := s.frontends[agentID]
	if !ok {
		return nil, fmt.Errorf("can't find agentID %s in the frontends", agentID)
	}
	conn, ok := conns[connID]
	if !ok {
		return nil, fmt.Errorf("can't find connID %d in the frontends[%s]", connID, agentID)
	}
	return conn, nil
}

func (s *ProxyServer) removeFrontendsForBackendConn(agentID string, backend Backend) ([]*ProxyClientConnection, error) {
	var ret []*ProxyClientConnection
	if backend == nil {
		return ret, nil
	}
	s.fmu.Lock()
	defer s.fmu.Unlock()
	frontends, ok := s.frontends[agentID]
	if !ok {
		return nil, fmt.Errorf("can't find agentID %s in the frontends", agentID)
	}
	for _, frontend := range frontends {
		if frontend.backend == backend {
			delete(s.frontends, agentID)
			ret = append(ret, frontend)
		}
	}

	metrics.Metrics.SetEstablishedConnCount(s.getCount(s.frontends))
	return ret, nil
}

func (s *ProxyServer) getCount(frontends map[string](map[int64]*ProxyClientConnection)) int {
	count := 0
	for _, frontend := range frontends {
		count = count + len(frontend)
	}
	return count
}

// removeForStream removes and returns all established ProxyClientConnection associated with a given
// Proxy gRPC connection (expected to be at most 1 while konnectivity-client API gives single-use
// tunnels).
func (s *ProxyServer) removeFrontendsForStream(streamUID string) []*ProxyClientConnection {
	var ret []*ProxyClientConnection
	if streamUID == "" {
		return ret
	}
	s.fmu.Lock()
	defer s.fmu.Unlock()
	for agentID, frontends := range s.frontends {
		for connID, frontend := range frontends {
			if frontend.frontend == nil {
				continue
			}
			if frontend.frontend.streamUID == streamUID {
				delete(frontends, connID)
				ret = append(ret, frontend)
			}
		}
		if len(frontends) == 0 {
			delete(s.frontends, agentID)
		}
	}
	return ret
}

// NewProxyServer creates a new ProxyServer instance
func NewProxyServer(serverID string, proxyStrategies []ProxyStrategy, serverCount int, agentAuthenticationOptions *AgentTokenAuthenticationOptions) *ProxyServer {
	var bms []BackendManager
	for _, ps := range proxyStrategies {
		switch ps {
		case ProxyStrategyDestHost:
			bms = append(bms, NewDestHostBackendManager())
		case ProxyStrategyDefault:
			bms = append(bms, NewDefaultBackendManager())
		case ProxyStrategyDefaultRoute:
			bms = append(bms, NewDefaultRouteBackendManager())
		default:
			klog.ErrorS(nil, "Unknown proxy strategy", "strategy", ps)
		}
	}

	return &ProxyServer{
		frontends:                  make(map[string](map[int64]*ProxyClientConnection)),
		PendingDial:                NewPendingDialManager(),
		serverID:                   serverID,
		serverCount:                serverCount,
		BackendManagers:            bms,
		AgentAuthenticationOptions: agentAuthenticationOptions,
		// use the first backend-manager as the Readiness Manager
		Readiness:       bms[0],
		proxyStrategies: proxyStrategies,
	}
}

// Proxy handles incoming streams from gRPC frontend.
func (s *ProxyServer) Proxy(stream client.ProxyService_ProxyServer) error {
	metrics.Metrics.ConnectionInc(metrics.Proxy)
	defer metrics.Metrics.ConnectionDec(metrics.Proxy)

	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return fmt.Errorf("failed to get context")
	}
	userAgent := md.Get(header.UserAgent)
	streamUID := uuid.New().String()
	klog.V(5).InfoS("Proxy request from client", "userAgent", userAgent, "serverID", s.serverID, "streamUID", streamUID)

	recvCh := make(chan *client.Packet, xfrChannelSize)
	stopCh := make(chan error, 1)

	frontend := GrpcFrontend{
		stream:    stream,
		streamUID: streamUID,
	}

	defer func() {
		// The frontend stream and goroutines are completely shut down, so we will never get a
		// subsequent close message from the frontend. This can happen for several reasons, such as
		// stream canceled. Clean any lingering tunnel connections to avoid resource leaks.
		for _, p := range s.PendingDial.removeForStream(streamUID) {
			klog.V(2).InfoS("frontend stream shutdown, cleaning dial", "dialID", p.dialID)
			// TODO: add agent support to handle this
			s.sendBackendDialClose(p.backend, p.dialID, "frontend stream shutdown")
		}
		for _, f := range s.removeFrontendsForStream(streamUID) {
			klog.V(2).InfoS("frontend stream shutdown, cleaning frontend", "connectionID", f.connectID, "dialID", f.dialID)
			s.sendBackendClose(f.backend, f.connectID, f.dialID, "frontend stream shutdown")
		}
	}()

	labels := runpprof.Labels(
		"serverCount", strconv.Itoa(s.serverCount),
		"userAgent", strings.Join(userAgent, ", "),
	)
	// Start goroutine to receive packets from frontend and push to recvCh
	go runpprof.Do(context.Background(), labels, func(context.Context) { s.readFrontendToChannel(&frontend, userAgent, recvCh, stopCh) })

	s.serveRecvFrontend(&frontend, recvCh)

	return <-stopCh
}

func (s *ProxyServer) readFrontendToChannel(frontend *GrpcFrontend, userAgent []string, recvCh chan *client.Packet, stopCh chan error) {
	defer close(stopCh)
	defer close(recvCh)

	for {
		in, err := frontend.Recv()
		if err == io.EOF {
			klog.V(2).InfoS("Receive stream from frontend closed", "userAgent", userAgent)
			return
		}
		if err != nil {
			if status.Code(err) == codes.Canceled {
				klog.V(2).InfoS("Stream read from frontend cancelled", "userAgent", userAgent)
			} else {
				klog.ErrorS(err, "Stream read from frontend failure", "userAgent", userAgent)
			}
			stopCh <- err
			return
		}

		select {
		case recvCh <- in: // Send didn't block, carry on.
		default: // Send blocked; record it and try again.
			klog.V(2).InfoS("Receive channel from frontend is full", "userAgent", userAgent)
			fullRecvChannelMetric := metrics.Metrics.FullRecvChannel(metrics.Proxy)
			fullRecvChannelMetric.Inc()
			recvCh <- in
			fullRecvChannelMetric.Dec()
		}
	}
}

func (s *ProxyServer) serveRecvFrontend(frontend *GrpcFrontend, recvCh <-chan *client.Packet) {
	klog.V(5).Infoln("start serving frontend stream")

	var firstConnID int64
	// The first packet should be a DIAL_REQ, we will randomly get a
	// backend from the BackendManger then.
	// TODO: either add agentID to protocol (DATA, CLOSE_RSP, etc) or replace {agentID,
	// connectionID} with a simpler key (#462).
	var backend Backend
	var err error

	defer func() {
		klog.V(5).InfoS("Close frontend streaming", "connectionID", firstConnID, "streamUID", frontend.streamUID)
		if backend == nil {
			klog.V(2).InfoS("Streaming closed before backend intialized")
		}

		// As the read side of the recvCh channel, we cannot close it.
		// However readFrontendToChannel() may be blocked writing to the channel,
		// so we need to consume the channel until it is closed.
		discardedPktCount := 0
		for range recvCh {
			// Ignore values as this indicates there was a problem
			// with the remote connection.
			discardedPktCount++
		}
		if discardedPktCount > 0 {
			klog.V(2).InfoS("Discard packets while exiting serveRecvFrontend", "pktCount", discardedPktCount, "connectionID", firstConnID)
		}
	}()

	for pkt := range recvCh {
		switch pkt.Type {
		case client.PacketType_DIAL_REQ:
			random := pkt.GetDialRequest().Random
			address := pkt.GetDialRequest().Address
			klog.V(3).InfoS("Received DIAL_REQ", "dialID", random, "dialAddress", address)
			// TODO: if we track what agent has historically served
			// the address, then we can send the Dial_REQ to the
			// same agent. That way we save the agent from creating
			// a new connection to the address.
			backend, err = s.getBackend(address)
			if err != nil {
				klog.ErrorS(err, "Failed to get a backend", "dialID", random)
				metrics.Metrics.ObserveDialFailure(metrics.DialFailureNoAgent)

				resp := &client.Packet{
					Type: client.PacketType_DIAL_RSP,
					Payload: &client.Packet_DialResponse{
						DialResponse: &client.DialResponse{
							Random: random,
							Error:  err.Error(),
						},
					},
				}
				if err := frontend.Send(resp); err != nil {
					klog.V(5).InfoS("Failed to send DIAL_RSP for no backend", "error", err, "dialID", random)
				}
				// The Dial is failing; no reason to keep this goroutine.
				return
			}
			s.PendingDial.Add(
				random,
				&ProxyClientConnection{
					Mode:        "grpc",
					frontend:    frontend,
					dialID:      random,
					connected:   make(chan struct{}),
					start:       time.Now(),
					backend:     backend,
					dialAddress: address,
				})
			if err := backend.Send(pkt); err != nil {
				klog.ErrorS(err, "DIAL_REQ to Backend failed", "dialID", random)
			} else {
				klog.V(5).InfoS("DIAL_REQ sent to backend", "dialID", random)
			}

		case client.PacketType_CLOSE_REQ:
			connID := pkt.GetCloseRequest().ConnectID
			klog.V(5).InfoS("Received CLOSE_REQ", "connectionID", connID)
			if backend == nil {
				klog.V(2).InfoS("Backend has not been initialized for this connection", "connectionID", connID)
				s.sendFrontendClose(frontend, connID, "backend uninitialized")
				continue
			}
			if err := backend.Send(pkt); err != nil {
				// TODO: retry with other backends connecting to this agent.
				klog.ErrorS(err, "CLOSE_REQ to Backend failed", "connectionID", connID)
				s.sendFrontendClose(frontend, connID, "CLOSE_REQ to backend failed")
			} else {
				klog.V(5).InfoS("CLOSE_REQ sent to backend", "connectionID", connID)
			}
			klog.V(3).InfoS("Closing frontend streaming per CLOSE_REQ", "connectionID", connID)
			return

		case client.PacketType_DIAL_CLS:
			random := pkt.GetCloseDial().Random
			klog.V(5).InfoS("Received DIAL_CLOSE", "dialID", random)
			// Currently not worrying about backend as we do not have an established connection,
			if pd := s.PendingDial.Remove(random); pd != nil {
				klog.ErrorS(nil, "Dial cancelled (DIAL_CLS) by frontend",
					"dialID", random,
					"dialAddress", pd.dialAddress,
					"dialDuration", time.Since(pd.start),
				)
				metrics.Metrics.ObserveDialFailure(metrics.DialFailureFrontendClose)
			} else {
				// TODO: Fix the leaked connection due to race (client cancels a pending dial but
				// server has completed the dial).
				klog.ErrorS(nil, "Unrecognized dial cancelled (DIAL_CLS) by frontend", "dialID", random)
			}

		case client.PacketType_DATA:
			connID := pkt.GetData().ConnectID
			data := pkt.GetData().Data
			klog.V(5).InfoS("Received data from connection", "bytes", len(data), "connectionID", connID)
			if backend == nil {
				klog.V(2).InfoS("Backend has not been initialized for this connection", "connectionID", connID)
				s.sendFrontendClose(frontend, connID, "backend not initialized")
				return
			}

			if connID == 0 {
				klog.ErrorS(nil, "Received packet missing ConnectID from frontend", "packetType", "DATA")
				continue
			}
			if firstConnID == 0 {
				firstConnID = connID
			} else if firstConnID != connID {
				klog.ErrorS(nil, "Data does not match first connection id", "firstConnectionID", firstConnID, "connectionID", connID)
				// Something went very wrong if we get here. Close both connections to avoid leaks.
				s.sendBackendClose(backend, connID, 0, "mismatched connection IDs")
				s.sendBackendClose(backend, firstConnID, 0, "mismatched connection IDs")
				s.sendFrontendClose(frontend, connID, "mismatched connection IDs")
				s.sendFrontendClose(frontend, firstConnID, "mismatched connection IDs")
				return
			}
			if err := backend.Send(pkt); err != nil {
				// TODO: retry with other backends connecting to this agent.
				klog.ErrorS(err, "DATA to Backend failed", "connectionID", connID)
				continue
			}
			klog.V(5).Infoln("DATA sent to Backend")

		default:
			klog.V(5).InfoS("Ignoring unrecognized packet from frontend",
				"type", pkt.Type, "connectionID", firstConnID)
		}
	}
}

func agentID(stream agent.AgentService_ConnectServer) (string, error) {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return "", fmt.Errorf("failed to get context")
	}
	agentIDs := md.Get(header.AgentID)
	if len(agentIDs) != 1 {
		return "", fmt.Errorf("expected one agent ID in the context, got %v", agentIDs)
	}
	return agentIDs[0], nil
}

func getAgentIdentifiers(stream agent.AgentService_ConnectServer) (pkgagent.Identifiers, error) {
	var agentIdentifiers pkgagent.Identifiers
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return agentIdentifiers, fmt.Errorf("failed to get context")
	}
	agentIDs := md.Get(header.AgentIdentifiers)
	if len(agentIDs) > 1 {
		return agentIdentifiers, fmt.Errorf("expected at most one agent IP in the context, got %v", agentIDs)
	}
	if len(agentIDs) == 0 {
		return agentIdentifiers, nil
	}

	agentIdentifiers, err := pkgagent.GenAgentIdentifiers(agentIDs[0])
	if err != nil {
		return agentIdentifiers, err
	}
	return agentIdentifiers, nil
}

func (s *ProxyServer) validateAuthToken(ctx context.Context, token string) (username string, err error) {
	trReq := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: []string{s.AgentAuthenticationOptions.AuthenticationAudience},
		},
	}
	r, err := s.AgentAuthenticationOptions.KubernetesClient.AuthenticationV1().TokenReviews().Create(ctx, trReq, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to authenticate request. err:%v", err)
	}

	if r.Status.Error != "" {
		return "", fmt.Errorf("lookup failed: %s", r.Status.Error)
	}

	if !r.Status.Authenticated {
		return "", fmt.Errorf("lookup failed: service account jwt not valid")
	}

	// The username is of format: system:serviceaccount:(NAMESPACE):(SERVICEACCOUNT)
	username = r.Status.User.Username
	parts := strings.Split(username, ":")
	if len(parts) != 4 {
		return "", fmt.Errorf("lookup failed: unexpected username format")
	}
	// Validate the user that comes back from token review is a service account
	if parts[0] != "system" || parts[1] != "serviceaccount" {
		return "", fmt.Errorf("lookup failed: username returned is not a service account")
	}

	ns := parts[2]
	sa := parts[3]
	if s.AgentAuthenticationOptions.AgentNamespace != ns {
		return "", fmt.Errorf("lookup failed: incoming request from %q namespace. Expected %q", ns, s.AgentAuthenticationOptions.AgentNamespace)
	}

	if s.AgentAuthenticationOptions.AgentServiceAccount != sa {
		return "", fmt.Errorf("lookup failed: incoming request from %q service account. Expected %q", sa, s.AgentAuthenticationOptions.AgentServiceAccount)
	}

	return username, nil
}

func (s *ProxyServer) authenticateAgentViaToken(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("Failed to retrieve metadata from context")
	}

	authContext := md.Get(header.AuthenticationTokenContextKey)
	if len(authContext) == 0 {
		return fmt.Errorf("Authentication context was not found in metadata")
	}

	if len(authContext) > 1 {
		return fmt.Errorf("too many (%d) tokens are received", len(authContext))
	}

	if !strings.HasPrefix(authContext[0], header.AuthenticationTokenContextSchemePrefix) {
		return fmt.Errorf("received token does not have %q prefix", header.AuthenticationTokenContextSchemePrefix)
	}

	username, err := s.validateAuthToken(ctx, strings.TrimPrefix(authContext[0], header.AuthenticationTokenContextSchemePrefix))
	if err != nil {
		return fmt.Errorf("Failed to validate authentication token, err:%v", err)
	}

	klog.V(5).InfoS("Agent successfully authenticated via token", "username", username)
	return nil
}

// Connect is for agent to connect to ProxyServer as next hop
func (s *ProxyServer) Connect(stream agent.AgentService_ConnectServer) error {
	metrics.Metrics.ConnectionInc(metrics.Connect)
	defer metrics.Metrics.ConnectionDec(metrics.Connect)

	agentID, err := agentID(stream)
	if err != nil {
		return err
	}

	klog.V(5).InfoS("Connect request from agent", "agentID", agentID, "serverID", s.serverID)
	labels := runpprof.Labels(
		"serverCount", strconv.Itoa(s.serverCount),
		"agentID", agentID,
	)
	ctx := runpprof.WithLabels(context.Background(), labels)
	runpprof.SetGoroutineLabels(ctx)

	if s.AgentAuthenticationOptions.Enabled {
		if err := s.authenticateAgentViaToken(stream.Context()); err != nil {
			klog.ErrorS(err, "Client authentication failed", "agentID", agentID)
			return err
		}
	}

	h := metadata.Pairs(header.ServerID, s.serverID, header.ServerCount, strconv.Itoa(s.serverCount))
	if err := stream.SendHeader(h); err != nil {
		klog.ErrorS(err, "Failed to send server count back to agent", "agentID", agentID)
		return err
	}

	klog.V(2).InfoS("Agent connected", "agentID", agentID, "serverID", s.serverID)
	backend := s.addBackend(agentID, stream)
	defer s.removeBackend(agentID, stream)

	recvCh := make(chan *client.Packet, xfrChannelSize)

	go runpprof.Do(context.Background(), labels, func(context.Context) { s.serveRecvBackend(backend, agentID, recvCh) })

	defer func() {
		close(recvCh)
	}()

	stopCh := make(chan error)
	go runpprof.Do(context.Background(), labels, func(context.Context) { s.readBackendToChannel(backend, recvCh, stopCh) })

	return <-stopCh
}

func (s *ProxyServer) readBackendToChannel(backend Backend, recvCh chan *client.Packet, stopCh chan error) {
	for {
		in, err := backend.Recv()
		if err == io.EOF {
			klog.V(2).InfoS("Receive stream from agent is closed", "agentID", agentID)
			close(stopCh)
			return
		}
		if err != nil {
			klog.ErrorS(err, "Receive stream from agent read failure")
			stopCh <- err
			close(stopCh)
			return
		}

		select {
		case recvCh <- in: // Send didn't block, carry on.
		default: // Send blocked; record it and try again.
			klog.V(2).InfoS("Receive channel from agent is full", "agentID", agentID)
			fullRecvChannelMetric := metrics.Metrics.FullRecvChannel(metrics.Connect)
			fullRecvChannelMetric.Inc()
			recvCh <- in
			fullRecvChannelMetric.Dec()
		}
	}
}

// route the packet back to the correct client
func (s *ProxyServer) serveRecvBackend(backend Backend, agentID string, recvCh <-chan *client.Packet) {
	defer func() {
		// Drain recvCh to ensure that readBackendToChannel is not blocked on a channel write.
		// This should never happen, as termination of this function should only be initiated by closing recvCh.
		discardedPktCount := 0
		for range recvCh {
			discardedPktCount++
		}
		if discardedPktCount > 0 {
			klog.V(2).InfoS("Discard packets while exiting serveRecvBackend", "pktCount", discardedPktCount, "agentID", agentID)
		}
	}()

	defer func() {
		// Close all connected frontends when the agent connection is closed
		// TODO(#126): Frontends in PendingDial state that have not been added to the
		//             list of frontends should also be closed.
		frontends, err := s.removeFrontendsForBackendConn(agentID, backend)
		if err != nil {
			return
		}
		if len(frontends) > 0 {
			klog.V(2).InfoS("Close frontends connected to agent",
				"count", len(frontends), "agentID", agentID)
		}

		for _, frontend := range frontends {
			pkt := &client.Packet{
				Type: client.PacketType_CLOSE_RSP,
				Payload: &client.Packet_CloseResponse{
					CloseResponse: &client.CloseResponse{},
				},
			}
			pkt.GetCloseResponse().ConnectID = frontend.connectID
			if err := frontend.send(pkt); err != nil {
				klog.ErrorS(err, "CLOSE_RSP to frontend failed", "agentID", agentID)
			}
		}
	}()

	for pkt := range recvCh {
		switch pkt.Type {
		case client.PacketType_DIAL_RSP:
			resp := pkt.GetDialResponse()
			klog.V(5).InfoS("Received DIAL_RSP", "dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)

			frontend := s.PendingDial.Remove(resp.Random)
			if frontend == nil {
				klog.V(2).InfoS("DIAL_RSP not recognized; dropped", "dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)
				metrics.Metrics.ObserveDialFailure(metrics.DialFailureUnrecognizedResponse)
				if resp.ConnectID != 0 {
					s.sendBackendClose(backend, resp.ConnectID, resp.Random, "unknown dial id")
				}
			} else {
				dialErr := false
				if resp.Error != "" {
					// Dial response with error should not contain a valid ConnID.
					klog.ErrorS(errors.New(resp.Error), "DIAL_RSP contains failure", "dialID", resp.Random, "agentID", agentID)
					metrics.Metrics.ObserveDialFailure(metrics.DialFailureErrorResponse)
					dialErr = true
				}
				err := frontend.send(pkt)
				if err != nil {
					klog.ErrorS(err, "DIAL_RSP send to frontend stream failure",
						"dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)
					if !dialErr { // Avoid double-counting.
						metrics.Metrics.ObserveDialFailure(metrics.DialFailureSendResponse)
					}
					// If we never finish setting up the tunnel for ConnectID, then the connection is dead.
					// Currently, the agent will no resend DIAL_RSP, so connection is dead.
					// We already attempted to tell the frontend that. We should ensure we tell the backend.
					s.sendBackendClose(backend, resp.ConnectID, resp.Random, "dial error")
					dialErr = true
				}
				// Avoid adding the frontend if there was an error dialing the destination
				if dialErr {
					break
				}
				frontend.connectID = resp.ConnectID
				frontend.agentID = agentID
				// TODO: this connection may be cleaned on serveRecvFrontend exit, make it independent.
				s.addFrontend(agentID, resp.ConnectID, frontend)
				close(frontend.connected)
				metrics.Metrics.ObserveDialLatency(time.Since(frontend.start))
				klog.V(3).InfoS("Proxy connection established",
					"dialID", resp.Random,
					"connectionID", resp.ConnectID,
					"agentID", agentID,
					"dialAddress", frontend.dialAddress,
					"dialDuration", time.Since(frontend.start),
				)
			}

		case client.PacketType_DIAL_CLS:
			resp := pkt.GetCloseDial()
			klog.V(5).InfoS("Received DIAL_CLS", "agentID", agentID, "dialID", resp.Random)
			frontend := s.PendingDial.Remove(resp.Random)
			if frontend == nil {
				klog.V(2).InfoS("DIAL_CLS not recognized; dropped", "dialID", resp.Random, "agentID", agentID)
			} else {
				if err := frontend.send(pkt); err != nil {
					klog.ErrorS(err, "DIAL_CLS send to client stream error", "agentID", agentID, "dialID", resp.Random)
				} else {
					klog.V(5).InfoS("DIAL_CLS sent to frontend", "dialID", resp.Random)
				}
				klog.ErrorS(nil, "Dial terminated (DIAL_CLS) by backend",
					"dialID", resp.Random,
					"agentID", agentID,
					"dialAddress", frontend.dialAddress,
					"dialDuration", time.Since(frontend.start),
				)
				metrics.Metrics.ObserveDialFailure(metrics.DialFailureBackendClose)
			}

		case client.PacketType_DATA:
			resp := pkt.GetData()
			klog.V(5).InfoS("Received data from agent", "bytes", len(resp.Data), "agentID", agentID, "connectionID", resp.ConnectID)
			if resp.ConnectID == 0 {
				klog.ErrorS(nil, "Received packet missing ConnectID from agent", "packetType", "DATA")
				continue
			}

			frontend, err := s.getFrontend(agentID, resp.ConnectID)
			if err != nil {
				klog.V(2).InfoS("could not get frontend client; closing connection", "agentID", agentID, "connectionID", resp.ConnectID, "error", err)
				s.sendBackendClose(backend, resp.ConnectID, 0, "missing frontend")
				break
			}
			if err := frontend.send(pkt); err != nil {
				klog.ErrorS(err, "send to client stream failure", "agentID", agentID, "connectionID", resp.ConnectID)
			} else {
				klog.V(5).InfoS("DATA sent to frontend")
			}

		case client.PacketType_CLOSE_RSP:
			resp := pkt.GetCloseResponse()
			klog.V(5).InfoS("Received CLOSE_RSP", "agentID", agentID, "connectionID", resp.ConnectID)
			frontend := s.removeFrontend(agentID, resp.ConnectID)
			if frontend == nil {
				// assuming it is already closed, just log it
				klog.V(2).InfoS("could not get frontend client for closing", "agentID", agentID, "connectionID", resp.ConnectID)
				break
			}
			if err := frontend.send(pkt); err != nil {
				// Normal when frontend closes it.
				klog.ErrorS(err, "CLOSE_RSP send to client stream error", "agentID", agentID, "connectionID", resp.ConnectID)
			} else {
				klog.V(5).InfoS("CLOSE_RSP sent to frontend", "connectionID", resp.ConnectID)
			}

		default:
			klog.V(5).InfoS("Ignoring unrecognized packet from backend", "packet", pkt, "agentID", agentID)
		}
	}
	klog.V(5).InfoS("Close backend of agent", "agentID", agentID)
}

func (s *ProxyServer) sendBackendClose(backend Backend, connectID int64, random int64, reason string) {
	pkt := &client.Packet{
		Type: client.PacketType_CLOSE_REQ,
		Payload: &client.Packet_CloseRequest{
			CloseRequest: &client.CloseRequest{
				ConnectID: connectID,
			},
		},
	}
	if err := backend.Send(pkt); err != nil {
		klog.V(5).ErrorS(err, "Failed to send close to agent", "closeReason", reason, "dialID", random, "agentID", agentID, "connectionID", connectID)
	}
}

func (s *ProxyServer) sendBackendDialClose(backend Backend, random int64, reason string) {
	pkt := &client.Packet{
		Type: client.PacketType_DIAL_CLS,
		Payload: &client.Packet_CloseDial{
			CloseDial: &client.CloseDial{
				Random: random,
			},
		},
	}
	if err := backend.Send(pkt); err != nil {
		klog.V(5).ErrorS(err, "Failed to send dial close to agent", "closeReason", reason, "dialID", random, "agentID", agentID)
	}
}

func (s *ProxyServer) sendFrontendClose(frontend *GrpcFrontend, connectID int64, reason string) {
	pkt := &client.Packet{
		Type: client.PacketType_CLOSE_RSP,
		Payload: &client.Packet_CloseResponse{
			CloseResponse: &client.CloseResponse{
				ConnectID: connectID,
				Error:     reason,
			},
		},
	}
	if err := frontend.Send(pkt); err != nil {
		klog.V(5).ErrorS(err, "Failed to send close to frontend", "closeReason", reason, "connectionID", connectID)
	}
}
