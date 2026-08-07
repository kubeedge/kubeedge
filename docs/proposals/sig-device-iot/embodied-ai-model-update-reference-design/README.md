# Edge-Cloud Model Update Reference Design for Embodied AI Devices

## Motivation

Embodied AI devices such as mobile robots, inspection vehicles, edge cameras, and autonomous platforms usually run in changing physical environments.

A model that performs well in a controlled or well-lit environment may produce missed detections or low-confidence results under low light, occlusion, viewpoint changes, complex backgrounds, or other environmental changes.

Updating models on distributed edge devices manually is inefficient. It may require operators to collect data, retrain models, package new versions, log in to individual devices, and replace models one by one.

This document describes a reference design for continuously updating edge AI models with KubeEdge. The design separates business data transfer from the KubeEdge control path and focuses on edge autonomy, model rollout, version switching, and behavior under unstable networks.

## Target Scenario

The primary validation scenario is visual recognition on a mobile robot.

The robot runs an inference service with Model v1.0 on an edge node. Model v1.0 works in a standard or well-lit environment but may produce missed detections or low-confidence results in a low-light environment.

The edge side identifies valuable hard samples, caches them locally, and uploads them to the cloud when the network is available. The cloud side organizes data processing, model training, evaluation, and version management. After Model v2.0 passes evaluation, the new model or inference service is rolled out to selected edge nodes through KubeEdge.

Other embodied AI devices, such as humanoid robots, underwater robots, multi-camera robots, and edge cameras, are considered possible extension scenarios. The initial validation does not cover all robot or sensor types.

## Goals

The design has the following goals:

- Keep AI inference running locally on edge devices.
- Identify and cache valuable hard samples at the edge.
- Upload hard samples when network connectivity is available.
- Separate the business data path from the KubeEdge control path.
- Manage model and inference-service updates from the cloud.
- Roll out updates to selected edge nodes or node groups.
- Keep the previous usable model available during update failures.
- Continue edge inference during cloud-edge disconnection.
- Make model versions and rollout status observable.
- Define a validation plan for model rollout and edge autonomy.

## Non-Goals

The following items are outside the initial scope:

- KubeEdge does not perform model training.
- This design does not define a new AI training framework.
- Raw images and videos are not required to pass through the KubeEdge control channel.
- The initial validation does not implement all robot and sensor types.
- Fully automatic data labeling is not required.
- Multi-node rollout and advanced rollback policies are optional extensions.

## Overall Workflow

The proposed workflow contains the following steps:

1. A robot or edge device collects data from cameras or sensors.
2. Model v1.0 performs local inference on the edge node.
3. The edge side identifies missed detections, low-confidence results, or other hard samples.
4. Hard samples are cached locally.
5. When the network is available, hard samples are uploaded through a business data service.
6. The cloud side performs data cleaning and label confirmation.
7. The dataset is updated and used for training or fine-tuning.
8. Model v2.0 is evaluated and stored in a model repository.
9. The model file, inference image, or workload configuration is published through KubeEdge.
10. The edge side switches to Model v2.0 and continues local inference.
11. Model version and rollout status are reported to the cloud.

## Architecture

The architecture contains three logical layers.

### Device Side

The device side includes:

- mobile robots;
- cameras;
- sensors;
- other embodied AI devices.

The device side provides physical-world data and executes the corresponding perception or inspection task.

### Edge Side

The edge side includes:

- EdgeCore;
- EdgeHub;
- MetaManager;
- the local inference service;
- the model loader;
- local hard-sample storage.

EdgeCore manages edge workloads and supports local operation. EdgeHub provides cloud-edge communication. MetaManager maintains local metadata and helps the edge node continue running when the cloud connection is unavailable.

The inference service loads Model v1.0 or Model v2.0 and performs local inference. Hard samples are stored locally before being uploaded.

### Cloud Side

The cloud side includes:

- Kubernetes API Server;
- CloudCore;
- model training and evaluation services;
- a model repository;
- a business data upload service.

The training and evaluation services are not part of KubeEdge. They are responsible for data processing, model training, evaluation, and packaging.

CloudCore works with the Kubernetes control plane to manage edge nodes, synchronize resources, and deliver workload updates to edge nodes.

## Business Data Path

Images, videos, and other hard-sample data use a business data path.

The business data path is responsible for:

- receiving hard samples;
- transferring image or video data;
- retrying uploads after temporary network failures;
- storing sample metadata;
- associating samples with model versions and edge devices.

Large sample data should not be transferred through the KubeEdge control channel.

A hard-sample record may contain:

- device identifier;
- edge node identifier;
- model name;
- model version;
- timestamp;
- confidence score;
- failure reason;
- local file path or object storage location.

## KubeEdge Control Path

KubeEdge provides the control and management path.

The control path is responsible for:

- edge node management;
- workload declaration and rollout;
- model or inference-service version updates;
- target-node selection;
- workload status synchronization;
- edge autonomy during temporary disconnection.

