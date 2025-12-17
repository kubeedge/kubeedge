* [v1.21.1](#v1211)
    * [Downloads for v1.21.1](#downloads-for-v1211)
    * [KubeEdge v1.21.1 Release Notes](#kubeedge-v1211-release-notes)
        * [Changelog since v1.21.0](#changelog-since-v1210)
* [v1.21.0](#v1210)
    * [Downloads for v1.21.0](#downloads-for-v1210)
    * [KubeEdge v1.21 Release Notes](#kubeedge-v121-release-notes)
        * [1.21 What's New](#121-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.21.1

## Downloads for v1.21.1

Download v1.21.1 in the [v1.21.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.21.1).

## KubeEdge v1.21.1 Release Notes

### Changelog since v1.21.0

- Fix incorrect runner register for NodeUpgradeJob. ([#6496](https://github.com/kubeedge/kubeedge/pull/6496), [@liuzhen21](https://github.com/liuzhen21))
- Fix policyManager reconciliation with non-cached reader. ([#6558](https://github.com/kubeedge/kubeedge/pull/6558), [@mkhon](https://github.com/mkhon))
- Fix Leases being parsed as Leas in MetaServer. ([#6567](https://github.com/kubeedge/kubeedge/pull/6567), [@brinker-tbaker](https://github.com/brinker-tbaker))
- Fix keadm reset edge failed to remove containers for containerRuntime.Connect missed. ([#6575](https://github.com/kubeedge/kubeedge/pull/6575), [@will4j](https://github.com/will4j))
- Fix data push operation in mapper-framework. ([#6578](https://github.com/kubeedge/kubeedge/pull/6578), [@aAAaqwq](https://github.com/aAAaqwq))

# v1.21.0

## Downloads for v1.21.0

Download v1.21.0 in the [v1.21.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.21.0).

## KubeEdge v1.21 Release Notes

## 1.21 What's New

### New Generation Node Task API and Implementation

In v1.21, we redesigned the status structure and operation process of node jobs to track error information and facilitate developers' understanding. In the new design, the node job status includes Phase (Init, InProgress, Completed, Failure) and nodeStatus. 
The nodeStatus consists of Phase (Pending, InProgress, Successful, Failure, Unknown), actionFlow, nodeName, reason, and business-related fields. 
A YAML example of the NodeUpgradeJob status is provided below.
```yaml
status:
  nodeStatus:
    - actionFlow:
        - action: Check
          status: 'True'
          time: '2025-05-28T08:12:01Z'
        - action: WaitingConfirmation
          status: 'True'
          time: '2025-05-28T08:12:01Z'
        - action: Backup
          status: 'True'
          time: '2025-05-28T08:12:01Z'
        - action: Upgrade
          status: 'True'
          time: '2025-05-28T08:13:02Z'
      currentVersion: v1.21.0
      historicVersion: v1.20.0
      nodeName: ubuntu
      phase: Successful
  phase: Completed
```

Refer to the link for more details.([#6082](https://github.com/kubeedge/kubeedge/pull/6082), [#6084](https://github.com/kubeedge/kubeedge/pull/6084))

### Support Closed Loop Flow Control

In v1.21, we have optimized the traffic closed-loop function of node groups. Applications within a node group can only access application services within the same group and unable to access services of other node groups. 
With this mechanism, users can easily achieve network isolation between multiple edge regions, ensuring that application services in different regions do not interfere with each other.

Refer to the link for more details.([#6097](https://github.com/kubeedge/kubeedge/pull/6097), [#6077](https://github.com/kubeedge/kubeedge/pull/6077))

### Support Update Edge Configuration from Cloud

In many cases, cloud-based direct updates to EdgeCore configuration files for edge nodes offer greater convenience than manual updates from edge node, especially for batch operations that boost efficiency by managing multiple nodes simultaneously.

In v1.21.0, `ConfigUpdateJob` CRD is introduced to allows users to update configuration files for edge nodes in the cloud. The `updateFields` within the CRD is used to specify the configuration items that need to be updated.

CRD Sample:

```yaml
apiVersion: operations.kubeedge.io/v1alpha2
kind: ConfigUpdateJob
metadata:
  name: configupdate-test
spec:
  failureTolerate: "0.3"
  concurrency: 1
  timeoutSeconds: 180
  updateFields:
    modules.edgeStream.enable: "true"
  labelSelector:
    matchLabels:
      "node-role.kubernetes.io/edge": ""
      node-role.kubernetes.io/agent: ""
```
**Note:**
* This feature is disabled by default in v1.21.0. To enable it, please start the ControllerManager and TaskManager at cloud, as well as the TaskManager edge.
* Updating edge configurations will require a restart of EdgeCore.


Refer to the link for more details.([#6338](https://github.com/kubeedge/kubeedge/pull/6338))

### Support One-Click Deployment of Dashboard and Integration of kubeedge/keink

In v1.21, the dashboard functionality has been enhanced by designing a BFF (Backend for Frontend) layer to connect the frontend user interface layer with the KubeEdge backend API. 
Additionally, the dashboard is integrated with the keink project, allowing users to launch a keink cluster in the dashboard environment with just one command to experience KubeEdge features.

Refer to the link for more details.([#50](https://github.com/kubeedge/dashboard/pull/50))

## Important Steps before Upgrading

- From v1.21, the v1alpha2 node job enables by default, and the CRD definition will be backward compatible. If you want to continue to use the v1alpha1 version of the NodeUpgradeJob and ImagePrePullJob, please setting the feature gates of ControllerManager and CloudCore. 
  - Add a command arg for ControllerManager:
    ```shell
    --feature-gates=disableNodeTaskV1alpha2
    ```
  - Modify the CloudCore configuration:
    ```yaml
    apiVersion: cloudcore.config.kubeedge.io/v1alpha2
    kind: CloudCore
    featureGates:
      disableNodeTaskV1alpha2: true
    ...
    ```

**Note:**
The node job v1alpha2 CRDs are compatible with v1alpha1, but they **cannot be switched** between them. The code logic of v1alpha1 will destroy the data of v1alpha2 node job CR.

The v1alpha1 node jobs will no longer be maintained, and relevant codes will be clean up after v1.23. In addition, the node job has become a default **disabled Beehive module** in EdgeCore. 
If you want to use the node jobs, please modify the edgecore.yaml to enable it.
```yaml
  modules:
    ...
    taskManager:
      enable: true
```

- From v1.21, keadm upgrade related commands(backup, upgrade, rollback) at the edge have been adjusted.
  - The upgrade command will not automatically execute the backup. The backup command needs to be triggered manually.
  - The upgrade command hides business-related flags and relevant codes will be cleaned up after v1.23.
  - All upgrade related commands use level 3 commands:
  ```shell
  keadm edge upgrade
  keadm edge backup
  keadm edge rollback
  ```
