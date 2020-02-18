# KubeEdge Configuration

KubeEdge requires configuration on both [Cloud side (KubeEdge Master)](#configuration-cloud-side-kubeedge-master) and [Edge side (KubeEdge Worker Node)](#configuration-edge-side-kubeedge-worker-node)

## Configuration Cloud side (KubeEdge Master)

Setting up cloud side requires two steps

1. [Modification of the configuration files](#modification-of-the-configuration-file)
2. [Adding the edge nodes (KubeEdge Worker Node) on the Cloud side (KubeEdge Master)](#adding-the-edge-nodes-kubeedge-worker-node-on-the-cloud-side-kubeedge-master). Node Registration can be completed by a automatic or manual method.

### Modification of the configuration file

Cloudcore requires changes in `cloudcore.yaml` configuration file.

Create and set cloudcore config file

Create the `/etc/kubeedge/config` folder

```shell
# the default configuration file path is '/etc/kubeedge/config/cloudcore.yaml'
# also you can specify it anywhere with '--config'
mkdir -p /etc/kubeedge/config/
```

Either create a minimal configuration with command `~/cmd/cloudcore --minconfig`

```shell

~/cmd/cloudcore --minconfig > /etc/kubeedge/config/cloudcore.yaml
```

or a full configuration with command `~/cmd/cloudcore --defaultconfig`

```shell
~/cmd/cloudcore --defaultconfig > /etc/kubeedge/config/cloudcore.yaml
```

Edit the configuration file

```shell
vim /etc/kubeedge/config/cloudcore.yaml
```

Verify the configurations before running `cloudcore`

For completion purposes, below is the configuration created using `--defaultconfig`

```yaml
# With --defaultconfig flag, users can easily get a default full config file as reference, with all fields (and field descriptions) included and default values set.
# Users can modify/create their own configs accordingly as reference.
# Because it is a full configuration, it is more suitable for advanced users.

apiVersion: cloudcore.config.kubeedge.io/v1alpha1
kind: CloudCore
kubeAPIConfig:
  burst: 200
  contentType: application/vnd.kubernetes.protobuf
  kubeConfig: /root/.kube/config
  master: ""
  qps: 100
modules:
  cloudhub:
    enable: true
    keepaliveInterval: 30
    nodeLimit: 10
    quic:
      address: 0.0.0.0
      maxIncomingStreams: 10000
      port: 10001
    tlsCAFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    unixsocket:
      address: unix:///var/lib/kubeedge/kubeedge.sock
      enable: true
    websocket:
      address: 0.0.0.0
      enable: true
      port: 10000
    writeTimeout: 30
  edgecontroller:
    buffer:
      configmapEvent: 1
      endpointsEvent: 1
      podEvent: 1
      queryConfigmap: 1024
      queryEndpoints: 1024
      queryNode: 1024
      queryPersistentvolume: 1024
      queryPersistentvolumeclaim: 1024
      querySecret: 1024
      queryService: 1024
      queryVolumeattachment: 1024
      secretEvent: 1
      serviceEvent: 1
      updateNode: 1024
      updateNodeStatus: 1024
      updatePodStatus: 1024
    context:
      receiveModule: edgecontroller
      responseModule: cloudhub
      sendModule: cloudhub
    enable: true
    load:
      queryConfigmapWorkers: 4
      queryEndpointsWorkers: 4
      queryNodeWorkers: 4
      queryPersistentColumeClaimWorkers: 4
      queryPersistentVolumeWorkers: 4
      querySecretWorkers: 4
      queryServiceWorkers: 4
      queryVolumeAttachmentWorkers: 4
      updateNodeStatusWorkers: 1
      updateNodeWorkers: 4
      updatePodStatusWorkers: 1
    nodeUpdateFrequency: 10
```

#### Modification in cloudcore.yaml

In the cloudcore.yaml, modify the below settings.

1. Either `kubeAPIConfig.kubeConfig` or `kubeAPIConfig.master` : This would be the path to your kubeconfig file. It might be either

    ```shell
    /root/.kube/config
    ```

    or

    ```shell
    /home/<your_username>/.kube/config
    ```

    depending on where you have setup your kubernetes by performing the below step:

    ```shell
    To start using your cluster, you need to run the following as a regular user:

    mkdir -p $HOME/.kube
    sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
    sudo chown $(id -u):$(id -g) $HOME/.kube/config
    ```

    By default, cloudcore use https connection to Kubernetes apiserver. If `master` and `kubeConfig` are both set, `master` will override any value in kubeconfig.

2. Check whether the cert files for `modules.cloudhub.tlsCAFile`, `modules.cloudhub.tlsCertFile`,`modules.cloudhub.tlsPrivateKeyFile` exists.

### Adding the edge nodes (KubeEdge Worker Node) on the Cloud side (KubeEdge Master)

Node registration can be completed in two ways:

1. Node - Automatic Registration
2. Node - Manual Registration

#### Node - Automatic Registration

Edge node can be registered automatically if the `modules.edged.registerNode` is true.

```yaml
modules:
  edged:
    registerNode: true
```

#### Node - Manual Registration

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

 If you want to add an edge node to an existing kubernetes cluster you should a taint to this node. An example is given in this Node.yaml file:

 ```yaml
 {
    kind: Node
    apiVersion: v1
    metadata: {
      name: edge-node
      labels: {
        name: edge-node
        node-role.kubernetes.io/edge: ""
      }
    spec
      taints
        effect: noSchedule
        key: node.kubeedge.io
        value: edge
    }
 }
 ```

##### Modification in node.json

1. metadata.name : This is the name of your edge node device. (Referring the edge node configuration would make it more clear).

    **Note: you need to remember `metadata.name` , because edgecore needs it**.

2. metadata.labels: Labels are the name by which you can remember a set of nodes.

    + Make sure role is set to `edge` for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `labels` tag of `metadata`.
    + Please ensure to add the label `node-role.kubernetes.io/edge` to the `build/node.json` file.

    + If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.

    + This can be checked by running the below command on Cloud side

        ```shell
        kubectl get nodes --show-labels
        ```

##### Deploy edge node (Run on cloud side)

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

The cert/key will be generated in the /etc/kubeedge/ca and /etc/kubeedge/certs respectively, so this command should be run with root or users who have read/ write permission to those directories.
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
scp certs.tgz username@destination:/etc/kubeedge
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

### Create and set edgecore config file

Create the `/etc/kubeedge/config` folder

```shell

    # the default configration file path is '/etc/kubeedge/config/edgecore.yaml'
    # also you can specify it anywhere with '--config'
    mkdir -p /etc/kubeedge/config/
```

Either create a minimal configuration with command `~/cmd/edgecore --minconfig`

```shell
    ~/cmd/edgecore --minconfig > /etc/kubeedge/config/edgecore.yaml
```

or a full configuration with command `~/cmd/edgecore --defaultconfig`

```shell
~/cmd/edgecore --defaultconfig > /etc/kubeedge/config/edgecore.yaml
```

Edit the configuration file

```shell
    vim /etc/kubeedge/config/edgecore.yaml
```

Verify the configurations before running `edgecore`

For completion purposes, below is the configuration created using `--defaultconfig`

```yaml
# With --defaultconfig flag, users can easily get a default full config file as reference, with all fields (and field descriptions) included and default values set.
# Users can modify/create their own configs accordingly as reference.
# Because it is a full configuration, it is more suitable for advanced users.

apiVersion: edgecore.config.kubeedge.io/v1alpha1
database:
  aliasName: default
  dataSource: /var/lib/kubeedge/edgecore.db
  driverName: sqlite3
kind: EdgeCore
modules:
  dbtest:
    enable: false
  devicetwin:
    enable: true
  edged:
    cgroupDriver: cgroupfs
    clusterDNS: ""
    clusterDomain: ""
    devicePluginEnabled: false
    dockerAddress: unix:///var/run/docker.sock
    edgedMemoryCapacity: 7852396000
    enable: true
    gpuPluginEnabled: false
    hostnameOverride: example.local
    imageGCHighThreshold: 80
    imageGCLowThreshold: 40
    imagePullProgressDeadline: 60
    interfaceName: eth0
    maximumDeadContainersPerPod: 1
    nodeIP: 10.0.2.15
    nodeStatusUpdateFrequency: 10
    podSandboxImage: kubeedge/pause:3.1
    registerNode: true
    registerNodeNamespace: default
    remoteImageEndpoint: unix:///var/run/dockershim.sock
    remoteRuntimeEndpoint: unix:///var/run/dockershim.sock
    runtimeRequestTimeout: 2
    runtimeType: docker
  edgehub:
    enable: true
    heartbeat: 15
    projectID: e632aba927ea4ac2b575ec1603d56f10
    quic:
      handshakeTimeout: 30
      readDeadline: 15
      server: 127.0.0.1:10001
      writeDeadline: 15
    tlsCaFile: /etc/kubeedge/ca/rootCA.crt
    tlsCertFile: /etc/kubeedge/certs/edge.crt
    tlsPrivateKeyFile: /etc/kubeedge/certs/edge.key
    websocket:
      enable: true
      handshakeTimeout: 30
      readDeadline: 15
      server: 127.0.0.1:10000
      writeDeadline: 15
  edgemesh:
    enable: true
    lbStrategy: RoundRobin
  eventbus:
    enable: true
    mqttMode: 2
    mqttQOS: 0
    mqttRetain: false
    mqttServerExternal: tcp://127.0.0.1:1883
    mqttServerInternal: tcp://127.0.0.1:1884
    mqttSessionQueueSize: 100
  metamanager:
    contextSendGroup: hub
    contextSendModule: websocket
    enable: true
    podStatusSyncInterval: 60
  servicebus:
    enable: false
```

#### Modification in edgecore.yaml

1. Check `modules.edged.podSandboxImage` : This is very important and must be set correctly.

   To check the architecture of your machine run the following

    ```shell
    getconf LONG_BIT
    ```

    + `kubeedge/pause-arm:3.1` for arm arch
    + `kubeedge/pause-arm64:3.1` for arm64 arch
    + `kubeedge/pause:3.1` for x86 arch

2. Check whether the cert files for `modules.edgehub.tlsCaFile` and `modules.edgehub.tlsCertFile` and `modules.edgehub.tlsPrivateKeyFile` exists. If those files not exist, you need to copy them from cloud side.

3. Update the IP address and port of the KubeEdge Master in the `modules.edgehub.websocket.server` and `modules.edgehub.quic.server` field. You need set cloudcore ip address.

4. Configure the desired container runtime to be used as either docker or remote (for all CRI based runtimes including containerd). If this parameter is not specified docker runtime will be used by default

    ```yaml
    runtimeType: docker
    ```

    or

    ```yaml
    runtimeType: remote
    ```

5. If your runtime-type is remote, specify the following parameters for remote/CRI based runtimes

    ```yaml
    remoteRuntimeEndpoint: /var/run/containerd/containerd.sock
    remoteImageEndpoint: /var/run/containerd/containerd.sock
    runtimeRequestTimeout: 2
    podSandboxImage: k8s.gcr.io/pause
    kubelet-root-dir: /var/run/kubelet/
    ```

#### Configuring MQTT mode

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:

+ internalMqttMode: internal mqtt broker is enabled.
+ bothMqttMode: internal as well as external broker are enabled.
+ externalMqttMode: only external broker is enabled.

Use mode field in edge.yaml to select the desired mode.

To use KubeEdge in double mqtt or external mode, you need to make sure that mosquitto or emqx edge is installed on the edge node as an MQTT Broker.
