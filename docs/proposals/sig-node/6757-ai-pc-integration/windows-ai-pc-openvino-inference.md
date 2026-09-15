# Windows AI PC OpenVINO Inference with KubeEdge

This document records the midterm reference architecture and validation workflow
for running an OpenVINO inference service on a Windows AI PC managed through
KubeEdge. It contributes to
[issue #6757](https://github.com/kubeedge/kubeedge/issues/6757).

The design separates the KubeEdge-managed environment from the Windows-native
AI runtime:

- EdgeCore and containerized edge workloads run inside WSL2.
- OpenVINO Model Server (OVMS) runs natively on Windows.
- Models are delivered to a directory shared by WSL2 and Windows.
- An edge adapter reports OVMS status and handles lifecycle commands.
- KubeEdge Router transfers messages between cloud and edge endpoints.

This arrangement allows a Windows-native CPU, GPU, or NPU runtime to remain
available to applications managed through KubeEdge.

## Midterm Scope and Reproducibility

This stage documents the integration architecture, component boundaries,
configuration patterns, deployment sequence, and validation checklist. The
configuration fragments use placeholders and can be adapted to a specific
environment.

The document does not include a production-ready OVMS adapter, complete
deployment manifests, pinned image digests, or model artifact checksums.
Therefore, it describes a reproducible workflow but not an identical benchmark
run. A follow-up stage should version the adapter and manifests, pin all runtime
and model artifacts, and retain validation outputs.

This work does not modify KubeEdge core components.

## Problem Statement

On a Windows AI PC, the AI runtime and hardware drivers may need to run directly
on Windows, while EdgeCore and Kubernetes workloads run inside WSL2. This
creates two execution environments on the same device.

The WSL2 environment contains:

- EdgeCore;
- the container runtime;
- the edge inference application;
- the OVMS adapter.

The Windows environment contains:

- OpenVINO Runtime;
- OpenVINO Model Server;
- hardware drivers;
- the model repository.

KubeEdge manages the workloads in WSL2, but the inference service remains a
Windows-native service. The integration coordinates these environments to
support:

- edge-node management;
- model delivery;
- inference application deployment;
- service and model-version status reporting;
- remote service start and stop operations.

## Architecture

```text
Cloud Kubernetes cluster
  KubeEdge CloudCore
  CloudCore Router
  RuleEndpoint and Rule resources
  Cloud status and control endpoint
              |
              | KubeEdge cloud-edge channel
              |
Windows AI PC
  WSL2
    EdgeCore
    Container runtime
    Inference application
    OVMS adapter
  Windows host
    OpenVINO Runtime
    OpenVINO Model Server
    Model repository
    CPU / GPU / NPU
```

### KubeEdge Responsibilities

KubeEdge is responsible for:

1. Registering the WSL2 environment as an edge node.
2. Scheduling the model-delivery workload to that node.
3. Deploying the inference application and OVMS adapter.
4. Routing OVMS status messages from edge to cloud.
5. Routing lifecycle commands from cloud to edge.

Installing OVMS and configuring Windows hardware drivers remain outside
KubeEdge's responsibilities.

## Model Delivery

The model artifact is packaged in a container image and scheduled to the target
edge node. The workload copies the model into a directory shared by both
environments:

```text
Model container image
        |
        v
KubeEdge edge workload
        |
        v
WSL2: /mnt/c/models
        |
        v
Windows: C:\models
        |
        v
OpenVINO Model Server
```

Example volume and mount:

```yaml
volumes:
  - name: model-repository
    hostPath:
      path: /mnt/c/models
      type: Directory

containers:
  - name: model-copy
    image: <CONTAINER_REGISTRY>/<MODEL_IMAGE>:<MODEL_VERSION>
    volumeMounts:
      - name: model-repository
        mountPath: /model-repository

nodeSelector:
  kubernetes.io/hostname: <EDGE_NODE_NAME>
```

The image reference and node name must be supplied by the deployment
environment. No private registry or production node name is included here.

## Edge Inference Application

The inference application runs as a containerized workload in WSL2:

```text
Client
  |
  v
Edge inference application
  |
  v
OpenVINO Model Server HTTP API
  |
  v
CPU / GPU / NPU
```

The OVMS endpoint and model name are configurable:

```yaml
env:
  - name: OVMS_ENDPOINT
    value: "http://<OVMS_HOST>:<OVMS_PORT>"
  - name: MODEL_NAME
    value: "<MODEL_NAME>"
```

The application must not contain a fixed host address or model name.

## OVMS Adapter

The OVMS adapter connects the KubeEdge-managed WSL2 environment to the
Windows-native inference service. It is responsible for:

- checking whether the OVMS Windows service is running;
- querying the models loaded by OVMS;
- reading the configured inference device;
- reporting loaded model versions;
- publishing service status to the edge EventBus;
- receiving lifecycle commands from the edge EventBus;
- starting or stopping the OVMS Windows service.

The adapter is a reference integration component. It is not intended to replace
the KubeEdge Mapper Framework.

## Status Reporting

The adapter periodically publishes an OVMS status message:

```json
{
  "device_name": "NPU",
  "model_name": "yolo",
  "loaded_versions": [
    "1"
  ],
  "service_status": "running",
  "timestamp": "2026-01-01T10:00:00Z"
}
```

The status flow is:

```text
OpenVINO Model Server
        |
        v
OVMS adapter
        |
        v
Edge EventBus
        |
        v
KubeEdge Rule
        |
        v
Cloud REST endpoint
```

An example EventBus topic is `device/ovms/status`. The node name and topic
prefix should remain configurable.

## Lifecycle Control

The cloud sends a validated lifecycle command to the Windows-native inference
service:

```json
{
  "action": "start"
}
```

```json
{
  "action": "stop"
}
```

The command flow is:

```text
Cloud control endpoint
        |
        v
CloudCore Router
        |
        v
KubeEdge Rule
        |
        v
Edge EventBus
        |
        v
OVMS adapter
        |
        v
Windows Service Manager
```

An example command topic is `device/ovms/cmd`. The adapter must reject
unsupported actions.

## KubeEdge Message Routing

The integration uses:

- a REST RuleEndpoint in the cloud;
- an EventBus RuleEndpoint at the edge;
- a cloud-to-edge Rule for lifecycle commands;
- an edge-to-cloud Rule for status messages.

Cloud-to-edge command route:

```text
Cloud REST RuleEndpoint
        |
        v
Cloud-to-edge Rule
        |
        v
Edge EventBus RuleEndpoint
        |
        v
device/ovms/cmd
```

Edge-to-cloud status route:

```text
device/ovms/status
        |
        v
Edge EventBus RuleEndpoint
        |
        v
Edge-to-cloud Rule
        |
        v
Cloud REST RuleEndpoint
```

Endpoint addresses must use environment-specific configuration:

```yaml
targetResource:
  resource: http://<CLOUD_STATUS_SERVICE>/<STATUS_PATH>
```

## Configuration

Non-sensitive values should be supplied through environment variables or
ConfigMaps:

```yaml
env:
  - name: WINDOWS_HOST
    value: "<WINDOWS_HOST>"
  - name: MQTT_BROKER
    value: "127.0.0.1"
  - name: MQTT_PORT
    value: "1883"
  - name: STATUS_TOPIC
    value: "device/ovms/status"
  - name: COMMAND_TOPIC
    value: "device/ovms/cmd"
```

Windows credentials must be stored in a Kubernetes Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: windows-access
type: Opaque
stringData:
  username: "<WINDOWS_USERNAME>"
  password: "<WINDOWS_PASSWORD>"
```

The adapter Deployment references that Secret:

```yaml
env:
  - name: WINDOWS_USER
    valueFrom:
      secretKeyRef:
        name: windows-access
        key: username
  - name: WINDOWS_PASSWORD
    valueFrom:
      secretKeyRef:
        name: windows-access
        key: password
```

Real credentials must not be committed to the repository.

## Prerequisites

The workflow requires:

- a Kubernetes cluster with KubeEdge CloudCore;
- a Windows AI PC with WSL2;
- EdgeCore and a container runtime running inside WSL2;
- OpenVINO Model Server installed on Windows;
- a model repository accessible from both WSL2 and Windows;
- network connectivity between CloudCore and EdgeCore.

Example shared paths:

```text
Windows: C:\models
WSL2:    /mnt/c/models
```

The exact paths may be changed for the deployment environment.

## Deployment Workflow

1. Deploy KubeEdge CloudCore and enable CloudCore Router.
2. Install and configure EdgeCore inside WSL2.
3. Join the WSL2 edge node to the KubeEdge cluster.
4. Install OpenVINO Model Server on Windows.
5. Prepare the shared model repository.
6. Deploy the RuleEndpoint and Rule resources.
7. Deploy the model-delivery workload.
8. Deploy the edge inference application.
9. Deploy the OVMS adapter.
10. Verify model loading, status reporting, lifecycle control, and inference.

## Validation Workflow

### Edge Node

```bash
kubectl get nodes
```

The target edge node should report `Ready`.

### Model Delivery

Check the shared model repository from WSL2:

```bash
ls -la /mnt/c/models
```

Confirm that the same model files are visible under `C:\models` on Windows.

### Inference Service

Query the OVMS model endpoint:

```bash
curl http://<OVMS_HOST>:<OVMS_PORT>/v1/models/<MODEL_NAME>
```

The response should indicate that the model is available.

### Edge Application

Submit an inference request to the edge application and confirm that it receives
a response from OVMS.

### Status Reporting

Confirm that the cloud status endpoint receives messages containing:

- service status;
- model name;
- loaded model version;
- inference device;
- timestamp.

### Lifecycle Control

Send `start` and `stop` actions through the cloud control endpoint. Confirm that
the OVMS Windows service changes to the requested state.

## Security Considerations

This example must not include:

- real public IP addresses;
- internal registry addresses;
- production endpoints;
- usernames or passwords;
- internal node names or namespaces;
- access tokens.

For production deployments:

- use Kubernetes Secrets for credentials;
- restrict Windows remote-management access to trusted networks;
- use encrypted transport and a limited Windows service account;
- protect cloud control APIs with authentication and authorization;
- restrict CloudCore Router access with firewall rules;
- validate every lifecycle command;
- avoid exposing OVMS directly to the public network.

Unencrypted WinRM and unrestricted Basic authentication are not recommended for
production environments.

## Limitations and Next Steps

This reference integration does not provide:

- automatic WSL2 or OpenVINO Model Server installation;
- Windows driver installation;
- production identity management;
- a general-purpose Windows management framework;
- vendor-specific model optimization;
- replacement functionality for the KubeEdge Mapper Framework.

The next stage should:

1. Add the versioned OVMS adapter and deployment manifests.
2. Pin container image digests and model artifacts.
3. Record model checksums, runtime versions, and hardware-driver versions.
4. Retain representative status, lifecycle, and inference outputs.
5. Validate reconnect behavior and long-running service health.
6. Document CPU, GPU, or NPU utilization and memory usage.

The main purpose of this work is to demonstrate how KubeEdge can coordinate
WSL2 workloads and a Windows-native AI inference runtime on the same edge
device.
