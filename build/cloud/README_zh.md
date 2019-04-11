## 部署 cloud 端到 k8s 集群

此方式将部署 cloud 端到 k8s 集群，所以需要登录到 k8s 的 master 节点上（或者其他可以用 `kubectl` 操作集群的机器）。

存放在 `github.com/kubeedge/kubeedge/build/cloud` 里的各个编排文件和脚本会被用到。所以需要先将这些文件放到可以用 kubectl 操作的地方。

首先， 确保 k8s 集群可以拉到 edge controller 镜像。如果没有， 可以构建一个，然后推到集群能拉到的 registry 上。

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make cloudimage
```

然后，需要生成 tls 证书。这步成功的话，会生成 `06-secret.yaml`。

```bash
cd build/cloud
../tools/certgen.sh buildSecret | tee ./06-secret.yaml
```

接着，按照编排文件的文件名顺序创建各个 k8s 资源。在创建之前，应该检查每个编排文件内容，以确保符合特定的集群环境。

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```

最后，基于`08-service.yaml.example`，创建一个适用于你集群环境的 service，
将 cloud hub 暴露到集群外，让 edge core 能够连到。