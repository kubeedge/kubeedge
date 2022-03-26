## 部署 cloud 端到 k8s 集群

此方式将部署 cloud 端到 k8s 集群，所以需要登录到 k8s 的 master 节点上（或者其他可以用 `kubectl` 操作集群的机器）。

存放在 `github.com/kubeedge/kubeedge/build/cloud` 里的各个编排文件和脚本会被用到。所以需要先将这些文件放到可以用 kubectl 操作的地方。

首先， 确保 k8s 集群可以拉到 edge controller 镜像。如果没有， 可以构建一个，然后推到集群能拉到的 registry 上。

```bash
cd $GOPATH/src/github.com/kubeedge/kubeedge
make image WHAT=cloudcore
```

(可选)然后，使用1.3.0以下版本时，需要手动生成 tls 证书。这步成功的话，会生成 `06-secret.yaml`。

```bash
cd build/cloud
../tools/certgen.sh buildSecret | tee ./06-secret.yaml
```

(可选)从KubeEdge 1.3.0开始，我们可以在`05-configmap.yaml`中配置暴露给边缘节点的所有CloudCore IP地址（例如浮动IP），并将其添加到Cloudcore证书中的SAN中。

```
modules:
  cloudHub:
    advertiseAddress:
    - 10.1.11.85
```

最后，基于`08-service.yaml.example`，创建一个适用于你集群环境的 service`08-service.yaml`，
将 cloud hub 暴露到集群外，让 edge core 能够连到。

接着，按照编排文件的文件名顺序创建各个 k8s 资源。在创建之前，应该检查每个编排文件内容，以确保符合特定的集群环境。

```bash
for resource in $(ls *.yaml); do kubectl create -f $resource; done
```

---
> 以下仅针对网络环境不能正常下载相应镜像的说明:

在可能有网络问题的情况下,在执行07-deployment.yaml的时候
deployment中对应的init container会先去
```
apk --no-cache add coreutils && cat | tee /etc/kubeedge/cloud/kubeconfig.yaml
```
这一步可能会因为网络的原因不能成功执行,报错为
```
ERROR: unsatisfiable constraints:
  coreutils (missing):
    required by: world[coreutils]
The command '/bin/sh -c apk --no-cache add coreutils' returned a non-zero code: 1
```

解决办法为在有网络环境的先制作一个init container image
Dockerfile
```
FROM alpine:3.9
RUN apk --no-cache add coreutils
```
再替换老的init container的image,以及删除掉`apk --no-cache add coreutils &&`即可
