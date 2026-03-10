* [v1.23.0](#v1230)
    * [Downloads for v1.23.0](#downloads-for-v1230)
    * [KubeEdge v1.23 Release Notes](#kubeedge-v123-release-notes)
        * [1.23 What's New](#123-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)

# v1.23.0

## Downloads for v1.23.0

Download v1.23.0 in the [v1.23.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.23.0).

## KubeEdge v1.23 Release Notes

## 1.23 What's New

### EdgeCore and Keadm Capability Enhancements on Windows OS

- Provide local DMI service: As we cannot use Unix Domain Socket on Windows, we implement local network communication through Windows named pipes, similar to Containerd and Kubelet. 

    Refer to the link for more details.([#6563](https://github.com/kubeedge/kubeedge/pull/6563))

- Keadm Upgrade/Download Enhancement on Windows: In v1.23.0, keadm detects the version of an existing `edgecore.exe` and will re-download the EdgeCore package when a newer version is available. This prevents upgrades from being skipped simply because `edgecore.exe` already exists on disk.

    Refer to the link for more details.([#6580](https://github.com/kubeedge/kubeedge/pull/6580))

- Observability Enhancement: In new release, when EdgeCore runs as a Windows service, logs are redirected to a log file. This improves troubleshooting and operational visibility on Windows. 

  Refer to the link for more details.([#6565](https://github.com/kubeedge/kubeedge/pull/6565))

### Support Device Anomaly Detection in Device CRDs and mappers

In v1.23.0, device anomaly detection framework is introduced in KubeEdge device management. Users can specify the configuration for anomaly detection in Device CRDs `pushMethod`. It also enables mappers to implement and run the anomaly detection logic, making anomaly detection pluggable at the mapper level and connected to the device status reporting workflow. 

Refer to the link for more details.([#6478](https://github.com/kubeedge/kubeedge/pull/6478), [#6543](https://github.com/kubeedge/kubeedge/pull/6543))

### Optimizing Node Querying Path from Edge to Reduce Edge-Cloud Bandwidth

Previously, EdgeCore relied on remote querying of node resources via CloudCore, leading to significant bandwidth strain on the edge-cloud channel as the number of nodes scaled.

In this release, we have optimized the node querying path. EdgeCore now retrieves node directly from the local edge database. Furthermore, CloudCore is enhanced to automatically synchronize any updated node information to the edge database upon detection. This optimization significantly improves performance and reliability, especially in large-scale edge deployments.

Refer to the link for more details.([#6489](https://github.com/kubeedge/kubeedge/pull/6489))

### Replace Beego with Gorm and Reconstruct Edge DB

Previously, the edge database utilized the `Beego` framework, although only its ORM module was employed. In v1.23.0, we have replaced `Beego` with `GORM`, resulting in a lighter-weight edge component.

Furthermore, we have refactored all database operations. A unified database operation entry point has been introduced within the MetaManager module. This refactoring ensures that all database interactions are clearer, more maintainable, and centralized.

Refer to the link for more details.([#6296](https://github.com/kubeedge/kubeedge/issues/6296), [#6585](https://github.com/kubeedge/kubeedge/pull/6585))

### Upgrade Kubernetes Dependency to v1.32.10

Upgrade the vendered kubernetes version to v1.32.10, users are now able to use the features of new version on the cloud and on the edge side.

Refer to the link for more details.([#6549](https://github.com/kubeedge/kubeedge/pull/6549))

### New Release Dashboard: i18n(Chinese), Performance Enhancements and UI Improvement

We are pleased to announce the release of Dashboard v0.2.0. This new version brings the following key updates: 

- Introduce a Backend-for-Frontend(BFF) layer to offload data processing from the UI and improve performance. 
- Provide a foundational framework for internationalization and introduce Chinese language pack.
- Standardize the visual style, improve user-friendly interactions and optimize the data flow for `PodTable` and `TableCard` components to improve user experience. 

Refer to the link for more details. ([Dashboard/v0.2.0](https://github.com/kubeedge/dashboard/tree/v0.2.0))

## Important Steps before Upgrading

- In v1.23.0, we extract the status part of `Device CRD` into a seperate `DeviceStatus CRD`. This change maintains backward compatibility with older versions of the CRD. However, please note that device status must now be retrieved from the new `DeviceStatus CRD`. You can refer to [#6534](https://github.com/kubeedge/kubeedge/pull/6534) for more details.