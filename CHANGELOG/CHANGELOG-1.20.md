* [v1.20.0](#v1200)
    * [Downloads for v1.20.0](#downloads-for-v1200)
    * [KubeEdge v1.20 Release Notes](#kubeedge-v120-release-notes)
        * [1.20 What's New](#120-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)


# v1.20.0

## Downloads for v1.20.0

Download v1.20.0 in the [v1.20.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.20.0).

## KubeEdge v1.20 Release Notes

## 1.20 What's New

### Support Batch Node Process

Currently, the keadm tool of KubeEdge only supports manual single-node management. However, in edge scenarios, the scale of nodes is often very large, and the management process of a single node can no longer cope with such large-scale scenarios. 

In v1.20, we have provided the batch node operation and maintenance capability. With this capability, users only need to use one configuration file to perform batch operation and maintenance on all edge nodes through a control node (which can log in to all edge nodes). The supported operation and maintenance capabilities include join, reset, and upgrade.

Refer to the link for more details.([#5988](https://github.com/kubeedge/kubeedge/pull/5988), [#5968](https://github.com/kubeedge/kubeedge/pull/5968))

### Multi-language Mapper-Framework Support

To further reduce the complexity of developing custom Mapper, in this version, KubeEdge provides the Java version of Mapper-Framework. Users can access the KubeEdge feature-multilingual-mapper branch to use Mapper-Framework to generate a Java version of custom Mapper project.

Refer to the link for more details.([#5773](https://github.com/kubeedge/kubeedge/pull/5773), [#5900](https://github.com/kubeedge/kubeedge/pull/5900))

### Support Pods logs/exec/describe and Devices get/edit/describe Operation at Edge Using `keadm ctl`

In v1.17, a new command `keadm ctl` has been introduced to support pods query and restart at Edge. In this release, `keadm ctl` supports more functionality including `pod logs/exec/describe` and `device get/edit/describe` to help users operate resources at edge, especially in offline scenarios.

Refer to the link for more details.([#5752](https://github.com/kubeedge/kubeedge/pull/5752), [#5901](https://github.com/kubeedge/kubeedge/pull/5901))

### Decouple EdgeApplications from NodeGroups, Support Node Label Selector 

EdgeApplication can be overrides deployment spec(i.e. replicas, image, commands and environments) via the node group, and pod traffics are closed-loop in a node group(Deployments managed by EdgeApplication share a Service). But in the real scenario, the scope of nodes that need batch operations is different from that of nodes that need to collaborate with each other. 

We add a new targetNodeLabels field for node label selectors in the EdgeApplication CRD, this field will allow the application to deploy based on node labels and apply overrides specific to those nodes.

Refer to the link for more details.([#5755](https://github.com/kubeedge/kubeedge/issues/5755), [#5845](https://github.com/kubeedge/kubeedge/pull/5845))

### CloudHub-EdgeHub Supports IPv6 

We provide a configuration guide in the documentation on the official website, which is how KubeEdge enables the cloud-edge hub to support IPv6 in a K8s cluster.

Refer to the document https://kubeedge.io/docs/advanced/support_ipv6

### Upgrade Kubernetes Dependency to v1.30.7

Upgrade the vendered kubernetes version to v1.30.7, users are now able to use the feature of new version on the cloud and on the edge side.

Refer to the link for more details. ([#5997](https://github.com/kubeedge/kubeedge/issues/5997)

## Important Steps before Upgrading

- From v1.20, the default value of the EdgeCore configuration option `edged.rootDirectory` will change from `/var/lib/edged` to `/var/lib/kubelet`. If you wish to continue using the original path, you can set `--set edged.rootDirectory=/var/lib/edged` when installing EdgeCore with keadm.