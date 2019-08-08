# build目录说明

```
# tree build/
build/
├── cloud
│   ├── 01-namespace.yaml
│   ├── 02-serviceaccount.yaml
│   ├── 03-clusterrole.yaml
│   ├── 04-clusterrolebinding.yaml
│   ├── 05-configmap.yaml
│   ├── 07-deployment.yaml
│   ├── 08-service.yaml.example
│   ├── Dockerfile
│   ├── README.md
│   └── README_zh.md
├── crds
│   └── devices
│       ├── devices_v1alpha1_devicemodel.yaml
│       └── devices_v1alpha1_device.yaml
├── crd-samples
│   └── devices
│       ├── create-device-instance.yaml
│       └── create-device-model.yaml
├── deployment.yaml
├── deployment-armv7.yaml
├── edge
│   ├── docker-compose.yaml
│   ├── Dockerfile
│   ├── kubernetes
│   │   ├── 01-namespace.yaml
│   │   ├── 02-edgenode.yaml
│   │   ├── 03-configmap-edgenodeconf.yaml
│   │   ├── 04-deployment-edgenode.yaml
│   │   ├── README.md
│   │   └── README_zh.md
│   ├── README.md
│   ├── README_zh.md
│   └── run_daemon.sh
├── node.json
└── tools
    └── certgen.sh
```

- cloud
- crds
- crd-samples
- deployment.yaml
- edge
- node.json
- tools

## cloud
edegcontroller以deployment部署的相关步骤及说明

## crds

## crd-samples

## edge
edgecore以deployment部署的相关步骤及说明

## node.json
创建node资源的示例

## tools
`certgen.sh` 用于生成证书和密钥的脚本,主要用于两个地方
- 生成cloud和edge的证书,生成的路径在 `/etc/kubeedge/ca`和`/etc/kubeedge/certs`
  ```
  certgen.sh genCertAndKey edge
  ```
- 用于生成`06-secret.yaml`

## deployment.yaml
deployment的模板,用于部署nginx到edge node使用

## deployment-armv7.xml
deployment的模板,用于部署nginx到edge node使用,主要用于armv7架构的edgenode使用。

如果在armv7的边缘节点不使用专属架构的image,有可能会造成pod起不来,pod的状态为`ExitCode:1`
边缘节点对应的docker的日志信息为:`standard_init_linux.go: exec user process caused "exec format error"`
