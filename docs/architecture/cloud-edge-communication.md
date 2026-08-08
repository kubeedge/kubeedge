# Cloud-Edge Communication: CloudHub, EdgeHub, and Stream

This document explains the roles, data flows, and port relationships of the
components responsible for communication between the KubeEdge cloud and edge
nodes.

## 1. Component Responsibilities

| Component | Side | Transport | Responsibility | Failure Symptom |
|-----------|------|-----------|---------------|-----------------|
| **CloudHub** | Cloud | WebSocket (:10000), QUIC (:10001), HTTPS (:10002) | Listens for edge node connections. The WebSocket server is the primary bidirectional channel for all control-plane messages (pod/node/resource sync, device twin updates). The HTTPS server handles certificate issuance, rotation, node health checks, and upgrade triggers. | Edge nodes cannot join. Pod/node status stops syncing. Certificates expire without renewal. |
| **EdgeHub** | Edge | WebSocket / QUIC client | Connects to CloudHub. Routes downstream messages from cloud to local edge modules (MetaManager, DeviceTwin, EventBus). Routes upstream messages from edge modules back to cloud. Maintains keepalive heartbeats and handles TLS certificate auto-rotation. | Edge node disconnects from cloud. Pods continue running locally (edge autonomy) but status changes and new deployments stop syncing. |
| **CloudStream** | Cloud | HTTPS (:10003 stream), WSS (:10004 tunnel) | Provides the data-plane tunnel for streaming operations. The **Stream Server** (:10003) accepts connections from kube-apiserver for `kubectl exec`, `kubectl logs`, `kubectl attach`, and metrics. The **Tunnel Server** (:10004) accepts persistent WebSocket connections from edge nodes, multiplexing multiple API server requests over each edge session. | `kubectl exec/logs/attach` fail for edge pods. `kubectl top` shows no data for edge pods. |
| **EdgeStream** | Edge | WSS client | Connects to CloudStream Tunnel Server (:10004). Receives streaming requests (exec, logs, attach, metrics) from cloud and forwards them to the local container runtime via edged. | Same as CloudStream failures — streaming operations fail for pods on this edge node. |

## 2. Port to Purpose Map

| Port | Listener | Protocol | TLS | Direction | Default | Purpose |
|------|----------|----------|-----|-----------|---------|---------|
| 10000 | CloudHub WebSocket | WSS | mTLS | Edge → Cloud | Enabled | Bidirectional control-plane message sync (pods, nodes, configmaps, secrets, device twins) |
| 10001 | CloudHub QUIC | QUIC | TLS 1.3 | Edge → Cloud | Disabled | Alternative to WebSocket for control-plane sync. Use when WebSocket is blocked by intermediate proxies. |
| 10002 | CloudHub HTTPS | HTTPS | mTLS | Edge → Cloud | Enabled | Certificate lifecycle: issuing (`GET /edge.crt`), CA retrieval (`GET /ca.crt`), node health check (`GET /node/{nodename}`), upgrade triggers (`POST /nodeupgrade`) |
| 10003 | CloudStream Stream | HTTPS | mTLS | kube-apiserver → Cloud | Disabled (*) | Accepts `kubectl exec/logs/attach/metrics` requests from kube-apiserver and tunnels them to the edge node |
| 10004 | CloudStream Tunnel | WSS | mTLS | Edge → Cloud | Disabled (*) | Persistent WebSocket tunnel from each edge node for multiplexed data-plane stream proxying |
| 10350 | — | TCP | — | internal | Enabled | Base port for tunnel port negotiation. Starting point from which the negotiated tunnel port is derived. |

> (*) In the default Go config, the CloudStream module is disabled by default
> (`Enable: false`), though `StreamPort` and `TunnelPort` still have default
> values defined. The Helm chart deployment enables CloudStream in
> `manifests/profiles/version.yaml`.

**When deployed via Helm**, the Kubernetes Service exposes these NodePorts:

| Service | NodePort | Maps To |
|---------|----------|---------|
| cloudhub NodePort | 30000 | CloudHub WebSocket :10000 |
| cloudhub QUIC NodePort | 30001 | CloudHub QUIC :10001 |
| cloudhub HTTPS NodePort | 30002 | CloudHub HTTPS :10002 |
| cloudstream NodePort | 30003 | CloudStream Stream :10003 |
| tunnel NodePort | 30004 | CloudStream Tunnel :10004 |

**Edge-side port relationships** (includes both local listeners and outbound destination ports):

