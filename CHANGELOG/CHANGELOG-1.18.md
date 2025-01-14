* [v1.18.3](#v1183)
    * [Downloads for v1.18.3](#downloads-for-v1183)
    * [KubeEdge v1.18.3 Release Notes](#kubeedge-v1183-release-notes)
        * [Changelog since v1.18.2](#changelog-since-v1182)
* [v1.18.2](#v1182)
    * [Downloads for v1.18.2](#downloads-for-v1182)
    * [KubeEdge v1.18.2 Release Notes](#kubeedge-v1182-release-notes)
        * [Changelog since v1.18.1](#changelog-since-v1181)
* [v1.18.1](#v1181)
    * [Downloads for v1.18.1](#downloads-for-v1181)
    * [KubeEdge v1.18.1 Release Notes](#kubeedge-v1181-release-notes)
        * [Changelog since v1.18.0](#changelog-since-v1180)
* [v1.18.0](#v1180)
    * [Downloads for v1.18.0](#downloads-for-v1180)
    * [KubeEdge v1.18 Release Notes](#kubeedge-v118-release-notes)
        * [1.18 What's New](#118-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)


# v1.18.3

## Downloads for v1.18.3

Download v1.18.3 in the [v1.18.3 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.18.3).

## KubeEdge v1.18.3 Release Notes

### Changelog since v1.18.2

- Fix clusterobjectsync cannot be deleted when edge node deleted. ([#6059](https://github.com/kubeedge/kubeedge/pull/6059), [@wbc6080](https://github.com/wbc6080))
- Fix multiple `--set` parameters don't take effect in `keadm join` command. ([#6065](https://github.com/kubeedge/kubeedge/pull/6065), [@XmchxUp](https://github.com/XmchxUp))
- Fix duplicate generation of certificate if etcd fails. ([#6066](https://github.com/kubeedge/kubeedge/pull/6066), [@LRaito](https://github.com/LRaito))
- Fix iptablesmanager cannot clean iptables rules when CloudCore deleted. ([#6072](https://github.com/kubeedge/kubeedge/pull/6072), [@wbc6080](https://github.com/wbc6080))


# v1.18.2

## Downloads for v1.18.2

Download v1.18.2 in the [v1.18.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.18.2).

## KubeEdge v1.18.2 Release Notes

### Changelog since v1.18.1

- Fix errors due to singular and plural conversion in MetaServer. ([#5917](https://github.com/kubeedge/kubeedge/pull/5917), [@wbc6080](https://github.com/wbc6080))
- Fix token cannot be refreshed. ([#5984](https://github.com/kubeedge/kubeedge/pull/5984), [@WillardHu](https://github.com/WillardHu))
- Fix install EdgeCore failed with CRI-O(>v1.29.2) for uid missing. ([#5990](https://github.com/kubeedge/kubeedge/pull/5990), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.18.1

## Downloads for v1.18.1

Download v1.18.1 in the [v1.18.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.18.1).

## KubeEdge v1.18.1 Release Notes

### Changelog since v1.18.0

- Optimize time format to support international time. ([#5819](https://github.com/kubeedge/kubeedge/pull/5819), [@WillardHu](https://github.com/WillardHu))
- Fix keadm reset lack of flag remote-runtime-endpoint. ([#5848](https://github.com/kubeedge/kubeedge/pull/5848), [@tangming1996](https://github.com/tangming1996))
- Fix PersistentVolumes data stored at edge deleted abnormally.  ([#5867](https://github.com/kubeedge/kubeedge/pull/5867), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

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