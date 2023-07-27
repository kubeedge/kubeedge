
* [v1.14.0](#v1140)
    * [Downloads for v1.14.0](#downloads-for-v1140)
    * [KubeEdge v1.14 Release Notes](#kubeedge-v114-release-notes)
        * [1.14 What's New](#114-whats-new)
        * [Important Steps before Upgrading](#important-steps-before-upgrading)
        * [Other Notable Changes](#other-notable-changes)
        * [Bug Fixes](#bug-fixes)



# v1.14.0

## Downloads for v1.14.0

Download v1.14.0 in the [v1.14.0 release page](https://github.com/kubeedge/kubeedge/releases/tag/v1.14.0).

## KubeEdge v1.14 Release Notes

## 1.14 What's New

### Support Authentication and Authorization for Kube-API Endpoint for Applications On Edge Nodes

The Kube-API endpoint for edge applications is implemented through MetaServer in edegcore. However, in previous versions, 
the authentication and authorization of Kube-API endpoint are performed in the cloud, which prevents authentication and authorization 
especially in offline scenarios on the edge node.

In this release, the authentication and authorization functionalities are implemented within the MetaServer at edge, which allows for 
limiting the access permissions of edge applications when accessing Kube-API endpoint at edge.

Refer to the link for more details. ([#4802](https://github.com/kubeedge/kubeedge/pull/4802))

### Support Cluster Scope Resource Reliable Delivery to Edge Node

The cluster scope resource can guarantee deliver to the edge side reliably since this release, 
especially include using list-watch global resources, the cluster scope resource can be delivered to the edge side reliably,
and the edge applications can work normally.

Refer to the link for more details. ([#4758](https://github.com/kubeedge/kubeedge/pull/4758))


### Upgrade Kubernetes Dependency to v1.24.14

Upgrade the vendered kubernetes version to v1.24.14, users are now able to use the feature of new version on the cloud and on the edge side.
**Note**: The dockershim has been removed, which means users can't use docker runtime directly in this release.

Refer to the link for more details. ([#4789](https://github.com/kubeedge/kubeedge/pull/4789))


### Support Kubectl Attach to Container Running on Edge Node

KubeEdge already support `kubectl logs/exe` command, `kubectl attach` is supported in this release.
`kubectl attach` command can attach to a running container at edge node.
Users can execute these commands in the cloud and no need to operate on the edge nodes.

Refer to the link for more details. ([#4734](https://github.com/kubeedge/kubeedge/pull/4734))


### Alpha version of KubeEdge Dashboard

KubeEdge dashboard provides a graphical user interface (GUI) for managing and monitoring your KubeEdge clusters. 
It allows users to manage edge applications running in the cluster and troubleshoot them.

Refer to the link for more details. (https://github.com/kubeedge/dashboard)



## Important Steps before Upgrading

- On KubeEdge v1.14, EdgeCore has removed the dockeshim support, so users can only use `remote` type runtime, and uses `containerd` runtime by default. If you want to use `docker` runtime, you
  must first set `edged.containerRuntime=remote` and corresponding docker configuration like `RemoteRuntimeEndpoint` and `RemoteImageEndpoint` in EdgeCore, then install the cri-dockerd tools as docs below:
  
