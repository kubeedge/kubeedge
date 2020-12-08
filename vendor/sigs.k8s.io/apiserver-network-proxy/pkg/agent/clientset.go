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

package agent

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// ClientSet consists of clients connected to each instance of an HA proxy server.
type ClientSet struct {
	mu      sync.Mutex              //protects the clients.
	clients map[string]*AgentClient // map between serverID and the client
	// connects to this server.

	agentID     string // ID of this agent
	address     string // proxy server address. Assuming HA proxy server
	serverCount int    // number of proxy server instances, should be 1
	// unless it is an HA server. Initialized when the ClientSet creates
	// the first client.
	syncInterval time.Duration // The interval by which the agent
	// periodically checks that it has connections to all instances of the
	// proxy server.
	probeInterval time.Duration // The interval by which the agent
	// periodically checks if its connections to the proxy server is ready.
	dialOptions []grpc.DialOption
	// file path contains service account token
	serviceAccountTokenPath string
	// channel to signal shutting down the client set. Primarily for test.
	stopCh <-chan struct{}
}

func (cs *ClientSet) ClientsCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.clients)
}

func (cs *ClientSet) HealthyClientsCount() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var count int
	for _, c := range cs.clients {
		if c.conn.GetState() == connectivity.Ready {
			count++
		}
	}
	return count

}

func (cs *ClientSet) hasIDLocked(serverID string) bool {
	_, ok := cs.clients[serverID]
	return ok
}

func (cs *ClientSet) HasID(serverID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.hasIDLocked(serverID)
}

func (cs *ClientSet) addClientLocked(serverID string, c *AgentClient) error {
	if cs.hasIDLocked(serverID) {
		return fmt.Errorf("client for proxy server %s already exists", serverID)
	}
	cs.clients[serverID] = c
	return nil

}

func (cs *ClientSet) AddClient(serverID string, c *AgentClient) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.addClientLocked(serverID, c)
}

func (cs *ClientSet) RemoveClient(serverID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.clients[serverID] == nil {
		return
	}
	cs.clients[serverID].Close()
	delete(cs.clients, serverID)
}

type ClientSetConfig struct {
	Address                 string
	AgentID                 string
	SyncInterval            time.Duration
	ProbeInterval           time.Duration
	DialOptions             []grpc.DialOption
	ServiceAccountTokenPath string
}

func (cc *ClientSetConfig) NewAgentClientSet(stopCh <-chan struct{}) *ClientSet {
	return &ClientSet{
		clients:                 make(map[string]*AgentClient),
		agentID:                 cc.AgentID,
		address:                 cc.Address,
		syncInterval:            cc.SyncInterval,
		probeInterval:           cc.ProbeInterval,
		dialOptions:             cc.DialOptions,
		serviceAccountTokenPath: cc.ServiceAccountTokenPath,
		stopCh:                  stopCh,
	}
}

func (cs *ClientSet) newAgentClient() (*AgentClient, int, error) {
	return newAgentClient(cs.address, cs.agentID, cs, cs.dialOptions...)
}

func (cs *ClientSet) resetBackoff() *wait.Backoff {
	return &wait.Backoff{
		Steps:    3,
		Jitter:   0.1,
		Factor:   1.5,
		Duration: cs.syncInterval,
		Cap:      60 * time.Second,
	}
}

// sync makes sure that #clients >= #proxy servers
func (cs *ClientSet) sync() {
	defer cs.shutdown()
	backoff := cs.resetBackoff()
	var duration time.Duration
	for {
		if err := cs.syncOnce(); err != nil {
			klog.ErrorS(err, "cannot sync once")
			duration = backoff.Step()
		} else {
			backoff = cs.resetBackoff()
			duration = wait.Jitter(backoff.Duration, backoff.Jitter)
		}
		time.Sleep(duration)
		select {
		case <-cs.stopCh:
			return
		default:
		}
	}
}

func (cs *ClientSet) syncOnce() error {
	if cs.serverCount != 0 && cs.ClientsCount() >= cs.serverCount {
		return nil
	}
	c, serverCount, err := cs.newAgentClient()
	if err != nil {
		return err
	}
	if cs.serverCount != 0 && cs.serverCount != serverCount {
		klog.V(2).InfoS("Server count change suggestion by server",
			"current", cs.serverCount, "serverID", c.serverID, "actual", serverCount)

	}
	cs.serverCount = serverCount
	if err := cs.AddClient(c.serverID, c); err != nil {
		klog.ErrorS(err, "closing connection failure when adding a client")
		c.Close()
		return nil
	}
	klog.V(2).InfoS("sync added client connecting to proxy server", "serverID", c.serverID)
	go c.Serve()
	return nil
}

func (cs *ClientSet) Serve() {
	go cs.sync()
}

func (cs *ClientSet) shutdown() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for serverID, client := range cs.clients {
		client.Close()
		delete(cs.clients, serverID)
	}
}
