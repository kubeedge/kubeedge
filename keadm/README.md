
# KubeEdge Installer

Click [here](https://kubeedge.io/en/docs/setup/keadm/) for detailed documentation of the KubeEdge installer.

## Networking note for edge nodes

Pod networking on edge nodes is not automatically available just because a CNI plugin is installed on cloud-side Kubernetes nodes. If your edge workloads require container networking, install and configure a CNI plugin on each edge node (binaries + config). `hostNetwork` pods do not require CNI. KubeEdge does not install CNI on edge nodes by default.