| Port | Component | Type | Purpose |
|------|-----------|------|---------|
| 10550 | MetaServer | Local listener | Edge-side Kubernetes API proxy served over HTTPS |
| 9060 | ServiceBus | Local listener | HTTP proxy for edge-to-cloud REST calls |
| 1884 | EventBus | Local listener | Internal/embedded MQTT broker |
| 1883 | EventBus | Outbound destination | MQTT broker port used when EventBus connects to an external broker; may be off-node |

## 3. Three Communication Paths

### Path A — Control Plane Sync (mandatory)

```
  EdgeHub ──── WSS ────────→ CloudHub :10000
     ↕ (bidirectional)           ↕
  Edge modules                Cloud controllers
  (MetaManager,               (EdgeController,
   DeviceTwin,                 DeviceController,
   EventBus)                   SyncController, etc.)
```

**What flows here:** Pod specs (create/update/delete), node status updates,
configmaps, secrets, device twin properties, MQTT device events, rule engine
messages.

**What happens if blocked:** Edge node shows as `NotReady`. No new pods
scheduled. Existing pods keep running (edge autonomy). Device twin data stops
syncing.

### Path B — Certificate Lifecycle (mandatory)

```
  EdgeHub ──── HTTPS ────────→ CloudHub :10002
     GET /edge.crt              Cert issuance & renewal
     GET /ca.crt                CA certificate download
     GET /node/{nodename}           Node health check
     POST /nodeupgrade          Upgrade trigger
```

**What flows here:** Initial certificate issuance when an edge node joins,
automatic certificate rotation before expiry, node liveness verification, and
node upgrade task commands.

**What happens if blocked:** New edge nodes cannot join. Existing edge nodes
fail to rotate certificates (connection fails when cert expires). Upgrade tasks
cannot be dispatched.

### Path C — Data Plane Streaming (optional, for `kubectl exec/logs/attach`)

```
  kube-apiserver
       │
       │  kubectl exec/logs/attach
       ▼
  CloudStream Stream :10003
       │
       │  (internal bridge via TunnelServer)
       ▼
  CloudStream Tunnel :10004  ←── WSS ───  EdgeStream (edge node)
       │                                       │
       │                                       ▼
       │                                  Container Runtime
       │                                  (exec/logs/attach
       │                                   against target pod)
```

**What flows here:** Container log streams, exec stdin/stdout/stderr, attach
sessions, container metrics (cAdvisor, Prometheus probes, resource usage).

**What happens if blocked:** `kubectl exec`, `kubectl logs`, and `kubectl
attach` fail for pods on edge nodes. `kubectl top` shows no data for edge pods.

## 4. Data Plane Deep Dive

### How a kubectl exec/logs actually flows

When a user runs `kubectl exec my-pod` targeting a pod on an edge node:

1. **kube-apiserver** looks up the node's `KubeletEndpoint.Port` (set to the
   CloudStream TunnelPort by `updateNodeKubeletEndpoint()` in
   `cloud/pkg/cloudstream/tunnelserver.go:228`).

2. **iptables DNAT** on the cloud node (configured by the IptablesManager
   component, which runs alongside cloudcore, chain `TUNNEL-PORT` in
   `cloud/pkg/cloudstream/iptables/iptables.go:67`) redirects traffic destined
   for the negotiated tunnel port to the CloudStream Stream server at
   `{cloud-ip}:10003`.

3. **CloudStream Stream Server (:10003)** receives the request, identifies the
   target edge node from the URL path (namespace/pod/container), and looks up
   the active tunnel session for that node.

4. **CloudStream Tunnel Server (:10004)** forwards the request over the
   persistent WebSocket session established by EdgeStream.

5. **EdgeStream** on the edge node receives the forwarded request and hands it
   off to the local container runtime via edged.

6. Responses flow back through the same path in reverse.

### Tunnel Port Negotiation

At startup, CloudCore negotiates the TunnelPort using `10350`
(`constants.ServerPort`) as the base value, incrementing before checking
availability, so the first candidate is `10351`
(`cloud/cmd/cloudcore/app/server.go:271`). The negotiated port is then set on
the Node resource's `KubeletEndpoint.Port` so that kube-apiserver knows where
to send streaming requests.

### TLS Certificates

The Stream components use separate TLS certificates from CloudHub:

| Component | CA File | Cert File | Key File |
|-----------|---------|-----------|----------|
| CloudHub (all) | `rootCA.crt` | `server.crt` | `server.key` |
| CloudStream Tunnel | `rootCA.crt` (shared) | `server.crt` | `server.key` |
| CloudStream Stream | `streamCA.crt` | `stream.crt` | `stream.key` |

