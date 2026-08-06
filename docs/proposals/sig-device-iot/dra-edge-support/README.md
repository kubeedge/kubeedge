# Proposal: Minimal DRA Support for Edge Nodes

## Summary

This proposal introduces a minimal design to enable Dynamic Resource Allocation (DRA) support on KubeEdge edge nodes. The goal is to validate whether edge nodes can participate in a basic DRA workflow with minimal changes to the existing architecture.

## Motivation

Dynamic Resource Allocation (DRA) is a key feature in Kubernetes for managing extended and vendor-specific resources such as GPUs and other hardware accelerators.

KubeEdge extends Kubernetes workloads to edge environments, but currently lacks a clear support path for DRA on edge nodes. Specifically, there is no explicit mechanism for handling resource claims, preparing devices, and exposing them to containers on the edge.

This proposal aims to explore a minimal and feasible approach to support DRA on edge nodes without introducing complex scheduling or allocation logic.

## Goals

- Enable minimal DRA workflow on edge nodes
- Reuse Kubernetes DRA allocation on the cloud side
- Introduce lightweight edge-side execution logic
- Validate end-to-end feasibility as a minimal viable implementation (MVP)

## Non-Goals

- Full DRA feature parity with Kubernetes
- Complex scheduling logic on edge nodes
- Shared or fractional resource allocation
- Advanced lifecycle management

## Proposal

The proposal follows a cloud-edge collaborative model:

1. Kubernetes handles DRA allocation on the cloud side
2. CloudCore synchronizes DRA-related resources to edge nodes
3. EdgeCore resolves resource claims referenced by Pods
4. Edge-side components prepare devices
5. Devices are injected into containers

## Design Details

### Cloud Side

CloudCore reuses Kubernetes DRA allocation results and synchronizes relevant resources:

- ResourceClaim
- ResourceClaimTemplate
- DeviceClass
- ResourceSlice

### Edge Side

#### Claim Resolution

EdgeCore parses Pod specifications and resolves `resourceClaims`.

#### Device Preparation

A lightweight adapter prepares devices before container startup.

#### Container Injection

Prepared devices are injected into containers via runtime integration (e.g., mount or device exposure).

## KubeEdge Integration

This proposal integrates with existing KubeEdge components as follows:

- CloudCore:
  - Extends resource synchronization to include DRA-related resources
  - Reuses existing edge resource sync mechanisms

- EdgeCore:
  - Extends Pod processing logic to handle `resourceClaims`
  - Coordinates device preparation before container startup

- Edged:
  - Works with container runtime to inject prepared devices into containers

This design minimizes changes to the current architecture and reuses existing data paths wherever possible.

## Alternatives

- Implement full DRA support in EdgeCore  
  → Too complex for initial phase

- Ignore DRA on edge nodes  
  → Limits KubeEdge capabilities in device scenarios

## Risks and Mitigations

- Edge runtime differences  
  → Keep implementation minimal

- Device compatibility issues  
  → Use adapter abstraction

- Incomplete workflow support  
  → Validate step-by-step

## Backward Compatibility

This proposal is backward compatible.

- Existing workloads without DRA requirements are not affected
- DRA-related logic is only triggered when `resourceClaims` are present in Pod specifications
- No changes are required for existing edge deployments

## Implementation Plan

### Phase 1

CloudCore: sync DRA-related resources

### Phase 2

EdgeCore: claim resolution support

### Phase 3

Edge adapter: device prepare/unprepare

### Phase 4

Container injection support

### Phase 5

End-to-end validation

## Validation Plan

- Validate DRA scheduling on cloud
- Verify resource synchronization
- Verify claim resolution on edge
- Verify device preparation
- Verify container access to devices

## SIG Scope

This proposal is primarily relevant to SIG Device IoT, as it focuses on device resource management and execution on edge nodes.

It may also involve collaboration with:

- SIG Node: for container runtime and device injection behavior
- SIG Architecture: for overall design alignment with Kubernetes DRA

## Conclusion

This proposal introduces a minimal and incremental approach to explore DRA support on edge nodes, focusing on feasibility validation and future extensibility.