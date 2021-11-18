# RPM Packaging Guide

## Introduce
Compared with install kubeedge from source code, RPM packages can let user easy to compile, install and uninstall.

Some Linux distributors provide package management software(yum for example) for users to manage software packages.

## Preparation
1. OS with package management software `yum`(Such as Fedora, CentOS, openEuler)
2. Release tarball of kubeedge in Github. For example: `https://github.com/kubeedge/kubeedge/archive/refs/tags/v1.8.0.tar.gz`
3. `rpmbuild`, `rpmdev-setuptree` command installed
    `yum install rpmbuild rpmdevtools`
4. `kubeedge.spec` file

## Compilation
1. Use `rpmdev-setuptree` command to build working directories
   ```bash
   /root/rpmbuild/
   ├── BUILD
   ├── RPMS
   ├── SOURCES
   ├── SPECS
   └── SRPMS
   ```
2. Put sorce code tarball into `/root/rpmbuild/SOURCES`
3. Put `kubeedge.spec` file into `/root/rpmbuild/SPECS`
4. Use `rpmbuild` to build RPM package
   `rpmbuild -ba /root/rpmbuild/SPECS/kubeedge.spec`

## Installation
After compilation finished, the RPMs will show up in `RPMS` folder
```bash
/root/rpmbuild/RPMS/
└── x86_64
    ├── kubeedge-cloudcore-1.8.0-1.x86_64.rpm
    ├── kubeedge-edgecore-1.8.0-1.x86_64.rpm
    ├── kubeedge-edgesite-1.8.0-1.x86_64.rpm
    └── kubeedge-keadm-1.8.0-1.x86_64.rpm
```

Use `rpm -ivh xxx.rpm` to install the correspondingly rpm package:

## Note

### kubeedge-cloudcore
This package provides:
- cloudcore: binary
- admission: binary
- csidriver: binary
- cloudcore.service: unit file used for systemd
- crds: CRDS used for k8s
- certgen.sh: tools for generating certificates
- cloudcore.example.yaml: sample config file for cloudcore

### kubeedge-edgecore
This package provides:
- edgecore: binary
- edgecore.service: unit file used for systemd
- edgecore.example.yaml: sample config file for edgecore

### kubeedge-edgesite
This package provides:
- edgesite-agent: binary
- edgesite-server: binary

### kubeedge-keadm
This package provides:
- keadm: binary
- cloudcore.service: unit file used for systemd
- edgecore.service: unit file used for systemd
- kubeedge-v{version}-linux-{arch}.tar.gz: all files needed by keadm
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
- checksum_kubeedge-v{version}-linux-{arch}.tar.gz: checksum file for tarball


## Additional
- When use `keamd` to deploy the cluster, user need to install `kubeedge-keadm` package only since it already contain everything that keadm need.
- When use binary to deploy the cluster, user need to install `kubeedge-cloudcore` package on the cloud side and `kubeedge-edgecore` package on the edge side separately.