These are set via the CloudCore config fields `TLSTunnelCAFile`,
`TLSTunnelCertFile`, `TLSTunnelPrivateKeyFile`, `TLSStreamCAFile`,
`TLSStreamCertFile`, and `TLSStreamPrivateKeyFile`.

## 5. Deployment and Firewall Checklist

### Cloud Side — Ports to Expose

```
┌─────────────┬──────────┬──────────────────────────────────────────────┐
│ Port        │ Required │ Notes                                        │
├─────────────┼──────────┼──────────────────────────────────────────────┤
│ 10000       │ Yes      │ Must be reachable from all edge nodes         │
│ 10002       │ Yes      │ Must be reachable from all edge nodes         │
│ 10003       │ If using │ Must be reachable from the kube-apiserver     │
│             │ stream   │ (or wherever kubectl is proxied through)      │
│ 10004       │ If using │ Must be reachable from all edge nodes         │
│             │ stream   │                                              │
│ 10001       │ No       │ Only if you enable QUIC instead of WebSocket  │
│ 9443        │ No       │ Only if you enable the Router module          │
└─────────────┴──────────┴──────────────────────────────────────────────┘
```

### Edge Side — Outbound Destinations

Edge nodes are pure outbound clients; they do not need any inbound ports open:

```
  Edge node ──→ {cloud-ip}:10000  (WebSocket, control plane)
  Edge node ──→ {cloud-ip}:10002  (HTTPS, certs)
  Edge node ──→ {cloud-ip}:10004  (WebSocket, stream tunnel)
```

If using NodePorts (Helm deployment), replace cloud-ip with the Kubernetes node
IP and the ports with 30000, 30002, 30004.

### Edge Side — Local Ports

These are localhost only and do not need firewall rules:

- `10550` — MetaServer (edge-side API proxy)
- `1883` / `1884` — MQTT broker
- `9060` — ServiceBus HTTP proxy

## 6. Troubleshooting Map

| Symptom | Check Component | Check Port | Typical Cause |
|---------|----------------|------------|---------------|
| Edge node cannot join cluster | EdgeHub → CloudHub | :10002 | CloudHub HTTPS not reachable; token expired or wrong |
| Edge node joins but shows `NotReady` | EdgeHub → CloudHub | :10000 | CloudHub WebSocket blocked by firewall; TLS cert mismatch |
| Edge node joins but no pods schedule | EdgeController | — | Node labels missing; node is cordoned; resource mismatch |
| `kubectl logs` fails on edge pod | CloudStream | :10003, :10004 | CloudStream not enabled; IptablesManager not running; tunnel port mismatch |
| `kubectl exec` fails on edge pod | CloudStream | :10003, :10004 | Same as above; also check TLS certs for stream components |
| `kubectl top` shows no data for edge pod | CloudStream | :10003, :10004 | Metrics endpoint not routed through stream tunnel |
| Certificates expiring, not rotating | EdgeHub → CloudHub | :10002 | HTTPS server unreachable; clock skew between cloud and edge |
| Device data not syncing to cloud | DeviceTwin → EdgeHub → CloudHub | :10000 | DeviceController not running; MQTT broker issue on edge |
| Upgrade task stuck | EdgeHub → CloudHub | :10002 | TaskManager not running; node upgrade endpoint blocked |

### Logs to Check

| Component | Log Source | Command |
|-----------|-----------|---------|
| CloudHub | cloudcore process | `kubectl logs -n kubeedge deploy/cloudcore` |
| EdgeHub | edgecore process | `journalctl -u edgecore` or `tail -f /var/log/kubeedge/edgecore.log` |
| CloudStream | cloudcore process | Same as CloudHub — look for "tunnel" and "stream" messages |
| EdgeStream | edgecore process | Same as EdgeHub — look for "edgestream" or "tunnel" messages |
| IptablesManager | iptables-manager DaemonSet | `kubectl logs -n kubeedge ds/iptables-manager` |

---

## Configuration Reference

**CloudCore config** (`staging/src/github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1/types.go`):

```yaml
modules:
  cloudHub:
    websocket:
      enable: true
      port: 10000
    quic:
      enable: false
      port: 10001
    https:
      enable: true
      port: 10002
  cloudStream:
    enable: true
    tunnelPort: 10004
    streamPort: 10003
```

