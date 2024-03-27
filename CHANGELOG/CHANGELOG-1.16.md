* [v1.16.1](#v1161)
  * [Downloads for v1.16.1](#downloads-for-v1161)
  * [KubeEdge v1.16.1 Release Notes](#kubeedge-v1161-release-notes)
    * [Changelog since v1.16.0](#changelog-since-v1160)
* [v1.16.0](#v1160)
    * [Downloads for v1.16.0](#downloads-for-v1160)
    * [KubeEdge v1.16 Release Notes](#kubeedge-v116-release-notes)
        * [1.16 What's New](#116-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.16.1

## Downloads for v1.16.1

Download v1.16.1 in the [v1.16.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.16.1).

## KubeEdge v1.16.1 Release Notes

### Changelog since v1.16.0

- Fix edgeapplication differentiated configuration where env support was not comprehensive. ([#5455](https://github.com/kubeedge/kubeedge/pull/5455), [@tangming1996](https://github.com/tangming1996))
- Fix character error in edgeapplication API. ([#5460](https://github.com/kubeedge/kubeedge/pull/5460), [@tangming1996](https://github.com/tangming1996))
- Fix metaserver panic due to nil initializers in request scope. ([#5479](https://github.com/kubeedge/kubeedge/pull/5479), [@Windrow14](https://github.com/Windrow14))
- Fix incorrect handling of retryTimes in imagePrePullJob. ([#5491](https://github.com/kubeedge/kubeedge/pull/5491), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))
- Fix Keadm upgrade command. ([#5492](https://github.com/kubeedge/kubeedge/pull/5492), [@WillardHu](https://github.com/WillardHu))

## Downloads for v1.16.0

Download v1.16.0 in the [v1.16.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.16.0).

## KubeEdge v1.16 Release Notes

## 1.16 What's New

### Support Cloud and Edge Components Upgrade

The Cloud side and Edge side Upgrade capability is comprehensively enhanced in v1.16. Users can upgrade the cloud side components with Keadm tool, and upgrade edge nodes with the API through Kubernetes API-Server.

- **Cloud upgrade**

    Keadm supports the Cloud Upgrade command, and users can easily upgrade cloud components.

- **Edge upgrade**

    In KubeEdge v1.16, the node upgrade API was implemented. Users can remotely upgrade edge nodes in batches. The cloud-edge task architecture handles upgrade task flow and supports unified timeout processing, concurrency control, and subtask management, among other capabilities.

- **KubeEdge version compatibility testing**

    KubeEdge v1.16 provides KubeEdge version compatibility testing, which avoids problems caused by incompatible cloud-edge versions during the upgrading process.

Refer to the link for more details. ([#5330](https://github.com/kubeedge/kubeedge/pull/5330), [#5229](https://github.com/kubeedge/kubeedge/pull/5229), [#5289](https://github.com/kubeedge/kubeedge/pull/5289))

### Alpha Implementation of Images PrePull on Edge Nodes 

In scenarios with unstable network or limited edge bandwidth, deploying or updating edge applications often results in high failure rates or reduced efficiency, especially with large-scale edge nodes.

Images PrePull feature has been introduced in v1.16. Users can perform batch images prepull on large-scale edge nodes with `ImagePrePullJob` API when the network is stable, to improve the success rate and efficiency of batch edge applications deploying and updating.

Refer to the link for more details. ([#5310](https://github.com/kubeedge/kubeedge/pull/5310), [#5331](https://github.com/kubeedge/kubeedge/pull/5331))

### Support Installing Windows-based Edge Nodes with Keadm 

KubeEdge has supported the edge node running on Windows Server 2019 in v1.15, extending KubeEdge to the Windows ecosystem and expanding its use cases and ecosystem.

In this release, Windows-based Edge Nodes can be installed and registered to cloud with the installation tool `Keadm`, providing convenience for the application of KubeEdge in Windows OS.

Refer to the link for more details. ([#4968](https://github.com/kubeedge/kubeedge/pull/4968))

### Add Compatibility Tests for Multiple Runtimes 

The e2e test of KubeEdge v1.16 has integrated compatibility tests for multiple container runtimes. Currently, four container runtime compatibility tests have been added, including **containerd**, **docker**, **cri-o**, and **isulad**.

Refer to the link for more details.([#5321](https://github.com/kubeedge/kubeedge/pull/5321))

### Support More Deployment Fields to the EdgeApplication Overrides 

In previous versions, only replicas and image of the EdgeApplication could be overridden. In this release, we support overriding more Deployment fields: env, command, args and resources.

Refer to the link for more details.([#5038](https://github.com/kubeedge/kubeedge/pull/5038))

### Support Mapper Upgrade 

Build mapper upgrade framework. Users can upgrade the mapper by changing the referenced mapper-framework package version.

- **Mapper-framework code decouple**

    The code in mapper-framework was decoupled into user-layer code and business-layer code, and create the [kubeedge/mapper-framework](https://github.com/kubeedge/mapper-framework) repo to store the business layer code.

- **Mapper upgrade framework**

    Update the way mapper-framework generates mapper projects. The current execution script will only generate user-level code through dependent references. When the mapper project needs to be upgraded, it can be directly made by changing the version of mapper-framework package.

Refer to the link for more details.([#5308](https://github.com/kubeedge/kubeedge/pull/5308), [#5326](https://github.com/kubeedge/kubeedge/pull/5326))

### Integrate Redis and TDengine Database in DMI Data Plane

Integrate redis and tdengine database in DMI data plane. The mapper project generated by mapper-framework has build-in ability to push data to redis and tdengine database. Users can push data directly through configuring device instance files.

Refer to the link for more details.([#5064](https://github.com/kubeedge/kubeedge/pull/5064))

### New USB Camera Mapper 

Based on the mapper and dmi framework in KubeEdge v1.15.0, a mapper for USB cameras has been developed, which supports data push to Influxdb, mqtt, and http. It has been successfully applied in practice.

Refer to the link for more details.([#122](https://github.com/kubeedge/mappers-go/pull/122))

### Keadm’s Enhancement

- When using Keadm join in kubeEdge v1.16, it supports the selection of communication protocols for edge nodes and cloud center nodes. The cloud edge communication protocol is configured through the parameter --hub-protocol, and currently supports two communication protocols: websocket and quic.

  **Note**: When the --hub-protocol parameter is configured as quic, it is necessary to set the port of the parameter --cloudcore-ipport  to 10001 and modify configmap in cloudcore to open the quic protocol.

    Refer to the link for more details.([#5156](https://github.com/kubeedge/kubeedge/pull/5156))

- In KubeEdge v1.16, it is already supported for Keadm to complete edgecore deployment through Keadm join without installing the CNI plugin, decoupling the deployment of edge nodes from the CNI plugin. At the same time, this feature has been synchronized to v1.12 and later versions.

  **Note**: If the application deployed on edge nodes needs to use container networks, it is still necessary to install the CNI plugin after deploying edgecore.

    Refer to the link for more details.([#5196](https://github.com/kubeedge/kubeedge/pull/5196))

### Upgrade Kubernetes Dependency to v1.27.7

Upgrade the vendered kubernetes version to v1.27.7, users are now able to use the feature of new version on the cloud and on the edge side.

Refer to the link for more details. ([#5121](https://github.com/kubeedge/kubeedge/pull/5121))

## Important Steps before Upgrading

- Now we use DaemonSet to manage the mqtt broker mosquitto. You need to consider whether to use the static pod managed mqtt broker in the edge node or use the DaemonSet managed mqtt broker in the cloud, they cannot coexist and there will be port conflicts. You can read the guide `For edge node low version compatibility` in [#5233](https://github.com/kubeedge/kubeedge/issues/5233).

- In this release, the flag `with-mqtt` will be set to deprecated and default to false, but will not be removed. After v1.18, the code related to static pod management will be removed in the edge, and the flag `with-mqtt` no longer supported.
