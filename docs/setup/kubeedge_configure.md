# KubeEdge Configuration

KubeEdge requires configuration on both [Cloud side (KubeEdge Master Node)](#configuration-cloud-side-(kubeedge-master-node)) and [Edge side (KubeEdge Worker Node)](#configuration-edge-side-(kubeedge-worker-node))

## Configuration Cloud side (KubeEdge Master Node)

Setting up cloud side requires two steps 

1. Modification of the configuration files
2. Adding the edge nodes (KubeEdge Worker Node) on the Cloud side KubeEdge Master Node).

### Modification of the configuration files

Cloudcore requires three configuration file as of now. However this might change in future as per the [Issue 1171](https://github.com/kubeedge/kubeedge/issues/1171)

1. controller.yaml
2. logging.yaml
3. modules.yaml


Set cloudcore config file

```shell
    cd ~/cmd/conf
    vim controller.yaml
```

Verify the configurations before running `cloudcore`
    
```yaml
controller:
    kube:
        master:     # kube-apiserver address (such as:http://localhost:8080)
        namespace: ""
        content_type: "application/vnd.kubernetes.protobuf"
        qps: 5
        burst: 10
        node_update_frequency: 10
        kubeconfig: "/root/.kube/config"   #Enter absolute path to kubeconfig file to enable https connection to k8s apiserver, if master and kubeconfig are both set, master will override any value in kubeconfig.
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
        kubeconfig: "/root/.kube/config" #Enter absolute path to kubeconfig file to enable https connection to k8s apiserver,if master and kubeconfig are both set, master will override any value in kubeconfig.
```

cloudcore default supports https connection to Kubernetes apiserver, so you need to check whether the path for `controller.kube.kubeconfig` and `devicecontroller.kube.kubeconfig` exist, but if `master` and `kubeconfig` are both set, `master` will override any value in kubeconfig.

Check whether the cert files for `cloudhub.ca`, `cloudhub.cert`,`cloudhub.key` exist.

#### Modification in controller.yaml

In the controller.yaml, modify the below settings.

1. `controller.kube.master` : This would be kube-apiserver address. It might be 

    ```
    https://<your_hostname>:6443
    ```
    or
    ```
    http://<your_hostname>:8080
    ```
    based on your kubernetes configuration. `your_hostname` should be replaced with the IP Address of your hostname.

2. `controller.kube.kubeconfig` and `devicecontroller.kube.kubeconfig` : This would be the path to your kubeconfig file. It might be either

    ```
    /root/.kube/config
    ```
    or
    ```
    /home/<your_username>/.kube/config
    ```
    depending on where you have setup your kubernetes by performing the below step:

    ```
    To start using your cluster, you need to run the following as a regular user:

    mkdir -p $HOME/.kube
    sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
    sudo chown $(id -u):$(id -g) $HOME/.kube/config
    ```

### Adding the edge nodes (KubeEdge Worker Node) on the Cloud side (KubeEdge Master Node)

We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node.

+ Copy the `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` file and change `metadata.name` to the name of the edge node

    ```shell
        mkdir -p ~/cmd/yaml
        cp $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json ~/cmd/yaml
    ```

    Node.json
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

#### Modification in node.json

1. metadata.name : This is the name of your edge node device. (Referring the edge node configuration would make it more clear). 
 
    **Note: you need to remember `metadata.name` , because edgecore needs it**.

2. metadata.labels: Labels are the name by which you can remember a set of nodes.

    + Make sure role is set to `edge` for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `labels` tag of `metadata`.
    + Please ensure to add the label `node-role.kubernetes.io/edge` to the `build/node.json` file.
    
    + If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.

    + This can be checked by running the below command on Cloud side

        ```
        kubectl get nodes --show-labels
        ```

#### Deploy edge node (Run on cloud side)

```shell
    kubectl apply -f ~/cmd/yaml/node.json
```

#### Check if the certificates are created (Run on cloud side)

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

Ideally, when you setup KubeEdge on the cloud side, certificates would have been generated. Check `/etc/kubeedge`. 

If not, perform the following below steps

```shell
$GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
```

or 

```shell
wget -L https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/tools/certgen.sh
# make script executable
chmod +x certgen.sh
bash -x ./certgen.sh genCertAndKey edge
```
The cert/key will be generated in the /etc/kubeedge/ca and /etc/kubeedge/certs respectively, so this command should be run with root or users who have access to those directories. 
We need to copy these files to the corresponding edge side server directory.

We can create the `certs.tgz` by 

```shell
cd /etc/kubeedge
tar -cvzf certs.tgz certs/
```

#### Transfer certificate file from the cloud side to edge side

Transfer certificate files to the edge node, because `edgecore` uses these certificate files to connect to `cloudcore`

This can be done by utilising scp

```shell
cd /etc/kubeedge/
scp -r certs.tgz username@destination:/etc/kubeedge
```
Here, we are copying the certs.tgz from the cloud side to the edge node in the /etc/kubeedge directory. You may copy in any directory and then move the certs to /etc/kubeedge folder.

## Configuration Edge side (KubeEdge Worker Node)

### Manually copy certs.tgz from cloud host to edge host(s)

On edge host

```shell
mkdir -p /etc/kubeedge
```

On edge host untar the certs.tgz file

```shell
cd /etc/kubeedge
tar -xvzf certs.tgz
```

### Set edgecore config file


```shell
  mkdir -p ~/cmd/conf
  cp $GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/* ~/cmd/conf
  vim ~/cmd/conf/edge.yaml
```

**Note:** `~/cmd/` dir is also an example as well as `cloudcore`, `conf/` should be in the same directory as edgecore binary,

Verify the configurations before running `edgecore`

```yaml
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

#### Modification in edge.yaml

1. Update the IP address of the master in the `edgehub.websocket.url` field. You need set cloudcore ip address.

    ```yaml
    url: wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events
    ```
    to 

    ```yaml
    url: wss://<X.X.X.X>:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events
    ```
    where X.X.X.X is your cloudcore IP Address.

2. Update the IP address of the master in the `edgehub.quic.url` field. You need set cloudcore ip address.

    ```yaml
    quic:
        url: 127.0.0.1:10001
    ```
    to 
    ```yaml
    quic:
        url:<X.X.X.X>:10001
    ```
3. Check `edged.podsandbox-image` : This is very important and must be set correctly.

   To check the architecture of your machine run the following

    ```
    getconf LONG_BIT
    ```

    + `kubeedge/pause-arm:3.1` for arm arch
    + `kubeedge/pause-arm64:3.1` for arm64 arch
    + `kubeedge/pause:3.1` for x86 arch

4. Check whether the cert files for `edgehub.websocket.certfile` and `edgehub.websocket.keyfile`  exist.
    
5. Check whether the cert files for `edgehub.quic.certfile` ,`edgehub.quic.keyfile` and `edgehub.quic.cafile` exist. If those files do not exist, you need to copy them from the cloud side.
   
6. Most importantly, Replace `edge-node` with edge node name in edge.yaml for the below fields :
    + `websocket:URL`

        ```yaml
        url: wss://0.0.0.0:10000/e632aba927ea4ac2b575ec1603d56f10/edge-node/events 
        ```
    + `controller:node-id`
        ```yaml
        controller:
        node-id: edge-node
        ```
    + `edged:hostname-override`
        ```yaml
        edged:
        register-node-namespace: default
        hostname-override: edge-node
        ```
7.  Configure the desired container runtime to be used as either docker or remote (for all CRI based runtimes including containerd). If this parameter is not specified docker runtime will be used by default
    ```yaml
    runtime-type: docker
    ```
    or
    ```yaml
    runtime-type: remote
    ```
8. If your runtime-type is remote, specify the following parameters for remote/CRI based runtimes
    ```yaml
    remote-runtime-endpoint: /var/run/containerd/containerd.sock
    remote-image-endpoint: /var/run/containerd/containerd.sock
    runtime-request-timeout: 2
    podsandbox-image: k8s.gcr.io/pause
    kubelet-root-dir: /var/run/kubelet/
    ```

#### Configuring MQTT mode
The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:

internalMqttMode: internal mqtt broker is enabled.
bothMqttMode: internal as well as external broker are enabled.
externalMqttMode: only external broker is enabled.
Use mode field in edge.yaml to select the desired mode.

To use KubeEdge in double mqtt or external mode, you need to make sure that mosquitto or emqx edge is installed on the edge node as an MQTT Broker.
