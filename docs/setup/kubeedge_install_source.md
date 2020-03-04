# Setup from Source Code

This guide provide steps which can be utilised to install KubeEdge Cloud and Edge side. At this point, we assume that you would have installed the [Pre-Requisite](develop_kubeedge.md#pre-requisite) for Cloud and Edge.

## Setup Cloud Side (KubeEdge Master)

1. Clone KubeEdge

    + Setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH) to clone the KubeEdge repository in the `$GOPATH`.

        ```shell
        git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
        cd $GOPATH/src/github.com/kubeedge/kubeedge
        ```

2. Generate Certificates

   RootCA certificate and a cert/ key pair is required to have a setup for KubeEdge. Same cert/ key pair can be used in both cloud and edge.

    ```bash
    $GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
    ```

    The cert/ key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively, so this command should be run with root or users who have access to those directories. Copy these files to the corresponding edge side server directory.

3. Compile Cloudcore

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

4. Create DeviceModel and Device CRDs.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices

    kubectl create -f devices_v1alpha1_devicemodel.yaml
    kubectl create -f devices_v1alpha1_device.yaml
    ```

5. Create ClusterObjectSync and ObjectSync CRDs which are used in reliable message delivery.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/reliablesyncs
    kubectl create -f cluster_objectsync_v1alpha1.yaml
    kubectl create -f objectsync_v1alpha1.yaml
    ```

6. Copy cloudcore binary

    At this point, cloudcore can be copied to a new directory.

    Copy cloudcore binary

     ```shell
     # copy $GOPATH/src/github.com/kubeedge/kubeedge/_output/local/bin/cloudcore to `~/kubeedge/`
     mkdir ~/kubeedge/
     cp cloudcore ~/kubeedge/
     ```

     **Note:**  `~/kubeedge/` dir is an example, in the following examples we continue to  use `~/kubeedge/` as the binary startup directory. You can move `cloudcore` or  `edgecore` binary to anywhere.


+ (**Optional**) Run `admission`, this feature is still being evaluated.
    please read the docs in [install the admission webhook](../../build/admission/README.md)

## Setup Edge Node (KubeEdge Worker Node)

1. Clone KubeEdge

    + Setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH) to clone the KubeEdge repository in the `$GOPATH`.

        ```shell
        git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
        ```

2. Compile Edgecore

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge
    make all WHAT=edgecore
    ```

    KubeEdge can also be cross compiled to run on ARM based processors.
    Please follow the instructions given below or click [Cross Compilation](cross-compilation.md) for detailed instructions.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make cross_build
    ```

    KubeEdge can also be compiled with a small binary size. Please follow the below steps to build a binary of lesser size:

    ```shell
    apt-get install upx-ucl
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make edge_small_build
    ```

    **Note:** If you are using the smaller version of the binary, it is compressed using upx, therefore the possible side effects of using upx compressed binaries like more RAM usage,
    lower performance, whole code of program being loaded instead of it being on-demand, not allowing sharing of memory which may cause the code to be loaded to memory
    more than once etc. are applicable here as well.

    **Note:** If you don't want to compile, you may perform the next step

    + Download KubeEdge from [Releases](https://github.com/kubeedge/kubeedge/releases)

        Download `kubeedge-$VERSION-$OS-$ARCH.tar.gz` from above link. It would contain Edgecore and the configuration files.

3. Copy edgecore binary

+ Copy edgecore file in a new directory

    ```shell
    cp $GOPATH/src/github.com/kubeedge/kubeedge/_output/local/bin/edgecore ~/kubeedge/
    cd ~/kubeedge
    ```
