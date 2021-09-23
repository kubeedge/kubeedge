# RPM 包指南

## 介绍
相对于使用源码包进行安装，RPM包具有事先编译、安装、卸载方便等优势，某些Linux 发行版提供了包管理软件(yum)方便用户对软件包进行管理。

## 准备
1. 使用`yum` 为包管理器的Linux发行版OS(如Fedora、CentOS、openEuler)
2. Kubeedge 在Github 发行的release包，如`https://github.com/kubeedge/kubeedge/archive/refs/tags/v1.8.0.tar.gz`
3. 安装rpmbuild、rpmdev-setuptree命令
   `yum install rpmbuild rpmdevtools`
4. `kubeedge.spec`文件

## 编译
1. 使用`rpmdev-setuptree`命令创建工作目录
   目录结构如下:
   ```bash
   /root/rpmbuild/
   ├── BUILD
   ├── RPMS
   ├── SOURCES
   ├── SPECS
   └── SRPMS
   ```
2. 放置源码包`v1.8.0.tar.gz`到`/root/rpmbuild/SOURCES`目录下
3. 放置`kubeedge.spec`文件到`/root/rpmbuild/SPECS`目录下
4. 使用`rpmbuild`命令进行RPM包构建
   `rpmbuild -ba /root/rpmbuild/SPECS/kubeedge.spec`

## 安装
RPM包编译结束后，会在RPMS目录下出现对应的rpm包:
```bash
/root/rpmbuild/RPMS/
└── x86_64
    ├── kubeedge-cloudcore-1.8.0-1.x86_64.rpm
    ├── kubeedge-edgecore-1.8.0-1.x86_64.rpm
    ├── kubeedge-edgesite-1.8.0-1.x86_64.rpm
    └── kubeedge-keadm-1.8.0-1.x86_64.rpm
```

使用`rpm -ivh xxx.rpm`进行安装对应的rpm组件

## 说明

### kubeedge-cloudcore
此包提供了如下文件:
- cloudcore: 二进制
- admission: 二进制
- csidriver: 二进制
- cloudcore.service: systemd管理文件
- crds: 用户自定义资源文件
- certgen.sh: 证书生成工具
- cloudcore.example.yaml: cloudcore配置文件样例

### kubeedge-edgecore
此包提供了如下文件:
- edgecore: 二进制
- edgecore.service: systemd管理文件
- edgecore.example.yaml: edgecore配置文件样例

### kubeedge-edgesite
此包提供了如下文件:
- edgesite-agent: 二进制
- edgesite-server: 二进制

### kubeedge-keadm
此包提供了如下文件:
- keadm: 二进制
- cloudcore.service: systemd管理文件
- edgecore.service: systemd管理文件
- kubeedge-v{version}-linux-{arch}.tar.gz: 包含keadm安装所需所有文件
    ```bash
	kubeedge-v1.8.0-linux-amd64
	├── cloud
	│   ├── admission
	│   │   └── admission
	│   ├── cloudcore
	│   │   └── cloudcore
	│   └── csidriver
	│       └── csidriver
	├── crds
	│   ├── devices
	│   │   ├── devices_v1alpha1_devicemodel.yaml
	│   │   ├── devices_v1alpha1_device.yaml
	│   │   ├── devices_v1alpha2_devicemodel.yaml
	│   │   └── devices_v1alpha2_device.yaml
	│   ├── reliablesyncs
	│   │   ├── cluster_objectsync_v1alpha1.yaml
	│   │   └── objectsync_v1alpha1.yaml
	│   └── router
	│       ├── router_v1_ruleEndpoint.yaml
	│       └── router_v1_rule.yaml
	├── edge
	│   └── edgecore
	└── version
    ```
- checksum_kubeedge-v{version}-linux-{arch}.tar.gz: checksum文件

## 额外
- 如果使用`keadm`的方式进行部署，则用户只需要安装`kubeedge-keadm`包，因为其已包含了集群建立所需要的所有文件
- 如果使用二进制部署的方式，则需要分别在云端安装`kubeedge-cloudcore`包，在端侧安装`kubeedge-edgecore`包

