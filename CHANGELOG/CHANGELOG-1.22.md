* [v1.22.0](#v1220)
    * [Downloads for v1.22.0](#downloads-for-v1220)
    * [KubeEdge v1.22 Release Notes](#kubeedge-v122-release-notes)
        * [1.22 What's New](#122-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)


# v1.22.0

## Downloads for v1.22.0

Download v1.22.0 in the [v1.22.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.22.0).

## KubeEdge v1.22 Release Notes

## 1.22 What's New

### Add Hold/Release Mechanism for Controlling Edge Resource Updates

In applications such as autonomous driving, drones, and robotics, we want to control when updates to edge resources occur, ensuring that these resources cannot be updated without the permission of the edge administrator. 

In v1.22.0, we introduced a hold/release mechanism to control updates to edge resources. We can use annotations `edge.kubeedge.io/hold-upgrade: "true"` on Deployments, StatefulSets, DaemonSets, etc., to indicate that their Pod updates should be held at the edge and use `keadm ctl <command>` (`keadm ctl unhold-upgrade pod <podname>` and `keadm ctl unhold-upgrade node`) to release the hold and allow the update to apply.

Refer to the link for more details.([#6348](https://github.com/kubeedge/kubeedge/pull/6348), [#6418](https://github.com/kubeedge/kubeedge/pull/6418))

### Beehive Framework Upgrade, Supporting Configurable Submodule Restart Policies

In release 1.17, we implemented auto-restart for the EdgeCore modules, allowing global configuration of edge modules restarts. In release 1.22, we optimized the `Beehive` framework to support restart policy configurations for edge submodules. We also standardized the error handling for starting `Beehive` submodules.

Refer to the link for more details.([#6444](https://github.com/kubeedge/kubeedge/pull/6444), [#6445](https://github.com/kubeedge/kubeedge/pull/6445))

### Device Model Update Based On Thing Model and Product Concept

Current Device Model is designed based on the thing model concept. In traditional IoT, devices are usually designed with a three-tier structure: thing model, product, and device instance, which can lead to user confusion during actual use.

In v1.22.0, we upgraded the device model design by integrating the concepts of thing models and actual products. We extracted the `protocolConfigData` and `visitors` fields from existing device instances into the device model, allowing device instances to share these model configurations. Additionally, to reduce the cost of separating models, device instances can override these configurations.

Refer to the link for more details.([#6457](https://github.com/kubeedge/kubeedge/pull/6457), [#6458](https://github.com/kubeedge/kubeedge/pull/6458))

### Adds FeatureGates for Pod Resources Server and CSI Plugin in EdgeCore Integrated Lightweight Kubelet

In previous versions, we removed the Pod Resources Server capability from the integrated lightweight Kubelet in EdgeCore. However, in some use cases, users wish to restore this capability for monitoring Pods. Additionally, the default activation of the CSI Plugin in Kubelet can lead to failures in offline environments due to failed CSINode creation when starting EdgeCore.

In v1.22.0, we added featureGates for the Pod Resources Server and CSI Plugin in the lightweight Kubelet. If you need to enable the Pod Resources Server or disable the CSI Plugin, you can add corresponding featureGates to your EdgeCore configuration.

Refer to the link for more details.([kubeedge/kubernetes#12](https://github.com/kubeedge/kubernetes/pull/12), [kubeedge/kubernetes#13](https://github.com/kubeedge/kubernetes/pull/13), [#6452](https://github.com/kubeedge/kubeedge/pull/6452))

### C language Mapper-Framework Support

In v1.20.0, we added a Java version of the Mapper-Framework based on the existing Go language version. Due to the diversity of communication protocols for edge IoT devices, many edge device driver protocols are implemented in C. Thus, in the new release, KubeEdge offers a C language version of the Mapper-Framework. Users can access the `feature-multilingual-mapper-c` branch in the KubeEdge main repository to generate custom Mapper projects in C using the Mapper Framework.

Refer to the link for more details.([#6405](https://github.com/kubeedge/kubeedge/pull/6405), [#6455](https://github.com/kubeedge/kubeedge/pull/6455))

### Upgrade Kubernetes Dependency to v1.31.12

Upgrade the vendered kubernetes version to v1.31.12, users are now able to use the features of new version on the cloud and on the edge side.

Refer to the link for more details.([#6443](https://github.com/kubeedge/kubeedge/pull/6443))

## Important Steps before Upgrading

NA