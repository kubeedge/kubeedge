* [v1.18.0](#v1180)
    * [Downloads for v1.18.0](#downloads-for-v1180)
    * [KubeEdge v1.18 Release Notes](#kubeedge-v118-release-notes)
        * [1.18 What's New](#118-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)


# v1.18.0

## Downloads for v1.18.0

Download v1.18.0 in the [v1.18.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.18.0).

## KubeEdge v1.18 Release Notes

## 1.18 What's New

### Router Manager Supports High Availability(HA)

When CloudCore adopts high availability deployment, RouterManager needs to determine whether to route messages to the correct CloudCore. This feature is already supported in v1.18.0, and RouterManager supports high availability.

Refer to the link for more details. ([#5619](https://github.com/kubeedge/kubeedge/pull/5619), [#5635](https://github.com/kubeedge/kubeedge/pull/5635))

### Authorization Enhancement for CloudCore Websocket API

CloudCore need restrict the access to cluster resources for edge nodes. In this release，CloudCore supports node authorization mode. CloudHub identify the sender of messages and check whether the sender has sufficient permissions, so that CloudCore can restrict an edge node from operating the resources owned by other edge nodes.

Refer to the link for more details.([#5512](https://github.com/kubeedge/kubeedge/pull/5512), [#5585](https://github.com/kubeedge/kubeedge/pull/5585))

### Support Device Status Reporting 

Device status reporting is a capability required for device management. It was previously planned but not implemented. In version 1.18, we support this feature. Device status reporting can be easily implemented based on the community mapper template.

Refer to the link for more details.([#5651](https://github.com/kubeedge/kubeedge/pull/5651), [#5649](https://github.com/kubeedge/kubeedge/pull/5649), [#5650](https://github.com/kubeedge/kubeedge/pull/5650))

### Keadm Tool Enhancement

Before this release, keadm(KubeEdge Installation Tool) is only supported to configure a subset of parameters before EdgeCore was installed. Now we can use the '--set' flag to configure the parameters of the full configuration edgecore.yaml file, so that users can customize the parameters at installation time, without having to modify the configuration and restart the service after installation.

Refer to the link for more details.([#5564](https://github.com/kubeedge/kubeedge/pull/5564), [#5574](https://github.com/kubeedge/kubeedge/pull/5574))

### Encapsulate Token, CA and Certificate operations 

We refactor the token and certificate-related codes, summarize the same businesses, and abstract the ability of certificates to improve scalability, maintainability, and readability.

Refer to the link for more details.([#5502](https://github.com/kubeedge/kubeedge/pull/5502), [#5544](https://github.com/kubeedge/kubeedge/pull/5544))

### Upgrade Kubernetes Dependency to v1.29.6 

Upgrade the vendered kubernetes version to v1.29.6, users are now able to use the feature of new version on the cloud and on the edge side. 

Refer to the link for more details. ([#5656](https://github.com/kubeedge/kubeedge/pull/5656))

## Important Steps before Upgrading

- The CloudCore Authorization feature is disabled by default in release 1.18. If you need to use this feature, please set `cloudhub.authorization.enable=true`.