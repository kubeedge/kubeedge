# Edged

## Overview

EdgeD is an edge node module which manages pod lifecycle. It helps users to deploy containerized workloads or applications at the edge node. Those workloads could perform any operation from simple telemetry data manipulation to analytics or ML inference and so on. Using `kubectl` command line interface at the cloud side, users can issue commands to launch the workloads.

Several OCI-compliant runtimes are supported through the Container Runtime Interface (CRI). See [KubeEdge runtime configuration](../../configuration/cri.md) for more information on how to configure edged to make use of other runtimes.

There are many modules which work in tandem to achieve edged's functionalities.

![EdgeD Overall](../../images/edged/edged-overall.png)

*Fig 1: EdgeD Functionalities*

## Pod Management

It is handles for pod addition, deletion and modification. It also tracks the health of the pods using pod status manager and pleg.
Its primary jobs are as follows:

- Receives and handles pod addition/deletion/modification messages from metamanager.
- Handles separate worker queues for pod addition and deletion.
- Handles worker routines to check worker queues to do pod operations.
- Keeps separate cache for config map and secrets respectively.
- Regular cleanup of orphaned pods

![Pod Addition Flow](../../images/edged/pod-addition-flow.png)

*Fig 2: Pod Addition Flow*

![Pod Deletion Flow](../../images/edged/pod-deletion-flow.png)

*Fig 3: Pod Deletion Flow*

![Pod Updation Flow](../../images/edged/pod-update-flow.png)

*Fig 4: Pod Updation Flow*

## Pod Lifecycle Event Generator

This module helps in monitoring pod status for edged. Every second, using probes for liveness and readiness, it updates the information with pod status manager for every pod.

![PLEG Design](../../images/edged/pleg-flow.png)

*Fig 5: PLEG at EdgeD*

## CRI for edged

Container Runtime Interface (CRI) – a plugin interface which enables edged to use a wide variety of container runtimes like Docker, containerd, CRI-O, etc., without the need to recompile. For more on how to configure KubeEdge for container runtimes, see [KubeEdge runtime configuration](../../configuration/cri.md).

#### Why CRI for edged?
CRI support for multiple container runtimes in edged is needed in order to:
+ Support light-weight container runtimes on resource-constrained edge nodes which are unable to run the existing Docker runtime.
+ Support multiple container runtimes like Docker, containerd, CRI-O, etc., on edge nodes.

Support for corresponding CNI with pause container and IP will be considered later.

![CRI Design](../../images/edged/edged-cri.png)

*Fig 6: CRI at EdgeD*

## Secret Management

In edged, Secrets are handled separately. For operations like addition, deletion and modification, there are separate sets of config messages and interfaces.
Using these interfaces, secrets are updated in cache store.
The flow diagram below explains the message flow.

![Secret Message Handling](../../images/edged/secret-handling.png)

*Fig 7: Secret Message Handling at EdgeD*

Edged uses the MetaClient module to fetch secrets from MetaManager. If edged queries for a new secret which is not yet stored in MetaManager, the request is forwarded to the Cloud. Before sending the response containing the secret, MetaManager stores it in a local database. Subsequent queries for the same secret key will be retrieved from the database, reducing latency. The flow diagram below shows how a secret is fetched from MetaManager and the Cloud. It also describes how the secret is stored in MetaManager.

![Query Secret](../../images/edged/query-secret-from-edged.png)

*Fig 8: Query Secret by EdgeD*

## Probe Management

Probe management creates two probes for readiness and liveness respectively for pods to monitor the containers. The readiness probe helps by monitoring when the pod has reached a running state. The liveness probe helps by monitoring the health of pods, indicating if they are up or down.
As explained earlier, the PLEG module uses its services.


## ConfigMap Management
In edged, ConfigMaps are also handled separately. For operations like addition, deletion and modification, there are separate sets of config messages and interfaces.
Using these interfaces, ConfigMaps are updated in cache store.
The flow diagram below explains the message flow.

![ConfigMap Message Handling](../../images/edged/configmap-handling.png)

*Fig 9: ConfigMap Message Handling at EdgeD*

Edged uses the MetaClient module to fetch ConfigMaps from MetaManager. If edged queries for a new ConfigMap which is not yet stored in MetaManager, the request is forwarded to the Cloud. Before sending the response containing the ConfigMap, MetaManager stores it in a local database. Subsequent queries for the same ConfigMap key will be retrieved from the database, reducing latency. The flow diagram below shows how ConfigMaps are fetched from MetaManager and the Cloud. It also describes how ConfigMaps are stored in MetaManager.

![Query Configmaps](../../images/edged/query-configmap-from-edged.png)

*Fig 10: Query Configmaps by EdgeD*

## Container GC

The container garbage collector is an edged routine which wakes up every minute, collecting and removing dead containers using the specified container gc policy.
The policy for garbage collecting containers is determined by three variables, which can be user-defined:
+ `MinAge` is the minimum age at which a container can be garbage collected, zero for no limit.
+ `MaxPerPodContainer` is the maximum number of dead containers that any single pod (UID, container name) pair is allowed to have, less than zero for no limit.
+ `MaxContainers` is the maximum number of total dead containers, less than zero for no limit. Generally, the oldest containers are removed first.

## Image GC

The image garbage collector is an edged routine which wakes up every 5 secs, and collects information about disk usage based on the policy used.
The policy for garbage collecting images takes two factors into consideration, `HighThresholdPercent` and `LowThresholdPercent`. Disk usage above the high threshold will trigger garbage collection, which attempts to delete unused images until the low threshold is met. Least recently used images are deleted first.

## Status Manager

Status manager is an independent edge routine, which collects pods statuses every 10 seconds and forwards this information to the cloud using the metaclient interface.

![Status Manager Flow](../../images/edged/pod-status-manger-flow.png)

*Fig 11: Status Manager Flow*

## Volume Management

Volume manager runs as an edge routine which brings out the information of which volume(s) are to be attached/mounted/unmounted/detached based on pods scheduled on the edge node.

Before starting the pod, all the specified volumes referenced in pod specs are attached and mounted, Till then the flow is blocked and with its other operations.

## MetaClient

Metaclient is an interface of Metamanger for edged. It helps edged to get ConfigMaps and secret details from metamanager or cloud.
It also sends sync messages, node status and pod status towards metamanger to cloud.
