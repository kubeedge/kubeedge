/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	client "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	pkgagent "sigs.k8s.io/apiserver-network-proxy/pkg/agent"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server/metrics"
	"sigs.k8s.io/apiserver-network-proxy/proto/agent"
)

type ProxyStrategy string

const (
	// With this strategy the Proxy Server will randomly pick a backend from
	// the current healthy backends to establish the tunnel over which to
	// forward requests.
	ProxyStrategyDefault ProxyStrategy = "default"
	// With this strategy the Proxy Server will pick a backend that has the same
	// associated host as the request.Host to establish the tunnel.
	ProxyStrategyDestHost ProxyStrategy = "destHost"

	// ProxyStrategyDefaultRoute will only forward traffic to agents that have explicity advertised
	// they serve the default route through an agent identifier. Typically used in combination with destHost
	ProxyStrategyDefaultRoute ProxyStrategy = "defaultRoute"
)

// GenProxyStrategiesFromStr generates the list of proxy strategies from the
// comma-seperated string, i.e., destHost.
func GenProxyStrategiesFromStr(proxyStrategies string) ([]ProxyStrategy, error) {
	var ps []ProxyStrategy
	strs := strings.Split(proxyStrategies, ",")
	for _, s := range strs {
		switch s {
		case string(ProxyStrategyDestHost):
			ps = append(ps, ProxyStrategyDestHost)
		case string(ProxyStrategyDefault):
			ps = append(ps, ProxyStrategyDefault)
		case string(ProxyStrategyDefaultRoute):
			ps = append(ps, ProxyStrategyDefaultRoute)
		default:
			return nil, fmt.Errorf("Unknown proxy strategy %s", s)
		}
	}
	return ps, nil
}

type Backend interface {
	Send(p *client.Packet) error
	Context() context.Context
}

var _ Backend = &backend{}
var _ Backend = agent.AgentService_ConnectServer(nil)

type backend struct {
	// TODO: this is a multi-writer single-reader pattern, it's tricky to
	// write it using channel. Let's worry about performance later.
	mu   sync.Mutex // mu protects conn
	conn agent.AgentService_ConnectServer
}

func (b *backend) Send(p *client.Packet) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.conn.Send(p)
}

func (b *backend) Context() context.Context {
	// TODO: does Context require lock protection?
	return b.conn.Context()
}

func newBackend(conn agent.AgentService_ConnectServer) *backend {
	return &backend{conn: conn}
}

// BackendStorage is an interface to manage the storage of the backend
// connections, i.e., get, add and remove
type BackendStorage interface {
	// AddBackend adds a backend.
	AddBackend(identifier string, idType pkgagent.IdentifierType, conn agent.AgentService_ConnectServer) Backend
	// RemoveBackend removes a backend.
	RemoveBackend(identifier string, idType pkgagent.IdentifierType, conn agent.AgentService_ConnectServer)
	// NumBackends returns the number of backends.
	NumBackends() int
}

// BackendManager is an interface to manage backend connections, i.e.,
// connection to the proxy agents.
type BackendManager interface {
	// Backend returns a single backend.
	// WARNING: the context passed to the function should be a session-scoped
	// context instead of a request-scoped context, as the backend manager will
	// pick a backend for every tunnel session and each tunnel session may
	// contains multiple requests.
	Backend(ctx context.Context) (Backend, error)
	BackendStorage
	ReadinessManager
}

var _ BackendManager = &DefaultBackendManager{}

// DefaultBackendManager is the default backend manager.
type DefaultBackendManager struct {
	*DefaultBackendStorage
}

func (dbm *DefaultBackendManager) Backend(_ context.Context) (Backend, error) {
	klog.V(5).InfoS("Get a random backend through the DefaultBackendManager")
	return dbm.DefaultBackendStorage.GetRandomBackend()
}

// DefaultBackendStorage is the default backend storage.
type DefaultBackendStorage struct {
	mu sync.RWMutex //protects the following
	// A map between agentID and its grpc connections.
	// For a given agent, ProxyServer prefers backends[agentID][0] to send
	// traffic, because backends[agentID][1:] are more likely to be closed
	// by the agent to deduplicate connections to the same server.
	backends map[string][]*backend
	// agentID is tracked in this slice to enable randomly picking an
	// agentID in the Backend() method. There is no reliable way to
	// randomly pick a key from a map (in this case, the backends) in
	// Golang.
	agentIDs []string
	// defaultRouteAgentIDs tracks the agents that have claimed the default route.
	defaultRouteAgentIDs []string
	random               *rand.Rand
	// idTypes contains the valid identifier types for this
	// DefaultBackendStorage. The DefaultBackendStorage may only tolerate certain
	// types of identifiers when associating to a specific BackendManager,
	// e.g., when associating to the DestHostBackendManager, it can only use the
	// identifiers of types, IPv4, IPv6 and Host.
	idTypes []pkgagent.IdentifierType
}

// NewDefaultBackendManager returns a DefaultBackendManager.
func NewDefaultBackendManager() *DefaultBackendManager {
	return &DefaultBackendManager{
		DefaultBackendStorage: NewDefaultBackendStorage(
			[]pkgagent.IdentifierType{pkgagent.UID})}
}

