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
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	pkgagent "sigs.k8s.io/apiserver-network-proxy/pkg/agent"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server/metrics"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"
	"sigs.k8s.io/apiserver-network-proxy/proto/header"
)

const xfrChannelSize = 10

type key int

type ProxyClientConnection struct {
	Mode      string
	Grpc      client.ProxyService_ProxyServer
	HTTP      io.ReadWriter
	CloseHTTP func() error
	connected chan struct{}
	connectID int64
	agentID   string
	start     time.Time
	backend   Backend
}

const (
	destHost key = iota
)

func (c *ProxyClientConnection) send(pkt *client.Packet) error {
	start := time.Now()
	defer metrics.Metrics.ObserveFrontendWriteLatency(time.Since(start))
	if c.Mode == "grpc" {
		stream := c.Grpc
		return stream.Send(pkt)
	} else if c.Mode == "http-connect" {
		if pkt.Type == client.PacketType_CLOSE_RSP {
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
	} else {
		return fmt.Errorf("attempt to send via unrecognized connection mode %q", c.Mode)
	}
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

func (pm *PendingDialManager) Get(random int64) (*ProxyClientConnection, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	clientConn, ok := pm.pendingDial[random]
	return clientConn, ok
}

func (pm *PendingDialManager) Remove(random int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.pendingDial, random)
	metrics.Metrics.SetPendingDialCount(len(pm.pendingDial))
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

	// Allows a special debug flag which warns if we write to a full transfer channel
	warnOnChannelLimit bool

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
	klog.V(2).InfoS("Register frontend for agent", "frontend", p, "agentID", agentID, "connectionID", connID)
	s.fmu.Lock()
	defer s.fmu.Unlock()
	if _, ok := s.frontends[agentID]; !ok {
		s.frontends[agentID] = make(map[int64]*ProxyClientConnection)
	}
	s.frontends[agentID][connID] = p
}

func (s *ProxyServer) removeFrontend(agentID string, connID int64) {
	s.fmu.Lock()
	defer s.fmu.Unlock()
	conns, ok := s.frontends[agentID]
	if !ok {
		klog.V(2).InfoS("Cannot find agent in the frontends", "agentID", agentID)
		return
	}
	if _, ok := conns[connID]; !ok {
		klog.V(2).InfoS("Cannot find connection for agent in the frontends", "connectionID", connID, "agentID", agentID)
		return
	}
	klog.V(2).InfoS("Remove frontend for agent", "frontend", conns[connID], "agentID", agentID, "connectionID", connID)
	delete(s.frontends[agentID], connID)
	if len(s.frontends[agentID]) == 0 {
		delete(s.frontends, agentID)
	}
	return
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

func (s *ProxyServer) getFrontendsForBackendConn(agentID string, backend Backend) ([]*ProxyClientConnection, error) {
	var ret []*ProxyClientConnection
	s.fmu.RLock()
	defer s.fmu.RUnlock()
	frontends, ok := s.frontends[agentID]
	if !ok {
		return nil, fmt.Errorf("can't find agentID %s in the frontends", agentID)
	}
	for _, frontend := range frontends {
		if frontend.backend == backend {
			ret = append(ret, frontend)
		}
	}
	return ret, nil
}

// NewProxyServer creates a new ProxyServer instance
func NewProxyServer(serverID string, proxyStrategies []ProxyStrategy, serverCount int, agentAuthenticationOptions *AgentTokenAuthenticationOptions, warnOnChannelLimit bool) *ProxyServer {
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
			klog.V(4).InfoS("Unknonw proxy strategy", "strategy", ps)
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
		Readiness:          bms[0],
		proxyStrategies:    proxyStrategies,
		warnOnChannelLimit: warnOnChannelLimit,
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
	klog.V(2).InfoS("proxy request from client", "userAgent", userAgent)

	recvCh := make(chan *client.Packet, xfrChannelSize)
	stopCh := make(chan error)

	go s.serveRecvFrontend(stream, recvCh)

	defer func() {
		klog.V(2).InfoS("Receive channel on Proxy is stopping", "userAgent", userAgent, "serverID", s.serverID)
		close(recvCh)
	}()

	// Start goroutine to receive packets from frontend and push to recvCh
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				klog.V(2).InfoS("Stream closed on Proxy", "userAgent", userAgent, "serverID", s.serverID)
				close(stopCh)
				return
			}
			if err != nil {
				klog.ErrorS(err, "Stream read from frontend failure")
				stopCh <- err
				close(stopCh)
				return
			}

			if s.warnOnChannelLimit && len(recvCh) >= xfrChannelSize {
				klog.V(2).InfoS("Receive channel on Proxy is full", "userAgent", userAgent, "serverID", s.serverID)
			}
			recvCh <- in
		}
	}()

	return <-stopCh
}