**EdgeCore config** (`staging/src/github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2/types.go`):

```yaml
modules:
  edgeHub:
    websocket:
      server: <cloud-ip>:10000
    httpserver: https://<cloud-ip>:10002
  edgeStream:
    enable: true
    tunnelServer: <cloud-ip>:10004
```

---

## 7. NAT, Firewalls, and Private Networks

KubeEdge is designed for edge nodes that sit behind NAT, firewalls, or private
networks without public IPs. No inbound port forwarding, no VPN, no STUN/TURN
is required.

### Why NAT is transparent

All communication is **outbound from edge to cloud**. The edge node initiates
every connection — it behaves like a web browser, not a server. CloudHub and
CloudStream never attempt to reach the edge node directly.

```
  Edge node (behind NAT)           Cloud (public / reachable IP)
  ┌──────────────────┐            ┌──────────────────────────────┐
  │  EdgeHub ── WSS ──┼──────────→│  :10000  CloudHub WebSocket   │
  │  EdgeHub ── HTTPS ─┼─────────→│  :10002  CloudHub HTTPS       │
  │  EdgeStream ─WSS ──┼─────────→│  :10004  CloudStream Tunnel   │
  └──────────────────┘            └──────────────────────────────┘
```

Because the edge opens the TCP connection, NAT gateways along the path
automatically create the return mapping. The cloud responds over the same
established connection. No special NAT traversal protocol is needed.

### What the edge node needs

The edge node must be able to make outbound TCP connections to the cloud's IP
on these ports:

| Port | Required | Protocol | Purpose |
|------|----------|----------|---------|
| 10000 | Yes | WSS (TLS) | Control-plane sync |
| 10002 | Yes | HTTPS (TLS) | Certificate lifecycle |
| 10004 | Only if using `kubectl exec/logs` | WSS (TLS) | Data-plane stream tunnel |
| 10001 | No | QUIC (UDP) | Alternative control-plane transport |

The cloud does **not** need to reach the edge node. No inbound ports, no port
forwarding, no public IP on the edge side.

### Proxy environments

If the edge node requires an HTTP/HTTPS proxy to reach the internet, set the
`HTTPS_PROXY` and `NO_PROXY` environment variables before starting edgecore.
The WebSocket connections (ports 10000, 10004) tunnel through the proxy via
HTTP CONNECT.

### Streaming through NAT

`kubectl exec`, `kubectl logs`, and `kubectl attach` work without direct
cloud-to-edge connectivity because:

1. EdgeStream establishes a **persistent outbound** WebSocket to CloudStream
   Tunnel (:10004).

2. When kube-apiserver needs to exec/log/attach into an edge pod, it sends the
   request to the CloudStream Stream Server (:10003) — which is on the cloud
   side, always reachable.

3. CloudStream multiplexes the request over the edge node's already-established
   tunnel WebSocket. The response flows back over the same connection.

No second inbound channel is ever opened from cloud to edge.

### Common deployment patterns

| Edge Location | Works? | Notes |
|---------------|--------|-------|
| Home/office behind consumer NAT | Yes | Outbound WebSocket passes through NAT like any HTTPS website. |
| Factory floor behind corporate firewall | Yes | If outbound port 10000/10002 is allowed. WebSocket over port 443 is also possible if the cloud listens on 443. |
| 4G/5G cellular (carrier-grade NAT) | Yes | Works the same as home NAT. May see more connection resets; EdgeHub auto-reconnects. |
| Behind an HTTP proxy | Yes | Requires proxy config. WebSocket tunnels through HTTP CONNECT. |
| Air-gapped network (no internet) | No (*) | Edge node must reach the cloud. If no path exists, a relay or satellite cloudcore instance is needed inside the private network. |
| Cloud in private VPC, edge on public internet | Yes | Reverse of typical setup. Edge needs outbound access to the cloud VPC. |

> (*) Air-gapped: KubeEdge requires edge-to-cloud outbound connectivity.
> If the edge network has no route to the cloud at all, consider running a
> secondary cloudcore instance inside the private network, or use EdgeSite
> standalone mode.

### Quick checklist

- Cloud must have ports 10000, 10002, and optionally 10004 reachable from edge nodes.
- Edge nodes need outbound internet (or outbound to cloud IP) on those ports.
- No inbound ports, public IP, or port forwarding needed on edge nodes.
- If `kubectl exec` fails but pod sync works, check that port 10004 is open on
  the cloud side and that EdgeStream is enabled on the edge node.
