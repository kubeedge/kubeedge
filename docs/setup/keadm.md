# Deploying using Keadm

Keadm is used to install the cloud and edge components of KubeEdge. It is not responsible for installing K8s and runtime, so check [dependences](../getting-started.md#Dependencies) first.

Please refer [kubernetes-compatibility](https://github.com/kubeedge/kubeedge#kubernetes-compatibility) to get **Kubernetes compatibility** and determine what version of Kubernetes would be installed.

## Limitation

- Currently support of `keadm` is available for Ubuntu and CentOS OS. RaspberryPi supports is in-progress.
- Need super user rights (or root rights) to run.

## Setup Cloud Side (KubeEdge Master Node)

By default ports `10000` and `10002` in your cloudcore needs to be accessible for your edge nodes.

**Note**: port `10002` only needed since 1.3 release.

`keadm init` will install cloudcore, generate the certs and install the CRDs. It also provides a flag by which a specific version can be set.

**IMPORTANT NOTE:**
1. At least one of kubeconfig or master must be configured correctly, so that it can be used to verify the version and other info of the k8s cluster.
1. Please make sure edge node can connect cloud node using local IP of cloud node, or you need to specify public IP of cloud node with `--advertise-address` flag.
1. `--advertise-address`(only work since 1.3 release) is the address exposed by the cloud side (will be added to the SANs of the CloudCore certificate), the default value is the local IP.

Example:

```shell
# keadm init --advertise-address="THE-EXPOSED-IP"(only work since 1.3 release)
```

Output:
```
Kubernetes version verification passed, KubeEdge installation will start...
...
KubeEdge cloudcore is running, For logs visit:  /var/log/kubeedge/cloudcore.log
```

## (**Only Needed in Pre 1.3 Release**) Manually copy certs.tgz from cloud host to edge host(s)

**Note**: Since release 1.3, feature `EdgeNode auto TLS Bootstrapping` has been added and there is no need to manually copy certificate.

Now users still need to copy the certs to the edge nodes. In the future, it will support the use of tokens for authentication.

On edge host:

```
mkdir -p /etc/kubeedge
```

On cloud host:

```
cd /etc/kubeedge/
scp -r certs.tgz username@edge_node_ip:/etc/kubeedge
```

On edge host untar the certs.tgz file

```
cd /etc/kubeedge
tar -xvzf certs.tgz
```

## Setup Edge Side (KubeEdge Worker Node)

### Get Token From Cloud Side

Run `keadm gettoken` in **cloud side** will return the token, which will be used when joining edge nodes.

```shell
# keadm gettoken
27a37ef16159f7d3be8fae95d588b79b3adaaf92727b72659eb89758c66ffda2.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTAyMTYwNzd9.JBj8LLYWXwbbvHKffJBpPd5CyxqapRQYDIXtFZErgYE
```

### Join Edge Node

`keadm join` will install edgecore and mqtt. It also provides a flag by which a specific version can be set.

Example:

```shell
# keadm join --cloudcore-ipport=192.168.20.50:10000 --token=27a37ef16159f7d3be8fae95d588b79b3adaaf92727b72659eb89758c66ffda2.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTAyMTYwNzd9.JBj8LLYWXwbbvHKffJBpPd5CyxqapRQYDIXtFZErgYE
```

**IMPORTANT NOTE:**
1. `--cloudcore-ipport` flag is a mandatory flag.
1. If you want to apply certificate for edge node automatically, `--token` is needed.
1. The kubeEdge version used in cloud and edge side should be same.

Output:

```shell
Host has mosquit+ already installed and running. Hence skipping the installation steps !!!
...
KubeEdge edgecore is running, For logs visit:  /var/log/kubeedge/edgecore.log
```

## Reset KubeEdge Master and Worker nodes

### Master
`keadm reset` will stop `cloudcore` and delete KubeEdge related resources from Kubernetes master like `kubeedge` namespace. It doesn't uninstall/remove any of the pre-requisites.

It provides a flag for users to specify kubeconfig path, the default path is `/root/.kube/config`.

 Example:

```shell
 # keadm reset --kube-config=$HOME/.kube/config
```

 ### Node
`keadm reset` will stop `edgecore` and it doesn't uninstall/remove any of the pre-requisites.