func (s *ProxyServer) serveRecvFrontend(stream client.ProxyService_ProxyServer, recvCh <-chan *client.Packet) {
	klog.V(4).Infoln("start serving frontend stream")

	var firstConnID int64
	// The first packet should be a DIAL_REQ, we will randomly get a
	// backend from the BackendManger then.
	var backend Backend
	var err error

	for pkt := range recvCh {
		switch pkt.Type {
		case client.PacketType_DIAL_REQ:
			klog.V(5).Infoln("Received DIAL_REQ")
			// TODO: if we track what agent has historically served
			// the address, then we can send the Dial_REQ to the
			// same agent. That way we save the agent from creating
			// a new connection to the address.
			backend, err = s.getBackend(pkt.GetDialRequest().Address)
			if err != nil {
				klog.ErrorS(err, "Failed to get a backend", "serverID", s.serverID)

				resp := &client.Packet{
					Type:    client.PacketType_DIAL_RSP,
					Payload: &client.Packet_DialResponse{DialResponse: &client.DialResponse{Error: err.Error()}},
				}
				if err := stream.Send(resp); err != nil {
					klog.V(5).Infoln("Failed to send DIAL_RSP for no backend", "error", err, "serverID", s.serverID)
				}
				// The Dial is failing; no reason to keep this goroutine.
				return
			}
			s.PendingDial.Add(
				pkt.GetDialRequest().Random,
				&ProxyClientConnection{
					Mode:      "grpc",
					Grpc:      stream,
					connected: make(chan struct{}),
					start:     time.Now(),
					backend:   backend,
				})
			if err := backend.Send(pkt); err != nil {
				klog.ErrorS(err, "DIAL_REQ to Backend failed", "serverID", s.serverID)
			}
			klog.V(5).Infoln("DIAL_REQ sent to backend") // got this. but backend didn't receive anything.

		case client.PacketType_CLOSE_REQ:
			connID := pkt.GetCloseRequest().ConnectID
			klog.V(5).InfoS("Received CLOSE_REQ", "connectionID", connID)
			if backend == nil {
				klog.V(2).InfoS("Backend has not been initialized for requested connection. Client should send a Dial Request first",
					"serverID", s.serverID, "connectionID", connID)
				continue
			}
			if err := backend.Send(pkt); err != nil {
				// TODO: retry with other backends connecting to this agent.
				klog.ErrorS(err, "CLOSE_REQ to Backend failed", "serverID", s.serverID, "connectionID", connID)
			}
			klog.V(5).Infoln("CLOSE_REQ sent to backend", "serverID", s.serverID, "connectionID", connID)

		case client.PacketType_DIAL_CLS:
			random := pkt.GetCloseDial().Random
			klog.V(5).InfoS("Received DIAL_CLOSE", "serverID", s.serverID, "dialID", random)
			// Currently not worrying about backend as we do not have an established connection,
			s.PendingDial.Remove(random)
			klog.V(5).Infoln("Removing pending dial request", "serverID", s.serverID, "dialID", random)

		case client.PacketType_DATA:
			connID := pkt.GetData().ConnectID
			data := pkt.GetData().Data
			klog.V(5).InfoS("Received data from connection", "bytes", len(data), "connectionID", connID)
			if firstConnID == 0 {
				firstConnID = connID
			} else if firstConnID != connID {
				klog.V(5).InfoS("Data does not match first connection id", "fistConnectionID", firstConnID, "connectionID", connID)
			}

			if backend == nil {
				klog.V(2).InfoS("Backend has not been initialized for the connection. Client should send a Dial Request first", "connectionID", connID)
				continue
			}
			if err := backend.Send(pkt); err != nil {
				// TODO: retry with other backends connecting to this agent.
				klog.ErrorS(err, "DATA to Backend failed", "serverID", s.serverID, "connectionID", connID)
				continue
			}
			klog.V(5).Infoln("DATA sent to Backend")

		default:
			klog.V(5).InfoS("Ignore packet coming from frontend",
				"type", pkt.Type, "serverID", s.serverID, "connectionID", firstConnID)
		}
	}

	klog.V(5).InfoS("Close streaming", "serverID", s.serverID, "connectionID", firstConnID)

	pkt := &client.Packet{
		Type: client.PacketType_CLOSE_REQ,
		Payload: &client.Packet_CloseRequest{
			CloseRequest: &client.CloseRequest{
				ConnectID: firstConnID,
			},
		},
	}

	if backend == nil {
		klog.V(2).InfoS("Backend has not been initialized for requested connection. Client should send a Dial Request first", "connectionID", firstConnID)
		return
	}
	if err := backend.Send(pkt); err != nil {
		klog.ErrorS(err, "CLOSE_REQ to Backend failed", "serverID", s.serverID)
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

func (s *ProxyServer) validateAuthToken(ctx context.Context, token string) error {
	trReq := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: []string{s.AgentAuthenticationOptions.AuthenticationAudience},
		},
	}
	r, err := s.AgentAuthenticationOptions.KubernetesClient.AuthenticationV1().TokenReviews().Create(ctx, trReq, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Failed to authenticate request. err:%v", err)
	}

	if r.Status.Error != "" {
		return fmt.Errorf("lookup failed: %s", r.Status.Error)
	}

	if !r.Status.Authenticated {
		return fmt.Errorf("lookup failed: service account jwt not valid")
	}

	// The username is of format: system:serviceaccount:(NAMESPACE):(SERVICEACCOUNT)
	parts := strings.Split(r.Status.User.Username, ":")
	if len(parts) != 4 {
		return fmt.Errorf("lookup failed: unexpected username format")
	}
	// Validate the user that comes back from token review is a service account
	if parts[0] != "system" || parts[1] != "serviceaccount" {
		return fmt.Errorf("lookup failed: username returned is not a service account")
	}

	ns := parts[2]
	sa := parts[3]
	if s.AgentAuthenticationOptions.AgentNamespace != ns {
		return fmt.Errorf("lookup failed: incoming request from %q namespace. Expected %q", ns, s.AgentAuthenticationOptions.AgentNamespace)
	}

	if s.AgentAuthenticationOptions.AgentServiceAccount != sa {
		return fmt.Errorf("lookup failed: incoming request from %q service account. Expected %q", sa, s.AgentAuthenticationOptions.AgentServiceAccount)
	}

	return nil
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

	if err := s.validateAuthToken(ctx, strings.TrimPrefix(authContext[0], header.AuthenticationTokenContextSchemePrefix)); err != nil {
		return fmt.Errorf("Failed to validate authentication token, err:%v", err)
	}

	klog.V(2).Infoln("Client successfully authenticated via token")
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

	klog.V(2).InfoS("Connect request from agent", "agentID", agentID)

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

	backend := s.addBackend(agentID, stream)
	defer s.removeBackend(agentID, stream)

	recvCh := make(chan *client.Packet, xfrChannelSize)

	go s.serveRecvBackend(backend, stream, agentID, recvCh)

	defer func() {
		klog.V(2).InfoS("Receive channel on Connect is stopping", "agentID", agentID, "serverID", s.serverID)
		close(recvCh)
	}()

	stopCh := make(chan error)
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				klog.V(2).InfoS("Stream closed on Connect", "agentID", agentID, "serverID", s.serverID)
				close(stopCh)
				return
			}
			if err != nil {
				klog.ErrorS(err, "stream read failure")
				stopCh <- err
				close(stopCh)
				return
			}

			if s.warnOnChannelLimit && len(recvCh) >= xfrChannelSize {
				klog.V(2).InfoS("Receive channel on Connect is full", "agentID", agentID, "serverID", s.serverID)
			}
			recvCh <- in
		}
	}()

	return <-stopCh
}

