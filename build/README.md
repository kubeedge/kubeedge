# build description

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
Relevant steps and instructions for cloudcore deployment with kubernetes deployment


## crds

## crd-samples

## edge
Relevant steps and instructions for edgecore deployment with kubernetes deployment (Only for test)

## node.json
Example of creating node resources

## tools
`certgen.sh` used to generate certificates and keys, mainly used in two places
- Generate cloud and edge certificates, the generated path is in `/etc/kubeedge/ca` and `/etc/kubeedge/certs`
  ```
  certgen.sh genCertAndKey edge
  ```
- used to generate `06-secret.yaml`

## deployment.yaml
deployment template, used to deploy nginx to edge node

## rpm
Make rpm package related files, such as `kubeedge.spec`
