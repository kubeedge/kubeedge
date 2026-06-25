---
title: EdgeTunnelIP Address Type for iptables-Free API Server Redirection
status: implementable
authors:
  - "@tushar743-ui"
approvers:
  - "@luomengY"
  - "@Shelley-BaoYue"
creation-date: 2026-05-08
last-updated: 2026-06-25
---

* [Abstract](#1-abstract)
* [Motivation](#2-motivation)
* [Goals](#3-goals)
* [Non-Goals](#4-non-goals)
* [Problem Statement](#5-problem-statement)
* [Proposed Solution](#6-proposed-solution)
  * [Overview](#61-overview)
  * [Component Design](#62-component-design)
  * [Request Flow](#63-request-flow)
  * [Configuration](#64-configuration)
  * [High Availability CloudCore Compatibility](#65-high-availability-cloudcore-compatibility)
  * [Risk Assessment](#66-risk-assessment)
* [Alternatives Considered](#7-alternatives-considered)
* [Test Plan](#8-test-plan)

# EdgeTunnelIP Address Type for iptables-Free API Server Redirection

## 1. Abstract

Currently `cloudstream` relies on `iptableManager` to intercept API server
kubelet requests via DNAT rules. This requires root-level network privileges
and fails on eBPF-based CNIs like Cilium. All previous alternative approaches
either modified `InternalIP` creating a pseudo-IP visible in
`kubectl get nodes -o wide`, or required kube-apiserver flag changes that
are not possible on managed Kubernetes distributions.

This proposal introduces a new `EdgeTunnelIP` node address type. When an
edge node connects to CloudCore, CloudCore appends an `EdgeTunnelIP` address
containing the CloudCore IP to the node's status addresses. Operators
configure kube-apiserver with
`--kubelet-preferred-address-types=EdgeTunnelIP,InternalIP,ExternalIP,Hostname`.
The API server finds `EdgeTunnelIP` first and routes directly to CloudCore.
`InternalIP` is completely untouched — the real edge node IP is always
visible in `kubectl get nodes -o wide` with no pseudo-IP anywhere.

`NodeAddressType` in Kubernetes is a plain string type explicitly designed
for extension by cloud providers. No upstream Kubernetes changes are required.

## 2. Motivation

`kubectl logs`, `kubectl exec`, and `kubectl top` on edge pods require the
API server to route streaming requests through CloudCore's `cloudstream`
component. All previous approaches to eliminate the iptables dependency
required modifying `InternalIP` to cloudCoreIP, creating a pseudo-IP that
misleads operators and tooling. The `EdgeTunnelIP` approach solves the routing
problem without touching `InternalIP`, preserving the semantic correctness
of all standard node address fields.

## 3. Goals

- Introduce `EdgeTunnelIP` custom node address type carrying cloudCoreIP,
  enabling the API server to route kubelet requests directly to CloudCore
- Keep `InternalIP` completely untouched — real edge node IP always visible
- Work in all deployment topologies: external DaemonSet mode, internal mode
  co-located, internal mode separated nodes, Cilium environments
- Gate the feature behind `FeatureGates` in cloudcore config for safe rollout
- Preserve full backward compatibility — existing deployments unchanged
- Zero additional Kubernetes objects per edge node
- Make iptableManager fail gracefully instead of calling `os.Exit`

## 4. Non-Goals

- Modifying the Kubernetes upstream NodeAddressType enum (not required,
  NodeAddressType is a plain extensible string type)
- Replacing the WebSocket tunnel between cloudcore and edgecore
- Modifying edgestream or edged components
- Implementing eBPF-based packet steering (future direction)

## 5. Problem Statement

### 5.1 Current Architecture

When a user runs `kubectl logs` or `kubectl exec` on an edge pod, the API
server calls `GetPreferredNodeAddress` which iterates through
`--kubelet-preferred-address-types` in order and returns the address of the
first matching type found on the node. It does NOT fall back on connection
failure — it simply finds the first matching type and uses that address.

The default order is `InternalIP,ExternalIP,Hostname`. Every edge node has
`InternalIP` set to its real IP address. The API server always uses the real
edge node IP, which is unreachable directly from the cloud. iptableManager
intercepts this traffic via DNAT.

### 5.2 Problems with iptableManager

**Root-level network privileges required:**
iptables manipulation requires `NET_ADMIN`/`NET_RAW` capabilities.

**Breaks on eBPF-based CNIs:**
Cilium bypasses the iptables stack. DNAT rules never apply.

**Internal mode + separated nodes:**
In internal mode, DNAT rules only exist on cloudcore's node. If cloudcore
and the API server run on different nodes, the redirect never fires.

**Hard process exit on failure:**
`os.Exit(1)` kills cloudcore when iptables manipulation fails.

### 5.3 Why Previous Approaches Failed

All previous alternatives modified `InternalIP` to carry cloudCoreIP. This
was rejected by the community because `InternalIP` should carry the real edge
node IP. The `EdgeTunnelIP` approach is the first design that routes correctly
without touching `InternalIP`.

### 5.4 The EdgeTunnelIP Insight

`NodeAddressType` in Kubernetes is defined as:

```go
type NodeAddressType string
// These are built-in addresses type of node.
// A cloud provider may set a type not listed here.
```

It is an extensible plain string type with no validation restricting it to
built-in values. KubeEdge can introduce `EdgeTunnelIP` as its own constant
without any upstream Kubernetes changes.

`GetPreferredNodeAddress` iterates address types in the configured order:

```go
func GetPreferredNodeAddress(node *v1.Node,
    preferredAddressTypes []v1.NodeAddressType) (string, error) {
    for _, addressType := range preferredAddressTypes {
        for _, address := range node.Status.Addresses {
            if address.Type == addressType {
                return address.Address, nil
            }
        }
    }
    return "", &NoMatchError{addresses: node.Status.Addresses}
}
```

If `EdgeTunnelIP` appears first in `--kubelet-preferred-address-types` and
the node has an `EdgeTunnelIP` address, the API server uses that address.
`InternalIP` is never consulted. No DNAT interception needed.

## 6. Proposed Solution

### 6.1 Overview

![edgetunnelip-apiserver-redirection](../../images/proposals/edgetunnelip-apiserver-redirection-to-cloudcore.png)

When an edge node connects to CloudCore:

1. CloudCore appends `EdgeTunnelIP = cloudCoreIP` to the node's
   `status.addresses`. `InternalIP` is never touched.
2. `KubeletEndpoint.Port` is set to `streamPort` (10003).
3. `upstream.go` is updated to preserve `EdgeTunnelIP` addresses during
   edgecore heartbeat status overwrites, mirroring the existing
   `KubeletEndpoint.Port` preservation pattern.
4. When the edge node disconnects, CloudCore removes the `EdgeTunnelIP`
   address from node status.
5. The feature is gated behind `FeatureGates["EdgeTunnelIP"]` in cloudcore
   config for safe rollout.
6. `iptableManager` `os.Exit(1)` is replaced with graceful error log and
   return.

With these changes the full request path becomes:
kubectl logs/exec

→ API Server calls GetPreferredNodeAddress

→ Finds EdgeTunnelIP first (cloudCoreIP)

→ Sends directly to cloudCoreIP:10003

→ CloudStream receives request (no iptables)

→ Proxied through WebSocket tunnel

→ EdgeStream → edged

→ Response returns via same path

`InternalIP` carries real edge node IP throughout. No pseudo-IP.

### 6.2 Component Design

#### 1. EdgeTunnelIP Constant

Define in `common/constants/default.go`:

```go
// NodeEdgeTunnelIP is a custom node address type used by KubeEdge to
// carry the CloudCore IP for API server kubelet request routing.
// NodeAddressType is an extensible string type in Kubernetes — cloud
// providers may define custom types not listed in core/v1 constants.
const NodeEdgeTunnelIP corev1.NodeAddressType = "EdgeTunnelIP"
```

#### 2. updateNodeEdgeTunnelIP (new in tunnelserver.go)

Called from `connect()` when the `EdgeTunnelIP` feature gate is enabled.
Appends `EdgeTunnelIP = cloudCoreIP` to `node.Status.Addresses`. Reads
`constants.EdgeMappingCloudKey` annotation first as authoritative CloudCore
IP, falls back to `s.cloudCoreIP`. Checks whether `EdgeTunnelIP` already
exists with the correct value before updating to avoid unnecessary API calls.
Uses `wait.PollUntilContextTimeout` to retry.

```go
func (s *TunnelServer) updateNodeEdgeTunnelIP(nodeName string) error {
    // appends EdgeTunnelIP = cloudCoreIP to node.Status.Addresses
    // does not modify InternalIP or any other address type
}
```

#### 3. removeNodeEdgeTunnelIP (new in tunnelserver.go)

Called after `session.Serve()` returns in `connect()` — the natural
disconnect hook. Removes all `EdgeTunnelIP` addresses from the node's
`status.addresses`. Treats not-found as success.

```go
func (s *TunnelServer) removeNodeEdgeTunnelIP(nodeName string) error {
    // removes EdgeTunnelIP entries from node.Status.Addresses
    // does not touch InternalIP or any other address type
}
```

#### 4. updateNodeKubeletEndpoint (modified)

Sets `KubeletEndpoint.Port = s.streamPort` (10003) when EdgeTunnelIP
feature gate is enabled, so the API server uses the correct CloudStream
port after routing via `EdgeTunnelIP`. The upstream controller in
`upstream.go:694` already preserves this field from edgecore heartbeat
overwrites permanently.

#### 5. EdgeTunnelIP preservation in upstream.go

`upstream.go` overwrites the entire node status on every edgecore heartbeat:

```go
getNode.Status = nodeStatusRequest.Status
```

Since edgecore never sends `EdgeTunnelIP` in its address list, this removes
the address on every heartbeat. The fix mirrors the existing
`KubeletEndpoint.Port` preservation pattern at `upstream.go:694`:

```go
// Preserve EdgeTunnelIP addresses set by CloudCore.
// edgecore heartbeat does not include EdgeTunnelIP so it must
// be re-applied after status overwrite, following the same
// pattern as KubeletEndpoint.Port preservation above.
var edgeTunnelAddrs []corev1.NodeAddress
for _, addr := range getNode.Status.Addresses {
    if addr.Type == constants.NodeEdgeTunnelIP {
        edgeTunnelAddrs = append(edgeTunnelAddrs, addr)
    }
}
getNode.Status = nodeStatusRequest.Status
if len(edgeTunnelAddrs) > 0 {
    getNode.Status.Addresses = append(
        getNode.Status.Addresses, edgeTunnelAddrs...)
}
```

This is a 3-line surgical addition to `upstream.go` that is completely
self-contained and does not affect any other behavior.

#### 6. FeatureGate

Uses the existing `FeatureGates map[string]bool` in cloudcore v1alpha1
config types. When `FeatureGates["EdgeTunnelIP"] = true`, CloudCore
activates `updateNodeEdgeTunnelIP` and `updateNodeKubeletEndpoint` on
edge node connect, and `removeNodeEdgeTunnelIP` on disconnect.

When the feature gate is false (default), behavior is completely unchanged
from the current release — iptableManager runs as before.

#### 7. iptableManager graceful degradation

Replace `os.Exit(1)` in `iptables/iptables.go` with:

```go
klog.Errorf("iptables unavailable: %v", err)
return
```

CloudCore remains operational when iptables is unavailable.

### 6.3 Request Flow

![edgetunnelip-flow](../../images/proposals/KubeEdge-EdgeTunnelIP-Connection-and-Request-Routing-Flow.png)

### 6.4 Configuration

**Feature disabled (default, backward compatible):**

```yaml
modules:
  cloudStream:
    enable: true
featureGates:
  EdgeTunnelIP: false
```

**Feature enabled:**

```yaml
modules:
  cloudHub:
    advertiseAddress:
    - <cloudCoreIP>
  cloudStream:
    enable: true
    streamPort: 10003
featureGates:
  EdgeTunnelIP: true
```

Operator must also configure kube-apiserver:
--kubelet-preferred-address-types=EdgeTunnelIP,InternalIP,ExternalIP,Hostname

For kubeadm clusters this is set in `ClusterConfiguration`:

```yaml
apiServer:
  extraArgs:
    kubelet-preferred-address-types: "EdgeTunnelIP,InternalIP,ExternalIP,Hostname"
```

### 6.5 High Availability CloudCore Compatibility

In HA deployments multiple CloudCore instances run simultaneously. The
existing `cloudcore` annotation (`constants.EdgeMappingCloudKey`) set by
CloudHub `UpdateAnnotation` on every connection records which CloudCore
instance manages each edge node.

`updateNodeEdgeTunnelIP` reads this annotation as the authoritative CloudCore
IP for the `EdgeTunnelIP` address. In HA:

1. Edge node connects to CloudCore-1. `updateNodeEdgeTunnelIP` sets
   `EdgeTunnelIP = cloudCore1-IP`.

2. On failover, edge node reconnects to CloudCore-2. `UpdateAnnotation`
   updates annotation to cloudCore2-IP. CloudCore-2 calls
   `updateNodeEdgeTunnelIP` setting `EdgeTunnelIP = cloudCore2-IP`.
   The `upstream.go` preservation logic re-appends the updated value
   on subsequent heartbeats.

3. `EdgeTunnelIP` address type is unique — only one entry per node.
   `updateNodeEdgeTunnelIP` replaces any existing `EdgeTunnelIP` value
   rather than appending a duplicate.

### 6.6 Risk Assessment

**InternalIP completely untouched:**
`InternalIP` is never read, written, or reconciled by this proposal.
`kubectl get nodes -o wide` always shows the real edge node IP.

**Backward compatibility:**
Feature gate defaults to false. All existing deployments are completely
unaffected until operators explicitly enable the feature and update the
kube-apiserver flag.

**upstream.go preservation:**
The 3-line addition mirrors an existing proven pattern. It only acts on
addresses with type `EdgeTunnelIP` and is a no-op when the feature gate
is disabled (since no `EdgeTunnelIP` addresses will exist).

**Managed Kubernetes compatibility:**
Setting `--kubelet-preferred-address-types` requires kube-apiserver access.
This is a documented operator prerequisite. On managed Kubernetes (EKS, GKE,
AKS) this flag may not be configurable — document that the feature requires
kube-apiserver configuration access and is not available on all managed
offerings without additional support.

**os.Exit removal:**
Prevents cloudcore process death when iptables is unavailable in Cilium
environments where the feature gate has not been enabled yet.

## 7. Alternatives Considered

### 7.1 InternalIP = cloudCoreIP

Setting InternalIP directly to cloudCoreIP was implemented and tested.
Rejected by community: InternalIP should carry the real edge node IP.

### 7.2 ExternalIP = cloudCoreIP

Setting ExternalIP to cloudCoreIP requires `ExternalIP` to appear before
`InternalIP` in `--kubelet-preferred-address-types`. Since `GetPreferredNodeAddress`
picks the first matching type by presence not reachability, and InternalIP
always exists on edge nodes, ExternalIP is only used if listed first in the
flag. This requires the same kube-apiserver flag change as `EdgeTunnelIP`.
`EdgeTunnelIP` is semantically cleaner — it explicitly communicates its
purpose rather than overloading the existing `ExternalIP` field which has
different semantics in non-edge Kubernetes contexts.

### 7.3 Per-node Service and Endpoints

Creating a Kubernetes Service per edge node. Rejected: does not scale to
tens of thousands of edge nodes.

### 7.4 cloudcore Service ClusterIP

Setting InternalIP to cloudcore Service ClusterIP. Rejected: still a
pseudo-IP, InternalIP should not be modified.

### 7.5 eBPF-based packet steering

Rejected: kernel version constraints on edge devices, zero existing eBPF
infrastructure in KubeEdge. Future direction.

### 7.6 Tunnel port fix (KubeletEndpoint.Port = tunnelPort)

Setting KubeletEndpoint.Port to the negotiated tunnel port so iptableManager
DNAT fires correctly. Works in external DaemonSet mode but fails in internal
mode with separated cloudcore and API server nodes. Does not work on Cilium.

## 8. Test Plan

### Unit Tests

- `TestUpdateNodeEdgeTunnelIP` — appends correctly, already correct skips
  update, reads annotation IP, falls back to s.cloudCoreIP, nil client,
  node not found
- `TestRemoveNodeEdgeTunnelIP` — removes EdgeTunnelIP only, leaves
  InternalIP and other types untouched, not found is success, nil client
- `TestUpdateNodeKubeletEndpoint` — port = streamPort when feature gate
  enabled
- `TestEdgeTunnelIPPreservation` — simulates upstream.go heartbeat overwrite,
  verifies EdgeTunnelIP is re-appended, InternalIP unchanged
- `TestFeatureGateDisabled` — when gate=false, connect() does not call
  updateNodeEdgeTunnelIP, node addresses unchanged

### Integration Tests

- `kubectl get nodes -o wide` INTERNAL-IP shows real edge node IP
- EdgeTunnelIP address present in node.Status.Addresses on connect
- EdgeTunnelIP address removed from node.Status.Addresses on disconnect
- EdgeTunnelIP survives 90 seconds of edgecore heartbeat cycles
- InternalIP unchanged throughout all heartbeat cycles
- `kubectl logs` and `kubectl exec` succeed when kube-apiserver configured
  with EdgeTunnelIP first in preferred address types
- Feature gate disabled: no EdgeTunnelIP address set, iptableManager
  runs unchanged
- HA failover: EdgeTunnelIP updates to new CloudCore IP on reconnect

### e2e Tests

- Deploy edge pod on kind cluster with EdgeTunnelIP feature gate enabled
- Confirm `kubectl get nodes -o wide` INTERNAL-IP = real edge node IP
- Confirm EdgeTunnelIP = cloudCoreIP in node.Status.Addresses
- Confirm `kubectl logs` and `kubectl exec` succeed
- Confirm EdgeTunnelIP removed after edge node disconnect
- Confirm no Service or Endpoints objects created
- Confirm cloudcore continues running when iptables fails (os.Exit removed)
