# Setup KubeEdge from sourcecode

## Abstract
KubeEdge is composed  of cloud and edge parts. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. So if we want to setup kubeedge, we need to setup kubernetes cluster, cloud side and edge side.

+ on cloud side, we need to install docker, kubernetes cluster and cloudcore.
+ on edge side, we need to install docker, mqtt and edgecore.

## Prerequisites

+ **Go dependency** and **Kubernetes compatibility** please refer to [compatibility-matrix](https://github.com/kubeedge/kubeedge#compatibility-matrix).

### Cloud side

+ [Install golang](https://golang.org/dl/)

+ [Install docker](https://docs.docker.com/install/), or other runtime, such as [containerd](https://github.com/containerd/containerd)

+ [Install kubeadm/kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/)

+ [Creating kubernetes cluster with kubeadm](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)


### Edge side

+ [Install golang](https://golang.org/dl/)

+ [Install docker](https://docs.docker.com/install/), or other runtime, such as [containerd](https://github.com/containerd/containerd)

+ [Install mosquitto](https://mosquitto.org/download/)

**Note:** 
+ Do not install **kubelet** and **kube-proxy** on edge side
+ If you use kubeadm to install kubernetes, the `Kubeadm init` command can not be followed by the "--experimental-upload-certs" or 
"--upload-certs" flag

## Run KubeEdge

### Setup cloud side

#### Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

#### Generate Certificates

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

```bash
$GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
```

The cert/key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively, so this command should be run with root or users who have access to those directories. We need to copy these files to the corresponding edge side server directory.

#### Run as a binary

+ Firstly, make sure gcc is already installed on your host. You can verify it via:

    ```shell
    gcc --version
    ```

+ Build cloudcore 

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/
    make all WHAT=cloudcore
    ```
  
+ Create DeviceModel and Device CRDs.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices
    kubectl create -f devices_v1alpha1_devicemodel.yaml
    kubectl create -f devices_v1alpha1_device.yaml
    ```
  
+ Create ClusterObjectSync and ObjectSync CRDs which used in reliable message delivery.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/reliablesyncs
    kubectl create -f cluster_objectsync_v1alpha1.yaml
    kubectl create -f objectsync_v1alpha1.yaml
    ```

+ Copy cloudcore binary

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud
    mkdir -p ~/cmd
    cp cloudcore ~/cmd/
    ```
    **Note** `~/cmd/` dir is an example, in the following examples we continue to use `~/cmd/` as the binary startup directory. You can move `cloudcore` or `edgecore` binary to anywhere.
    
+ Create and set cloudcore config file

    ```shell
    # the default configration file path is '/etc/kubeedge/config/cloudcore.yaml'
    # also you can specify it anywhere with '--config'
    mkdir -p /etc/kubeedge/config/ 
  
    # create a minimal configuration with command `~/cmd/cloudcore --minconfig`
    # or a full configuration with command `~/cmd/cloudcore --defaultconfig`
    ~/cmd/cloudcore --minconfig > /etc/kubeedge/config/cloudcore.yaml 
    vim /etc/kubeedge/config/cloudcore.yaml 
  
    ```

    verify the configurations before running `cloudcore`
    
    ```
    apiVersion: cloudcore.config.kubeedge.io/v1alpha1
    kind: CloudCore
    kubeAPIConfig:
      kubeConfig: /root/.kube/config #Enter absolute path to kubeconfig file to enable https connection to k8s apiserver,if master and kubeconfig are both set, master will override any value in kubeconfig.
      master: "" # kube-apiserver address (such as:http://localhost:8080)
    modules:
      cloudhub:
        nodeLimit: 10
        tlsCAFile: /etc/kubeedge/ca/rootCA.crt
        tlsCertFile: /etc/kubeedge/certs/edge.crt
        tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
        unixsocket:
          address: unix:///var/lib/kubeedge/kubeedge.sock # unix domain socket address
          enable: true # enable unix domain socket protocol
        websocket:
          address: 0.0.0.0
          enable: true # enable websocket protocol
          port: 10000 # open port for websocket server
    ```
    cloudcore use https connection to Kubernetes apiserver as default, so you should make sure the `kubeAPIConfig.kubeConfig` exist, but if `master` and `kubeConfig` are both set, `master` will override any value in kubeconfig.
    Check whether the cert files for `modules.cloudhub.tlsCAFile`, `modules.cloudhub.tlsCertFile`,`modules.cloudhub.tlsPrivateKeyFile` exists.

+ Run cloudcore

    ```shell
    cd ~/cmd/
    nohup ./cloudcore &
    ```

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

+ (**Optional**)Run `admission`, this feature is still being evaluated.
    please read the docs in [install the admission webhook](../../build/admission/README.md)
    
#### Deploy the edge node
Edge node can be registered automatically. But if you want to deploy edge node manually, [here](./deploy-edge-node.md) is an example.

### Setup edge side

+ Transfer certificate files from cloud side to edge node, because `edgecore` use these certificate files to connection `cloudcore` 

#### Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```
#### Run Edge

##### Configuring MQTT mode

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
1) internalMqttMode: internal mqtt broker is enabled.
2) bothMqttMode: internal as well as external broker are enabled.
3) externalMqttMode: only external broker is enabled.

To use KubeEdge in double mqtt or external mode, you need to make sure that [mosquitto](https://mosquitto.org/) or [emqx edge](https://www.emqx.io/downloads/edge) is installed on the edge node as an MQTT Broker.

##### Run as a binary
+ Build Edge

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge
    make all WHAT=edgecore
    ```

    KubeEdge can also be cross compiled to run on ARM based processors.
    Please follow the instructions given below or click [Cross Compilation](cross-compilation.md) for detailed instructions.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make edge_cross_build
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
    
+ Copy edgecore binary

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    mkdir -p ~/cmd
    cp edgecore ~/cmd/
    ```
    **Note:** `~/cmd/` dir is also an example as well as `cloudcore`
    
+ Create and set edgecore config file

    ```shell
    # the default configration file path is '/etc/kubeedge/config/edgecore.yaml'
    # also you can specify it anywhere with '--config'
    mkdir -p /etc/kubeedge/config/ 
    
    # create a minimal configuration with command `~/cmd/edgecore --minconfig`
    # or a full configuration with command `~/cmd/edgecore --defaultconfig`
    ~/cmd/edgecore --minconfig > /etc/kubeedge/config/edgecore.yaml 
    vim /etc/kubeedge/config/edgecore.yaml 
    ```
    verify the configurations before running `edgecore`

    ```
    apiVersion: edgecore.config.kubeedge.io/v1alpha1
    database:
      dataSource: /var/lib/kubeedge/edgecore.db
    kind: EdgeCore
    modules:
      edged:
        cgroupDriver: cgroupfs
        clusterDNS: ""
        clusterDomain: ""
        devicePluginEnabled: false
        dockerAddress: unix:///var/run/docker.sock
        gpuPluginEnabled: false
        hostnameOverride: $your_hostname
        interfaceName: eth0
        nodeIP: $your_ip_address
        podSandboxImage: kubeedge/pause:3.1  # kubeedge/pause:3.1 for x86 arch , kubeedge/pause-arm:3.1 for arm arch, kubeedge/pause-arm64 for arm64 arch
        remoteImageEndpoint: unix:///var/run/dockershim.sock
        remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
        runtimeType: docker
      edgehub:
        heartbeat: 15  # second
        tlsCaFile: /etc/kubeedge/ca/rootCA.crt
        tlsCertFile: /etc/kubeedge/certs/edge.crt
        tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
        websocket:
          enable: true
          handshakeTimeout: 30  # second
          readDeadline: 15  # second
          server: 127.0.0.1:10000  # cloudcore address
          writeDeadline: 15  # second
      eventbus:
        mqttMode: 2  # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker
        mqttQOS: 0  # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
        mqttRetain: false  # if the flag set true, server will store the message and can be delivered to future subscribers.
        mqttServerExternal: tcp://127.0.0.1:1883  # external mqtt broker url.
        mqttServerInternal: tcp://127.0.0.1:1884  # internal mqtt broker url.
    ```
    + Check `modules.edged.podSandboxImage`  
        + `kubeedge/pause-arm:3.1` for arm arch
        + `kubeedge/pause-arm64:3.1` for arm64 arch
        + `kubeedge/pause:3.1` for x86 arch
    + Check whether the cert files for `modules.edgehub.tlsCaFile` and `modules.edgehub.tlsCertFile` and `modules.edgehub.tlsPrivateKeyFile` exists.
    If those files not exist, you need to copy them from cloud side. 
    + Check `modules.edgehub.websocket.server`. It should be your cloudcore ip address.

+ Run edgecore

    ```shell
    # run mosquitto
    mosquitto -d -p 1883
    # or run emqx edge
    # emqx start

    cd ~/cmd
    ./edgecore
    # or
    nohup ./edgecore > edgecore.log 2>&1 &
    ```
    **Note:** Please run edgecore using the users who have root permission.

+ Run edgecore with systemd

    It is also possible to start the edgecore with systemd. If you want, you could use the example systemd-unit-file. The following command will show you how to setup this:

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

#### Check status

After the Cloud and Edge parts have started, you can use below command to check the edge node status.

```shell
kubectl get nodes
```

Please make sure the status of edge node you created is **ready**.

## Deploy Application on cloud side

Try out a sample application deployment by following below steps.

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

**Note:** Currently, for applications running on edge nodes, we don't support `kubectl logs` and `kubectl exec` commands(will support in future release), support pod to pod communication running on **edge nodes in same subnet** using edgemesh.

Then you can use below command to check if the application is normally running.

```shell
kubectl get pods
```

## Run Tests

### Run Edge Unit Tests

 ```shell
 make edge_test
 ```

 To run unit tests of a package individually.

 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge/edge
 cd <path to package to be tested>
 go test -v
 ```

### Run Edge Integration Tests

```shell
make integrationtest
```

### Details and use cases of integration test framework

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of intergration test framework for KubeEdge.
