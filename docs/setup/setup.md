# Setup KubeEdge from sourcecode

## Abstract
KubeEdge is composed  of cloud and edge parts. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. So if we want to setup kubeedge, we need to setup kubernetes cluster, cloud side and edge side.

+ on cloud side, we need to install docker, kubernetes cluster and cloudcore.
+ on edge side, we need to install docker, mqtt and edgecore.

## Prerequisites

+ [Install docker on cloud and edge side](https://docs.docker.com/install/)

    you can also run other runtime, such as [containerd](https://github.com/containerd/containerd)

+ [Install kubeadm/kubectl on cloud side](https://kubernetes.io/docs/setup/independent/install-kubeadm/)

+ [Creating kubernetes cluster with kubeadm on cloud side](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)

+ **Go** The minimum required go version is 1.12. You can install this version by using [this website.](https://golang.org/dl/) 

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
# $GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
```

The cert/key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively. We need to copy these files to the corresponding edge side server directory.

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
+ Create device model and device CRDs.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices
    kubectl create -f devices_v1alpha1_devicemodel.yaml
    kubectl create -f devices_v1alpha1_device.yaml
    ```

+ Copy cloudcore binary and config file 

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud
    # run edge controller
    # `conf/` should be in the same directory as the cloned KubeEdge repository
    # verify the configurations before running cloud(cloudcore)
    mkdir -p ~/cmd/conf
    cp cloudcore ~/cmd/
    cp -rf conf/* ~/cmd/conf/
    ```
    **Note** `~/cmd/` dir is an example, in the following examples we continue to use `~/cmd/` as the binary startup directory. You can move `cloudcore` or `edgecore` binary to anywhere, but you need to create `conf` dir in the same directory as binary.
    
+ Set cloudcore config file

    ```shell
    cd ~/cmd/conf
    vim controller.yaml
    ```

    verify the configurations before running `cloudcore`
    
    ```
    controller:
      kube:
        master:     # kube-apiserver address (such as:http://localhost:8080)
        namespace: ""
        content_type: "application/vnd.kubernetes.protobuf"
        qps: 5
        burst: 10
        node_update_frequency: 10
        kubeconfig: "~/.kube/config"   #Enter path to kubeconfig file to enable https connection to k8s apiserver, if master and kubeconfig are both set, master will override any value in kubeconfig.
    cloudhub:
      protocol_websocket: true # enable websocket protocol
      port: 10000 # open port for websocket server
      protocol_quic: true # enable quic protocol
      quic_port: 10001 # open prot for quic server
      max_incomingstreams: 10000 # the max incoming stream for quic server
      enable_uds: true # enable unix domain socket protocol
      uds_address: unix:///var/lib/kubeedge/kubeedge.sock # unix domain socket address
      address: 0.0.0.0
      ca: /etc/kubeedge/ca/rootCA.crt
      cert: /etc/kubeedge/certs/edge.crt
      key: /etc/kubeedge/certs/edge.key
      keepalive-interval: 30
      write-timeout: 30
      node-limit: 10
    devicecontroller:
      kube:
        master:        # kube-apiserver address (such as:http://localhost:8080)
        namespace: ""
        content_type: "application/vnd.kubernetes.protobuf"
        qps: 5
        burst: 10
        kubeconfig: "~/.kube/config" #Enter path to kubeconfig file to enable https connection to k8s apiserver,if master and kubeconfig are both set, master will override any value in kubeconfig.
    ```
    cloudcore default supports https connection to Kubernetes (required version is 1.15+) apiserver, so you need to check whether the path for `controller.kube.kubeconfig` and `devicecontroller.kube.kubeconfig` exist, but if `master` and `kubeconfig` are both set, `master` will override any value in kubeconfig. 
    Check whether the cert files for `cloudhub.ca`, `cloudhub.cert`,`cloudhub.key` exist.

+ Run cloudcore 

    ```shell
    cd ~/cmd/
    nohup ./cloudcore & 
    ```
    
#### Deploy the edge node 
We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node.

+ Copy the `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` file and change `metadata.name` to the name of the edge node
    
    ```shell
        mkdir ~/cmd/yaml
        cp $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json ~/cmd/yaml
    ```
    
+ Make sure role is set to edge for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `labels` tag of `metadata`.
+ Please ensure to add the label `node-role.kubernetes.io/edge` to the `build/node.json` file.

    ```script
    {
      "kind": "Node",
      "apiVersion": "v1",
      "metadata": {
        "name": "edge-node",
        "labels": {
          "name": "edge-node",
          "node-role.kubernetes.io/edge": ""
        }
      }
    }
    ```
    **Note: you need to remember `metadata.name` , because edgecore need it**.
+ If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.
+ Deploy edge node (**you must run on cloud side**)

    ```shell
    kubectl apply -f ~/cmd/yaml/node.json
    ```
+ Transfer certificate files to the edge node, because `edgecore` use these certificate files to connection `cloudcore` 

### Setup edge side

#### Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```
#### Run Edge

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
    
+ Run mqtt on edge side

    ```shell
    # run mosquitto
    mosquitto -d -p 1883
    # or run emqx edge
    # emqx start
    ``` 

+ Set edgecore config file
    
    ```shell
    mkdir ~/cmd/conf
    cp $GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/* ~/cmd/conf
    vim ~/cmd/conf/edge.yaml
    ```
    
    **Note:** `~/cmd/` dir is also an example as well as `cloudcore`, `conf/` should be in the same directory as edgecore binary, 
    
    verify the configurations before running `edgecore`
    
    ```
    mqtt:
        server: tcp://127.0.0.1:1883 # external mqtt broker url.
        internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
        mode: 0 # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker
    enable only.
        qos: 0 # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
        retain: false # if the flag set true, server will store the message and can be delivered to future subscribers.
        session-queue-size: 100 # A size of how many sessions will be handled. default to 100.
    
    edgehub:
        websocket:
            url: wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events 
            certfile: /etc/kubeedge/certs/edge.crt
            keyfile: /etc/kubeedge/certs/edge.key
            handshake-timeout: 30 #second
            write-deadline: 15 # second
            read-deadline: 15 # second
        quic:
            url: 127.0.0.1:10001
            cafile: /etc/kubeedge/ca/rootCA.crt
            certfile: /etc/kubeedge/certs/edge.crt
            keyfile: /etc/kubeedge/certs/edge.key
            handshake-timeout: 30 #second
            write-deadline: 15 # second
            read-deadline: 15 # second
        controller:
            protocol: websocket # websocket, quic
            heartbeat: 15  # second
            project-id: e632aba927ea4ac2b575ec1603d56f10
            node-id: edge-node
    
    edged:
        register-node-namespace: default
        hostname-override: edge-node
        interface-name: eth0
        edged-memory-capacity-bytes: 7852396000
        node-status-update-frequency: 10 # second
        device-plugin-enabled: false
        gpu-plugin-enabled: false
        image-gc-high-threshold: 80 # percent
        image-gc-low-threshold: 40 # percent
        maximum-dead-containers-per-container: 1
        docker-address: unix:///var/run/docker.sock
        runtime-type: docker
        remote-runtime-endpoint: unix:///var/run/dockershim.sock
        remote-image-endpoint: unix:///var/run/dockershim.sock
        runtime-request-timeout: 2
        podsandbox-image: kubeedge/pause:3.1 # kubeedge/pause:3.1 for x86 arch , kubeedge/pause-arm:3.1 for arm arch, kubeedge/pause-arm64 for arm64 arch
        image-pull-progress-deadline: 60 # second
        cgroup-driver: cgroupfs
        node-ip: ""
        cluster-dns: ""
        cluster-domain: ""
        
        mesh:
            loadbalance:
                strategy-name: RoundRobin
    ```
    + If you have run mosquitto on your edge host, please set `mqtt.mode` to `2`. 
    + To use KubeEdge in double mqtt or external mode (set `mqtt.mode` to 1), you need to make sure that [mosquitto](https://mosquitto.org/) or [emqx edge](https://www.emqx.io/downloads/edge) is installed on the edge node as an MQTT Broker. 
    + Check whether the cert files for `edgehub.websocket.certfile` and `edgehub.websocket.keyfile`  exist.
    + Check whether the cert files for `edgehub.quic.certfile` ,`edgehub.quic.keyfile` and `edgehub.quic.cafile` exist. If those files not exist, you need to copy them from cloud side. 
    + Check `edged.podsandbox-image`  
        + `kubeedge/pause-arm:3.1` for arm arch
        + `kubeedge/pause-arm64:3.1` for arm64 arch
        + `kubeedge/pause:3.1` for x86 arch
    + Update the IP address of the master in the `edgehub.websocket.url` field. You need set cloudcore ip address.
    + If you use quic protocol, please update the IP address of the master in the `edgehub.quic.url` field. You need set cloudcore ip address.
    + replace `edge-node` with edge node name in edge.yaml for the below fields :
        + `websocket:URL`
        + `controller:node-id`
        + `edged:hostname-override`

+ Run edgecore

    ```shell

    cp $GOPATH/src/github.com/kubeedge/kubeedge/edge/edgecore ~/cmd/ 
    cd ~/cmd
    ./edgecore
    # or
    nohup ./edgecore > edgecore.log 2>&1 &
    ```
    **Note:** Please run edge using the users who have root permission.

+ Run edgecore with systemd

    It is also possible to start the edgecore with systemd. If you want, you could use the example systemd-unit-file. The following command will show you how to setup this:

    ```shell
    sudo ln build/tools/edge.service /etc/systemd/system/edge.service
    sudo systemctl daemon-reload
    sudo systemctl start edgecore
    ```
    
    If you also want also an autostart, you have to execute this, too:
    
    ```shell
    sudo systemctl enable daemon-reload
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

**Note:** Currently, for edge node, we must use hostPort in the Pod container spec so that the pod comes up normally, or the pod will be always in ContainerCreating status. The hostPort must be equal to containerPort and can not be 0.

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
make edge_integration_test
```

### Details and use cases of integration test framework

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of intergration test framework for KubeEdge.
