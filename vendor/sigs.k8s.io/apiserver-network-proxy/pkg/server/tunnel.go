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
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/konnectivity-client/proto/client"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server/metrics"
)

// Tunnel implements Proxy based on HTTP Connect, which tunnels the traffic to
// the agent registered in ProxyServer.
type Tunnel struct {
	Server *ProxyServer
}

func (t *Tunnel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metrics.Metrics.HTTPConnectionInc()
	defer metrics.Metrics.HTTPConnectionDec()

	klog.V(2).InfoS("Received request for host", "method", r.Method, "host", r.Host, "userAgent", r.UserAgent())
	if r.TLS != nil {
		klog.V(2).InfoS("TLS", "commonName", r.TLS.PeerCertificates[0].Subject.CommonName)
	}
	if r.Method != http.MethodConnect {
		http.Error(w, "this proxy only supports CONNECT passthrough", http.StatusMethodNotAllowed)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var closeOnce sync.Once
	defer closeOnce.Do(func() { conn.Close() })

	random := rand.Int63() /* #nosec G404 */
	dialRequest := &client.Packet{
		Type: client.PacketType_DIAL_REQ,
		Payload: &client.Packet_DialRequest{
			DialRequest: &client.DialRequest{
				Protocol: "tcp",
				Address:  r.Host,
				Random:   random,
			},
		},
	}

	klog.V(4).Infof("Set pending(rand=%d) to %v", random, w)
	backend, err := t.Server.getBackend(r.Host)
	if err != nil {
		http.Error(w, fmt.Sprintf("currently no tunnels available: %v", err), http.StatusInternalServerError)
		return
	}
	closed := make(chan struct{})
	connected := make(chan struct{})
	connection := &ProxyClientConnection{
		Mode: "http-connect",
		HTTP: io.ReadWriter(conn), // pass as ReadWriter so the caller must close with CloseHTTP
		CloseHTTP: func() error {
			closeOnce.Do(func() { conn.Close() })
			close(closed)
			return nil
		},
		connected: connected,
		start:     time.Now(),
		backend:   backend,
	}
	t.Server.PendingDial.Add(random, connection)
	if err := backend.Send(dialRequest); err != nil {
		klog.ErrorS(err, "failed to tunnel dial request")
		return
	}
	ctxt := backend.Context()
	if ctxt.Err() != nil {
		klog.ErrorS(err, "context reports failure")
	}

	select {
	case <-ctxt.Done():
		klog.V(5).Infoln("context reports done")
	default:
	}

	select {
	case <-connection.connected: // Waiting for response before we begin full communication.
	case <-closed: // Connection was closed before being established
	}

	defer func() {
		packet := &client.Packet{
			Type: client.PacketType_CLOSE_REQ,
			Payload: &client.Packet_CloseRequest{
				CloseRequest: &client.CloseRequest{
					ConnectID: connection.connectID,
				},
			},
		}

		if err = backend.Send(packet); err != nil {
			klog.V(2).InfoS("failed to send close request packet", "host", r.Host, "agentID", connection.agentID, "connectionID", connection.connectID)
		}
		conn.Close()
	}()

	klog.V(3).InfoS("Starting proxy to host", "host", r.Host)
	pkt := make([]byte, 1<<15) // Match GRPC Window size

	connID := connection.connectID
	agentID := connection.agentID
	var acc int

	for {
		n, err := bufrw.Read(pkt[:])
		acc += n
		if err == io.EOF {
			klog.V(1).InfoS("EOF from host", "host", r.Host)
			break
		}
		if err != nil {
			klog.ErrorS(err, "Received failure on connection")
			break
		}

		packet := &client.Packet{
			Type: client.PacketType_DATA,
			Payload: &client.Packet_Data{
				Data: &client.Data{
					ConnectID: connID,
					Data:      pkt[:n],
				},
			},
		}
		err = backend.Send(packet)
		if err != nil {
			klog.ErrorS(err, "error sending packet")
			break
		}
		klog.V(5).InfoS("Forwarding data on tunnel to agent",
			"bytes", n,
			"totalBytes", acc,
			"agentID", connection.agentID,
			"connectionID", connection.connectID)
	}

	klog.V(5).InfoS("Stopping transfer to host", "host", r.Host, "agentID", agentID, "connectionID", connID)
}
