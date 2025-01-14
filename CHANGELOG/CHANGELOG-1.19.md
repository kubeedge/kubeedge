* [v1.19.2](#v1192)
    * [Downloads for v1.19.2](#downloads-for-v1192)
    * [KubeEdge v1.19.2 Release Notes](#kubeedge-v1192-release-notes)
        * [Changelog since v1.19.1](#changelog-since-v1191)
* [v1.19.1](#v1191)
    * [Downloads for v1.19.1](#downloads-for-v1191)
    * [KubeEdge v1.19.1 Release Notes](#kubeedge-v1191-release-notes)
        * [Changelog since v1.19.0](#changelog-since-v1190)
* [v1.19.0](#v1190)
    * [Downloads for v1.19.0](#downloads-for-v1190)
    * [KubeEdge v1.19 Release Notes](#kubeedge-v119-release-notes)
        * [1.19 What's New](#119-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.19.2

## Downloads for v1.19.2

Download v1.19.2 in the [v1.19.2 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.19.2).

## KubeEdge v1.19.2 Release Notes

### Changelog since v1.19.1

- Fix clusterobjectsync cannot be deleted when edge node deleted. ([#6058](https://github.com/kubeedge/kubeedge/pull/6058), [@wbc6080](https://github.com/wbc6080))
- Fix multiple `--set` parameters don't take effect in `keadm join` command. ([#6064](https://github.com/kubeedge/kubeedge/pull/6064), [@XmchxUp](https://github.com/XmchxUp))
- Fix duplicate generation of certificate if etcd fails. ([#6067](https://github.com/kubeedge/kubeedge/pull/6067), [@LRaito](https://github.com/LRaito))
- Fix iptablesmanager cannot clean iptables rules when CloudCore deleted. ([#6073](https://github.com/kubeedge/kubeedge/pull/6073), [@wbc6080](https://github.com/wbc6080))

# v1.19.1

## Downloads for v1.19.1

Download v1.19.1 in the [v1.19.1 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.19.1).

## KubeEdge v1.19.1 Release Notes

### Changelog since v1.19.0

- Fix token cannot be refreshed. ([#5984](https://github.com/kubeedge/kubeedge/pull/5984), [@WillardHu](https://github.com/WillardHu))
- Fix device compile failed. ([#5986](https://github.com/kubeedge/kubeedge/pull/5986), [@JiaweiGithub](https://github.com/JiaweiGithub))
- Fix install EdgeCore failed with CRI-O(>v1.29.2) for uid missing. ([#5990](https://github.com/kubeedge/kubeedge/pull/5990), [@Shelley-BaoYue](https://github.com/Shelley-BaoYue))

# v1.19.0

## Downloads for v1.19.0

Download v1.19.0 in the [v1.19.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.19.0).

## KubeEdge v1.19 Release Notes

## 1.19 What's New

### Support Edge Nodes Report Event

Kubernetes Event serve as a report of an event somewhere in the cluster, reflecting status changes of cluster resources such as Nodes and Pods. In v1.19, EdgeCore supports reporting events to cloud, allowing users to directly access the status of edge nodes or Pods in the cloud via `kubectl get events` or `kubectl describe {resource_type} {resource_name}`.

This feature is disabled by default in v1.19. To enable it, execute `--set modules.edged.reportEvent=true` when install EdgeCore with keadm or modify the EdgeCore configuration file and then restart EdgeCore.

Refer to the link for more details.([#5722](https://github.com/kubeedge/kubeedge/pull/5722), [#5811](https://github.com/kubeedge/kubeedge/pull/5811))

### Support OTA(Over-The-Air) Upgrades for Edge Nodes

On the basis of NodeUpgradeJob upgrade, we add the edge node confirmation card point and the validation of the image digest. The card point confirmation allows the node upgrade to be delivered to the edge side, and the upgrade can be performed only after the user is confirmed. Image digest validation can ensure that the kubeedge/installation-pacakge image to be upgraded is secure and reliable at the edge side.

In v1.19, we can use `spec.imageDigestGatter` in NodeUpgradeJob to define how to get the image digest. The `value` to directly define the digest, The `registryAPI` to get the mirror digest via registry v2 API, both are mutually exclusive. If none is configured, the image digest is not verified during the upgrade. 

We can also use `spec.requireConfirmation` to configure requireConfirmation for NodeUpgradeJob to determine whether we want to confirm at the edge side.

Refer to the link for more details.([#5589](https://github.com/kubeedge/kubeedge/issues/5589), [#5761](https://github.com/kubeedge/kubeedge/pull/5761), [#5863](https://github.com/kubeedge/kubeedge/pull/5863))

### Mapper Supports Device Data Writing

In v1.19, we add the ability to write device data in Mapper-Framework. User can use device methods through the API provided by Mapper and complete data writing to device properties.

- Device method API

A new definition of device methods is added in new release. Users can define device methods in the device-instance file that can be called by the outside world in device. Through device methods, users can control and write data to device properties.

- Device data writing

In v1.19, the Mapper API capability is improved and a new device method interface is added. The user can use the relevant interface to obtain all the device methods contained in a device, as well as the calling command of the device method.  Through the returned calling command, user can create a device write request to write data to device.

Refer to the link for more details.([#5662](https://github.com/kubeedge/kubeedge/pull/5662), [#5902](https://github.com/kubeedge/kubeedge/pull/5902))

### Add OpenTelemetry to Mapper-framework

In v1.19, we add the OpenTelemetry observability framework to mapper data plane, which can encapsulate device data and push data to multiple types of applications or databases. This feature can enhance the mapper data plane's ability to push device data.

Refer to the link for more details.([#5628](https://github.com/kubeedge/kubeedge/pull/5628))

### A New Release of KubeEdge Dashboard

Based on previous Dashboard release, we have refactored the KubeEdge Dashboard using the more popular frameworks Next.js and MUI. In the new release, we rewrote and optimized around 60 pages and components, reducing about 70% of redundant code. We also upgraded the KubeEdge and native Kubernetes APIs to the latest version to maintain compatibility and added TypeScript definitions for the APIs.

Refer to the link for more details.([#29](https://github.com/kubeedge/dashboard/pull/29))

## Important Steps before Upgrading

- In the next release (v1.20), the default value for the EdgeCore configuration option `edged.rootDirectory` will change from `/var/lib/edged` to `/var/lib/kubelet`. If you wish to continue using the original path, you can set `--set edged.rootDirectory=/var/lib/edged` when installing EdgeCore with keadm.

- In v1.19, please use `--kubeedge-version` to specify the version when installing KubeEdge with keadm, `--profile version` is no longer supported. 