// route the packet back to the correct client
func (s *ProxyServer) serveRecvBackend(backend Backend, stream agent.AgentService_ConnectServer, agentID string, recvCh <-chan *client.Packet) {
	defer func() {
		// Close all connected frontends when the agent connection is closed
		// TODO(#126): Frontends in PendingDial state that have not been added to the
		//             list of frontends should also be closed.
		frontends, _ := s.getFrontendsForBackendConn(agentID, backend)
		klog.V(3).InfoS("Close frontends connected to agent",
			"serverID", s.serverID, "count", len(frontends), "agentID", agentID)

		for _, frontend := range frontends {
			s.removeFrontend(agentID, frontend.connectID)
			pkt := &client.Packet{
				Type: client.PacketType_CLOSE_RSP,
				Payload: &client.Packet_CloseResponse{
					CloseResponse: &client.CloseResponse{},
				},
			}
			pkt.GetCloseResponse().ConnectID = frontend.connectID
			if err := frontend.send(pkt); err != nil {
				klog.ErrorS(err, "CLOSE_RSP to frontend failed", "serverID", s.serverID, "agentID", agentID)
			}
		}
	}()

	for pkt := range recvCh {
		switch pkt.Type {
		case client.PacketType_DIAL_RSP:
			resp := pkt.GetDialResponse()
			klog.V(5).InfoS("Received DIAL_RSP", "dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)

			if frontend, ok := s.PendingDial.Get(resp.Random); !ok {
				klog.V(2).Infoln("DIAL_RSP not recognized; dropped", "dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)
			} else {
				dialErr := false
				if resp.Error != "" {
					klog.ErrorS(errors.New(resp.Error), "DIAL_RSP contains failure", "dialID", resp.Random, "agentID", agentID, "connectionID", resp.ConnectID)
					dialErr = true
				}
				err := frontend.send(pkt)
				s.PendingDial.Remove(resp.Random)
				if err != nil {
					klog.ErrorS(err, "DIAL_RSP send to frontend stream failure",
						"dialID", resp.Random, "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
					dialErr = true
				}
				// Avoid adding the frontend if there was an error dialing the destination
				if dialErr == true {
					break
				}
				frontend.connectID = resp.ConnectID
				frontend.agentID = agentID
				s.addFrontend(agentID, resp.ConnectID, frontend)
				close(frontend.connected)
				metrics.Metrics.ObserveDialLatency(time.Since(frontend.start))
			}

		case client.PacketType_DATA:
			resp := pkt.GetData()
			klog.V(5).InfoS("Received data from agent", "bytes", len(resp.Data), "agentID", agentID, "connectionID", resp.ConnectID)
			frontend, err := s.getFrontend(agentID, resp.ConnectID)
			if err != nil {
				klog.ErrorS(err, "could not get frontend client", "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
				break
			}
			if err := frontend.send(pkt); err != nil {
				klog.ErrorS(err, "send to client stream failure", "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
			} else {
				klog.V(5).InfoS("DATA sent to frontend")
			}

		case client.PacketType_CLOSE_RSP:
			resp := pkt.GetCloseResponse()
			klog.V(5).InfoS("Received CLOSE_RSP", "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
			frontend, err := s.getFrontend(agentID, resp.ConnectID)
			if err != nil {
				klog.ErrorS(err, "could not get frontend client", "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
				break
			}
			if err := frontend.send(pkt); err != nil {
				// Normal when frontend closes it.
				klog.ErrorS(err, "CLOSE_RSP send to client stream error", "serverID", s.serverID, "agentID", agentID, "connectionID", resp.ConnectID)
			} else {
				klog.V(5).Infoln("CLOSE_RSP sent to frontend", "connectionID", resp.ConnectID)
			}
			s.removeFrontend(agentID, resp.ConnectID)
			klog.V(5).InfoS("Close streaming", "agentID", agentID, "connectionID", resp.ConnectID)

		default:
			klog.V(2).InfoS("Unrecognized packet", "packet", pkt, "serverID", s.serverID, "agentID", agentID)
		}
	}
	klog.V(5).InfoS("Close backend of agent", "backend", stream, "serverID", s.serverID, "agentID", agentID)
}
