## 部署 edge 端到 k8s 集群

此方式将部署 edge 端到 k8s 集群，所以需要登录到 k8s 的 master 节点上（或者其他可以用 `kubectl` 操作集群的机器）。

存放在 `github.com/kubeedge/kubeedge/build/edgecore/kubernetes` 里的各个编排文件和脚本会被用到。所以需要先将这些文件放到可以用 kubectl 操作的地方。

首先， 确保 k8s 集群可以拉到 edge core 镜像。如果没有， 可以构建一个，然后推到集群能拉到的 registry 上。

- 检查容器运行环境

```bash
  cd $GOPATH/src/github.com/kubeedge/kubeedge/build/edgecore
  ./run_daemon.sh prepare
```

- 构建edge core镜像

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make edgeimage
```

我们按照编排文件的文件名顺序创建各个 k8s 资源。在创建之前，应该检查每个编排文件内容，以确保符合特定的集群环境。

首先您需要去拷贝 edge certs 文件包括`edge.crt`和`edge.key`到您想要部署 edge part 的 k8s 节点上的`/etc/kubeedge/certs/`文件夹中。

另一方面，您需要替换`0.0.0.0:10000`成您的 kubeedge cloud web socket url。
* [url](03-configmap-edgenodeconf.yaml#L20)

默认的边缘节点名称是`edgenode1`，如果您想要改变节点名称或者是创建新的边缘节点，您需要用新的边缘节点名称替换如下几个地方。
* [name in 02-edgenode.yaml](02-edgenode.yaml#L4)
* [url in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L20)
* [node-id in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L33)
* [hostname-override in 03-configmap-edgenodeconf.yaml](03-configmap-edgenodeconf.yaml#L36)
* [name in 04-deployment-edgenode.yaml](04-deployment-edgenode.yaml#L4)

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```
