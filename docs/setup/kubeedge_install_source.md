# Setup from Source Code

This guide would provide steps which can be utilised to install KubeEdge Cloud and Edge side. At this point, we assume that you would have installed the Pre-Requisites for Cloud and Edge.

## Setup Cloud Side (KubeEdge Master Node)

1. Clone KubeEdge

    + Before this it might be a good idea to setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH)

        ```shell
        git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
        cd $GOPATH/src/github.com/kubeedge/kubeedge
        ```

2. Generate Certificates

   RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

    ```bash
    $GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
    ```

    The cert/key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively, so this command should be run with root or users who have access to those directories. We need to copy these files to the corresponding edge side server directory.

3. Compile Cloudcore


   + Firstly, make sure gcc is already installed on your host. You can verify it via:

     ```shell
        gcc --version
        ```

   + Build cloudcore 

        ```shell
        cd $GOPATH/src/github.com/kubeedge/kubeedge/
        make all WHAT=cloudcore
        ```

    **Note:** If you don't want to compile, you may perform the below step

    + Download KubeEdge from [Releases](https://github.com/kubeedge/kubeedge/releases)   
    
        Download KubeEdge-$VERSION-$OS-$ARCH.tar.gz. from above link. It would contain Cloudcore and the configuration files.

4. Create device model and device CRDs.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices
    kubectl create -f devices_v1alpha1_devicemodel.yaml
    kubectl create -f devices_v1alpha1_device.yaml
    ```
5. Copy cloudcore binary/ Setup as a systemctl

    At this point, cloudcore can be copied to a new directory or can be setup as a systemctl service. We also need to edit the configuration files to setup how cloudcore would run.

   + Run cloudcore with systemd

        It is also possible to start the cloudcore with systemd. If you want, you could use the example systemd-unit-file. The following command will show you how to setup this:

        ```shell
        sudo ln build/tools/cloudcore.service /etc/systemd/system/cloudcore.service
        sudo systemctl daemon-reload
        sudo systemctl start cloudcore
        ```
    
        **Note:** Please fix __ExecStart__ path in cloudcore.service. Do __NOT__ use relative path, use absoulte path instead.

        If you also want also an autostart, you have to execute this, too:

        ```shell
        sudo systemctl enable cloudcore
        ```

   + Copy cloudcore binary and config file 

        ```shell
        cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud
        # run cloudcore controller
        # `conf/` should be in the same directory as the cloned KubeEdge repository
        # verify the configurations before running cloud(cloudcore)
        mkdir -p ~/cmd/conf
        cp cloudcore ~/cmd/
        cp -rf conf/* ~/cmd/conf/
        ```
    
        **Note** `~/cmd/` dir is an example, in the following examples we continue to use `~/cmd/` as the binary startup directory. You can move `cloudcore` or `edgecore` binary to anywhere, but you need to create `conf` dir in the same directory as binary.

+ (**Optional**) Run `admission`, this feature is still being evaluated.
    please read the docs in [install the admission webhook](../../build/admission/README.md)

## Setup Edge Node (KubeEdge Worker Node)

1. Clone KubeEdge

    + Before this it might be a good idea to setup [$GOPATH ](https://github.com/golang/go/wiki/SettingGOPATH)

        ```shell
        git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
        cd $GOPATH/src/github.com/kubeedge/kubeedge
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
    
        Download KubeEdge-version-linux-$platform.tar.gz from above link. It would contain Edgecore and the configuration files.

3. Copy edgecore binary/ Setup as a systemctl

 + Run edgecore with systemd

    It is also possible to start the edgecore with systemd. If you want, you could use the example systemd-unit-file. 

    ```shell
    sudo ln build/tools/edgecore.service /etc/systemd/system/edgecore.service
    sudo systemctl daemon-reload
    sudo systemctl start edgecore
    ```
    **Note:** Please fix __ExecStart__ path in edgecore.service. Do __NOT__ use relative path, use absoulte path instead.

    If you also want also an autostart, you have to execute this, too:

    ```shell
    sudo systemctl enable edgecore
    ```

 + Copy edgecore file and sample config in a new directory

    ```
    cp $GOPATH/src/github.com/kubeedge/kubeedge/edge/edgecore ~/cmd/
    cd ~/cmd
    ```