// NewDefaultBackendStorage returns a DefaultBackendStorage
func NewDefaultBackendStorage(idTypes []pkgagent.IdentifierType) *DefaultBackendStorage {
	return &DefaultBackendStorage{
		backends: make(map[string][]*backend),
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
		idTypes:  idTypes,
	} /* #nosec G404 */
}

func containIDType(idTypes []pkgagent.IdentifierType, idType pkgagent.IdentifierType) bool {
	for _, it := range idTypes {
		if it == idType {
			return true
		}
	}
	return false
}

// AddBackend adds a backend.
func (s *DefaultBackendStorage) AddBackend(identifier string, idType pkgagent.IdentifierType, conn agent.AgentService_ConnectServer) Backend {
	if !containIDType(s.idTypes, idType) {
		klog.V(4).InfoS("fail to add backend", "backend", identifier, "error", &ErrWrongIDType{idType, s.idTypes})
		return nil
	}
	klog.V(2).InfoS("Register backend for agent", "connection", conn, "agentID", identifier)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.backends[identifier]
	addedBackend := newBackend(conn)
	if ok {
		for _, v := range s.backends[identifier] {
			if v.conn == conn {
				klog.V(1).InfoS("This should not happen. Adding existing backend for agent", "connection", conn, "agentID", identifier)
				return v
			}
		}
		s.backends[identifier] = append(s.backends[identifier], addedBackend)
		return addedBackend
	}
	s.backends[identifier] = []*backend{addedBackend}
	metrics.Metrics.SetBackendCount(len(s.backends))
	s.agentIDs = append(s.agentIDs, identifier)
	if idType == pkgagent.DefaultRoute {
		s.defaultRouteAgentIDs = append(s.defaultRouteAgentIDs, identifier)
	}
	return addedBackend
}

// RemoveBackend removes a backend.
func (s *DefaultBackendStorage) RemoveBackend(identifier string, idType pkgagent.IdentifierType, conn agent.AgentService_ConnectServer) {
	if !containIDType(s.idTypes, idType) {
		klog.ErrorS(&ErrWrongIDType{idType, s.idTypes}, "fail to remove backend")
		return
	}
	klog.V(2).InfoS("Remove connection for agent", "connection", conn, "identifier", identifier)
	s.mu.Lock()
	defer s.mu.Unlock()
	backends, ok := s.backends[identifier]
	if !ok {
		klog.V(1).InfoS("Cannot find agent in backends", "identifier", identifier)
		return
	}
	var found bool
	for i, c := range backends {
		if c.conn == conn {
			s.backends[identifier] = append(s.backends[identifier][:i], s.backends[identifier][i+1:]...)
			if i == 0 && len(s.backends[identifier]) != 0 {
				klog.V(1).InfoS("This should not happen. Removed connection that is not the first connection", "connection", conn, "remainingConnections", s.backends[identifier])
			}
			found = true
		}
	}
	if len(s.backends[identifier]) == 0 {
		delete(s.backends, identifier)
		for i := range s.agentIDs {
			if s.agentIDs[i] == identifier {
				s.agentIDs[i] = s.agentIDs[len(s.agentIDs)-1]
				s.agentIDs = s.agentIDs[:len(s.agentIDs)-1]
				break
			}
		}
		if idType == pkgagent.DefaultRoute {
			for i := range s.defaultRouteAgentIDs {
				if s.defaultRouteAgentIDs[i] == identifier {
					s.defaultRouteAgentIDs = append(s.defaultRouteAgentIDs[:i], s.defaultRouteAgentIDs[i+1:]...)
					break
				}
			}
		}
	}
	if !found {
		klog.V(1).InfoS("Could not find connection matching identifier to remove", "connection", conn, "identifier", identifier)
	}
	metrics.Metrics.SetBackendCount(len(s.backends))
}

// NumBackends resturns the number of available backends
func (s *DefaultBackendStorage) NumBackends() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.backends)
}

// ErrNotFound indicates that no backend can be found.
type ErrNotFound struct{}

// Error returns the error message.
func (e *ErrNotFound) Error() string {
	return "No backend available"
}

type ErrWrongIDType struct {
	got    pkgagent.IdentifierType
	expect []pkgagent.IdentifierType
}

func (e *ErrWrongIDType) Error() string {
	return fmt.Sprintf("incorrect id type: got %s, expect %s", e.got, e.expect)
}

func ignoreNotFound(err error) error {
	if _, ok := err.(*ErrNotFound); ok {
		return nil
	}
	return err
}

// GetRandomBackend returns a random backend connection from all connected agents.
func (s *DefaultBackendStorage) GetRandomBackend() (Backend, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.backends) == 0 {
		return nil, &ErrNotFound{}
	}
	agentID := s.agentIDs[s.random.Intn(len(s.agentIDs))]
	klog.V(4).InfoS("Pick agent as backend", "agentID", agentID)
	// always return the first connection to an agent, because the agent
	// will close later connections if there are multiple.
	return s.backends[agentID][0], nil
}
