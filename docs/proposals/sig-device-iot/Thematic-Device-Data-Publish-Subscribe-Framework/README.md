---
title: Thematic Device Data Publish/Subscribe Framework Based on KubeEdge  
status: implementable  
authors:  
  - “@Dirtybear-lam”
approvers:  
creation-date: 2025-07-10
last-updated: 2025-07-10
---

## Motivation

In industrial IoT scenarios, devices (sensors, cameras, PLCs, etc.) continuously generate heterogeneous data—temperature, vibration, images, energy metrics, and more. Downstream AI analytics (predictive maintenance, process optimization) and fine‑grained operations (fault alerting, energy monitoring) have divergent requirements:

- **High‑priority events** (device anomalies, faults) demand sub‑second delivery.  
- **Low‑priority attributes** (periodic metrics, status reports) can be batched with relaxed latency.  

A unified, topic‑based pub/sub layer that distinguishes priority, hides multi‑protocol complexity, and integrates natively with KubeEdge’s DeviceTwin is essential to deliver low‑latency, high‑reliability data streams and support dynamic scaling.

---

## Goals

1. **Hierarchical Topic Model**  
   Define topics like `sensor/temperature` or `camera/objectDetected`, and tag each message as Real‑Time (RT) or Batch (BT).

2. **Dynamic Subscription & Routing**  
   - **RT** messages: immediate push to subscribers.  
   - **BT** messages: buffered and delivered in batches (future iteration).  
   - Edge‑cloud collaborative routing to pre‑filter at the edge and aggregate in the cloud.

3. **Unified RESTful API**  
   Expose `/register`, `/discover`, `/publish`, and `/subscribe` endpoints so that edge nodes and applications interact via HTTP.

4. **Native KubeEdge Integration**  
   Leverage DeviceTwin for gRPC data ingestion and ZeroMQ for high‑throughput edge messaging.

---

## Implementation Details

### Project Structure
<pre>

├── datastub-producer
│   ├── producer.go          # Edge producer: ZeroMQ SUB → PUB, QoS tagging
│   ├── config.yaml          # Topics, QoS, DataManager URL, ports
│   ├── daemonset.yaml       # Kubernetes DaemonSet spec
│   └── Dockerfile           # Build image
├── datastub-server
│   ├── manager
│   │   ├── main.go          # DataManager HTTP server
│   │   ├── register.go      # POST /register (node registration + heartbeat)
│   │   └── discover.go      # GET /discover (list active nodes)
│   └── stub
│       ├── stub.go          # DataStub entrypoint: ZeroMQ + HTTP init
│       ├── consumer_client.go  # ConsumerClient: ZeroMQ SUB → HTTP push
│       ├── api.go           # /publish & /subscribe handlers
│       └── health.go        # /healthz endpoint
└── datastub-consumer
    └── example.go           # Sample app: discover nodes & subscribe via webhook
</pre>

![DataStub 架构图](docs/architecture.jpg)
![DataStub 流程图](docs/flowchart.jpg)

- **DataManager** (`datastub-server/manager`):  
  - Manages registration and heartbeat of each DataStub node.  
  - Serves `/discover` to return all active DataStub endpoints.

- **ProducerClient** (`datastub-producer/producer.go`):  
  - Launches a gRPC server (`PushDeviceData`) over a Unix socket to receive data from DeviceTwin/Mapper. 
  - On first encounter of a new `device:topic`, allocates a port via `getNewPort()`, creates and binds a ZeroMQ PUB socket, then registers `{id, address, topics}` with the DataManager via HTTP POST.  
  - For each incoming device data message, formats `topic + " " + payload` and calls `publisher.Send()`, instantly publishing over ZeroMQ PUB to all subscribers.  

- **ConsumerClient** (`datastub-server/stub/consumer_client.go`):  
  - Exposes a `POST /subscribe` endpoint; upon receiving `{ "topic": "xxx" }`, keeps the HTTP connection open.  
  - Queries the DataManager for producer addresses, then issues ZeroMQ SUB subscriptions; in a background goroutine, `Recv()` messages and pushes serialized JSON into a `chan []byte`. 
  - In `handleSubscribeRequest`, continuously reads from that channel and uses `w.Write(data)` plus `w.(http.Flusher).Flush()` to stream JSON chunks back to the client via HTTP chunked response.  

### Configuration Example (`config.yaml`)

```yaml
dataStub:
  httpPort: 8080
  dataManagerURL: "http://<manager-host>:8000"
  topics:
    - name: "sensor/temperature"
      qos: "RT"
    - name: "power/usage"
      qos: "BT"
  heartbeatIntervalSec: 30
  zmq:
    subEndpoint: "tcp://127.0.0.1:5557"
    pubEndpoint: "tcp://0.0.0.0:5556"

###DataFlow
Mapper → DeviceTwin (gRPC) → ProducerClient (ZeroMQ SUB) → [RT/BT routing]
                                               ↓
                                ZeroMQ PUB to peer DataStubs
                                               ↓
                                ConsumerClient (ZeroMQ SUB)
                                               ↓
                          HTTP POST → Application webhook
1. Startup & Registration
	Each DataStub pod reads config.yaml and calls POST /register on DataManager.
	Begins sending heartbeats.
2. Message Ingestion
	DeviceTwin pushes device data via ZeroMQ.
	ProducerClient subscribes (SUB) and receives all topic messages.
3. Priority Routing
	RT: immediately PUB to peers and enqueue for local consumers.
	BT: buffer in memory for periodic batch dispatch.
4. Subscription & Delivery
	Applications call GET /discover to get DataStub endpoints.
	Register their callback via POST /subscribe?topic=<topic>&callback=<url>.
	ConsumerClient subscribes (SUB) to the topic and forwards each message via HTTP POST.
###Plan
PoC 
	Implement DataManager /register & /discover
 	Wire up ProducerClient RT path end‑to‑end (< 100 ms latency)
 	Implement ConsumerClient webhook delivery
 Integration & Testing 
	Validate throughput ≥ 2 000 msgs/sec/node
 	Conduct fault‑injection (DataManager/DataStub restart)
 Kubernetes Deployment 
	Write DaemonSet & Service YAML, Helm chart
 	Integrate Prometheus/Grafana monitoring
 Enhancements 
	Add BT batch‑dispatch mechanism
 	Extend protocol support (Modbus, CoAP)
 	Harden security (TLS, RBAC)