Kubernetes Deployment or KubeEdge EdgeApplication may be used to deploy and update the inference service.

NodeGroup may be used to organize edge nodes and select rollout targets. NodeGroup is used for node grouping and target selection; it is not itself a workload.

## Edge-Side Design

The edge-side workflow contains the following components.

### Local Inference Service

The inference service:

- loads the current model version;
- receives camera or sensor input;
- performs local inference;
- returns inference results;
- records model version information.

### Hard-Sample Selection

Hard samples may be selected using rules such as:

- no detected object;
- confidence below a configured threshold;
- repeated unstable predictions;
- user-confirmed false positives or false negatives;
- environment metadata indicating low light or occlusion.

Rule-based selection is only an initial filter. Data confirmation and labeling may still require human review.

### Local Cache

The local cache stores hard samples when:

- the cloud is unreachable;
- upload bandwidth is limited;
- the upload service is temporarily unavailable.

The cache should record upload status and retry pending samples after connectivity is restored.

## Cloud-Side Design

The cloud-side workflow includes:

1. sample reception;
2. data cleaning;
3. label confirmation;
4. dataset update;
5. model training or fine-tuning;
6. model evaluation;
7. model packaging;
8. model version registration;
9. rollout request generation.

The cloud side should only publish a new model after it passes the configured evaluation criteria.

## Model Update Strategies

Two update strategies are considered.

### Model File Update

In this strategy, the inference service remains unchanged and only the model file is updated.

The workflow is:

1. Publish Model v2.0 to a model repository or object storage.
2. Notify the target edge node of the new model version.
3. Download the model to a temporary local path.
4. Verify the model checksum and metadata.
5. Load and validate the new model.
6. Switch the active model version.
7. Keep the previous version for rollback.

Advantages:

- smaller update size;
- faster model-only updates;
- no need to rebuild the inference image.

Considerations:

- the inference service must support model hot loading or controlled restart;
- compatibility between the model and runtime must be verified;
- rollback logic must be implemented by the application.

### Container Image Update

In this strategy, the model and inference service are packaged into a new container image.

The workflow is:

1. Build an inference image containing Model v2.0.
2. Update the Deployment or EdgeApplication specification.
3. Publish the workload update through KubeEdge.
4. Pull and start the new image on the edge node.
5. Verify workload readiness and model version.
6. Roll back to the previous image when the update fails.

Advantages:

- consistent runtime and model dependencies;
- clear image-based version management;
- easier integration with workload rollout mechanisms.

Considerations:

- larger update size;
- higher network and storage cost;
- image pull may be affected by unstable connectivity.

## Behavior Under Unstable Networks

The edge node should remain usable during cloud-edge disconnection.

Expected behavior includes:

- the current inference service continues running;
- the current usable model remains active;
- hard samples are cached locally;
- incomplete model downloads are not activated;
- workload and model status are synchronized after reconnection;
- cached samples are uploaded after connectivity is restored.

A new model must not replace the current model until download, integrity verification, loading, and basic validation have completed successfully.

## Version Switching and Rollback

The edge side should track at least:

- current model version;
- previous model version;
- desired model version;
- update status;
- update timestamp;
- failure reason.

A model update may use the following states:

- `Pending`
- `Downloading`
- `Verifying`
- `Loading`
- `Active`
- `Failed`
- `RolledBack`

If the new model cannot be downloaded, verified, loaded, or started, the edge side should continue using the previous usable version.

Rollback may be triggered by:

- checksum failure;
- model loading failure;
- inference-service startup failure;
- readiness timeout;
- explicit operator request.

## Validation Plan

### Core Validation

The initial validation should cover:

1. **Low-light recognition comparison**
   Compare Model v1.0 and Model v2.0 using the same low-light dataset.

2. **Model rollout success**
   Confirm that the target edge node receives the expected model or inference-service version.

3. **Version switching**
   Confirm that the inference service loads and runs the desired model version.

4. **Offline inference**
   Disconnect the cloud-edge network and confirm that local inference continues with the current usable model.

### Extended Validation

Optional validation may include:

- rollout latency;
- failed-update rollback;
- data upload after network recovery;
- status reconciliation after reconnection;
- rollout to multiple edge nodes;
- node-group-based rollout;
- model and image update comparison.

## Current Status

The current design work includes:

- target-scenario definition;
- cloud, edge, and device responsibility boundaries;
- hard-sample feedback workflow;
- business data path and control path separation;
- model file and container image update strategies;
- unstable-network behavior definition;
- validation planning.

Implementation details, deployment manifests, rollback mechanisms, and test results will be refined in follow-up work.

## Future Work

Future work includes:

- adding example Deployment and EdgeApplication manifests;
- defining model-version metadata;
- implementing model update status reporting;
- validating model-file and container-image update strategies;
- testing local caching and retry behavior;
- testing rollback after update failures;
- validating behavior during cloud-edge disconnection;
- documenting reproducible deployment and test steps;
- extending the design to additional embodied AI devices.