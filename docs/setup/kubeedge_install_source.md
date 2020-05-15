# Setup from Source Code

This guide provide steps which can be utilised to install KubeEdge Cloud and Edge side. At this point, we assume that you would have installed the [Pre-Requisite](develop_kubeedge.md#pre-requisite) for Cloud and Edge.

## Setup Cloud Side (KubeEdge Master)

### Clone KubeEdge

Setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH) to clone the KubeEdge repository in the `$GOPATH`.

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### Generate Certificates (Required for pre 1.3 releases)

**Note: KubeEdge v1.3 needs to skip this step Generate Certificates and clean up the local certificates in `/etc/kubeedge/ca` and `/etc/kubeedge/certs`. Because KubeEdge v1.3 has added the feature of generating certificates automatically.**

RootCA certificate and a cert/ key pair is required to have a setup for KubeEdge. Same cert/ key pair can be used in both cloud and edge.

```bash
$GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
```

The cert/ key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively, so this command should be run with root or users who have access to those directories. Copy these files to the corresponding edge side server directory.

#### Generate Certificates for support `kubectl logs` command

+ First , you need to make sure you can find the kubernetes ca.crt and ca.key files. if you start up your kubernetes cluster by `kubeadmin`. 
those files will be in `/etc/kubernetes/pki/` dir.

+ Second , set `CLOUDCOREIPS` env, The environment variable is set to specify the IP addresses of all cloudcore
this is an example:
```bash
export CLOUDCOREIPS="172.20.12.45 172.20.12.46"
```

+ third

```bash
$GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh stream 
```

+ fourth

Run the following command on the host on which each apiserver runs:
** Note: ** You need to set the cloudcoreip variable first
```
iptables -t nat -A OUTPUT -p tcp  --dport 10350 -j DNAT --to {cloudcoreip}:10003
```

### Compile Cloudcore

+ Make sure a C compiler is installed on your host. The installation is tested with `gcc` and `clang`.

  ```shell
  gcc --version
  ```

+ Build cloudcore

  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/
  make all WHAT=cloudcore
  ```

 **Note:** If you don't want to compile, you may perform the below step

+ Download KubeEdge (latest or stable version) from [Releases](https://github.com/kubeedge/kubeedge/releases)

  Download `kubeedge-$VERSION-$OS-$ARCH.tar.gz` from above link. It contains Cloudcore and the configuration files.

### Create DeviceModel and Device CRDs.

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices

kubectl create -f devices_v1alpha1_devicemodel.yaml
kubectl create -f devices_v1alpha1_device.yaml
```

### Create ClusterObjectSync and ObjectSync CRDs which are used in reliable message delivery.

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/reliablesyncs
kubectl create -f cluster_objectsync_v1alpha1.yaml
kubectl create -f objectsync_v1alpha1.yaml
```

### Copy cloudcore binary

At this point, cloudcore can be copied to a new directory.

Copy cloudcore binary

```shell
# copy $GOPATH/src/github.com/kubeedge/kubeedge/_output/local/bin/cloudcore to `~/kubeedge/`
mkdir ~/kubeedge/
cp cloudcore ~/kubeedge/
```

**Note:**  `~/kubeedge/` dir is an example, in the following examples we continue to  use `~/kubeedge/` as the binary startup directory. You can move `cloudcore` or  `edgecore` binary to anywhere.


### (**Optional**) Run `admission`

This feature is still being evaluated, please read the docs in [install the admission webhook](../../build/admission/README.md)

## Setup Edge Node (KubeEdge Worker Node)

### Clone KubeEdge

Setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH) to clone the KubeEdge repository in the `$GOPATH`.

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
```

### Compile Edgecore

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge
make all WHAT=edgecore
```

KubeEdge can also be cross compiled to run on ARM based processors.
Please follow the instructions given below or click [Cross Compilation](cross-compilation.md) for detailed instructions.

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
make crossbuild
```

KubeEdge can also be compiled with a small binary size. Please follow the below steps to build a binary of lesser size:

```shell
apt-get install upx-ucl
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
make smallbuild
```

**Note:** If you are using the smaller version of the binary, it is compressed using upx, therefore the possible side effects of using upx compressed binaries like more RAM usage,
lower performance, whole code of program being loaded instead of it being on-demand, not allowing sharing of memory which may cause the code to be loaded to memory
more than once etc. are applicable here as well.

**Note:** If you don't want to compile, you may perform the next step

+ Download KubeEdge from [Releases](https://github.com/kubeedge/kubeedge/releases)

  Download `kubeedge-$VERSION-$OS-$ARCH.tar.gz` from above link. It would contain Edgecore and the configuration files.

### Copy edgecore binary

Copy edgecore file in a new directory

```shell
cp $GOPATH/src/github.com/kubeedge/kubeedge/_output/local/bin/edgecore ~/kubeedge/
```
