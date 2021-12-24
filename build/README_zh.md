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
│       ├── devices_v1alpha2_devicemodel.yaml
│       └── devices_v1alpha2_device.yaml
├── crd-samples
│   └── devices
│       ├── create-device-instance.yaml
│       └── create-device-model.yaml
├── deployment.yaml
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
cloudcore使用kubernetes deployment部署的相关步骤及说明

## crds

## crd-samples

## edge
edgecore使用kubernetes deployment部署的相关步骤及说明（仅测试环境使用）

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

## rpm
制作rpm包相关文件,如`kubeedge.spec`
