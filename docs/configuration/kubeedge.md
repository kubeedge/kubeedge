# KubeEdge Configuration

KubeEdge requires configuration on both [Cloud side (KubeEdge Master)](#configuration-cloud-side-kubeedge-master) and [Edge side (KubeEdge Worker Node)](#configuration-edge-side-kubeedge-worker-node)

## Configuration Cloud side (KubeEdge Master)

Setting up cloud side requires two steps

1. [Modification of the configuration files](#modification-of-the-configuration-file)
2. Edge node will be auto registered by default. [Users can still choose to register manually](#adding-the-edge-nodes-kubeedge-worker-node-on-the-cloud-side-kubeedge-master).

### Modification of the configuration file

Cloudcore requires changes in `cloudcore.yaml` configuration file.

Create and set cloudcore config file

Create the `/etc/kubeedge/config` folder

```shell
# the default configuration file path is '/etc/kubeedge/config/cloudcore.yaml'
# also you can specify it anywhere with '--config'
mkdir -p /etc/kubeedge/config/
```

Either create a minimal configuration with command `~/kubeedge/cloudcore --minconfig`

```shell
~/kubeedge/cloudcore --minconfig > /etc/kubeedge/config/cloudcore.yaml
```

or a full configuration with command `~/kubeedge/cloudcore --defaultconfig`

```shell
~/kubeedge/cloudcore --defaultconfig > /etc/kubeedge/config/cloudcore.yaml
```

Edit the configuration file

```shell
vim /etc/kubeedge/config/cloudcore.yaml
```

Verify the configurations before running `cloudcore`

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

2. Before KubeEdge v1.3: check whether the cert files for `modules.cloudhub.tlsCAFile`, `modules.cloudhub.tlsCertFile`,`modules.cloudhub.tlsPrivateKeyFile` exists.

    From KubeEdge v1.3: just skip the above check. If you configure the CloudCore certificate manually, you must check if the path of certificate is right.



    **Note:** If your KubeEdge version is before the v1.3, then just skip the step 3.

3. Configure all the IP addresses of CloudCore which are exposed to the edge nodes(like floating IP) in the `advertiseAddress`, which will be added to SANs in cert of cloudcore.

    ```yaml
    modules:
      cloudHub:
        advertiseAddress:
        - 10.1.11.85
    ```

### Adding the edge nodes (KubeEdge Worker Node) on the Cloud side (KubeEdge Master)

Node registration can be completed in two ways:

1. Node - Automatic Registration
2. Node - Manual Registration

#### Node - Automatic Registration

Edge node can be registered automatically if the value of field `modules.edged.registerNode` in edgecore's [config](#create-and-set-edgecore-config-file) is set to true.

```yaml
modules:
  edged:
    registerNode: true
```

#### Node - Manual Registration

##### Copy `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` to your working directory and change `metadata.name` to the name of edge node

```shell
mkdir -p ~/kubeedge/yaml
cp $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json ~/kubeedge/yaml
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

**Note:**
1. the `metadata.name` must keep in line with edgecore's config `modules.edged.hostnameOverride`.

2. Make sure role is set to edge for the node. For this a key of the form `"node-role.kubernetes.io/edge"` must be present in `metadata.labels`.
If role is not set for the node, the pods, configmaps and secrets created/updated in the cloud cannot be synced with the node they are targeted for.

##### Deploy edge node (**you must run the command on cloud side**)

```shell
kubectl apply -f ~/kubeedge/yaml/node.json
```

#### Check the existence of certificates (cloud side) (Required for pre 1.3 releases)

**Note:** From KubeEdge v1.3, just skip the follow steps of checking the existence of certificates. However, if you configure the cloudcore certificate manually, you must check if the path of certificate is right. And there is no need to transfer certificate file from the cloud side to edge side.

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

cert/key should exist in /etc/kubeedge/ca and /etc/kubeedge/certs. Otherwise please refer to [generate certs](https://github.com/kubeedge/kubeedge/blob/release-1.3/docs/setup/kubeedge_install_source.md#generate-certificates) to generate them.
You need to copy these files to the corresponding directory on edge side.

Create the `certs.tgz` by

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

At this point we have completed all configuration changes related to cloudcore.

## Configuration Edge side (KubeEdge Worker Node)

### Manually copy certs.tgz from cloud host to edge host(s)  (Required for pre 1.3 releases)

**Note:**  From KubeEdge v1.3 just skip this step, the edgecore will apply for the certificate automatically from the cloudcore when starting. You can also configure the local certificate(The CA certificate in edge site must be the same with cloudcore now). Any directory is OK as long as you configure it in the edgecore.yaml below.

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

Either create a minimal configuration with command `~/kubeedge/edgecore --minconfig`

```shell
    ~/kubeedge/edgecore --minconfig > /etc/kubeedge/config/edgecore.yaml
```

or a full configuration with command `~/kubeedge/edgecore --defaultconfig`

```shell
~/kubeedge/edgecore --defaultconfig > /etc/kubeedge/config/edgecore.yaml
```

Edit the configuration file

```shell
    vim /etc/kubeedge/config/edgecore.yaml
```

Verify the configurations before running `edgecore`

#### Modification in edgecore.yaml

1. Check `modules.edged.podSandboxImage` : This is very important and must be set correctly.

   To check the architecture of your machine run the following

    ```shell
    getconf LONG_BIT
    ```

    + `kubeedge/pause-arm:3.1` for arm arch
    + `kubeedge/pause-arm64:3.1` for arm64 arch
    + `kubeedge/pause:3.1` for x86 arch

2. Before KubeEdge v1.3: check whether the cert files for `modules.edgehub.tlsCaFile` and `modules.edgehub.tlsCertFile` and `modules.edgehub.tlsPrivateKeyFile` exists. If those files not exist, you need to copy them from cloud side.

    From KubeEdge v1.3: just skip above check about cert files. However, if you configure the edgecore certificate manually, you must check if the path of certificate is right.

3. Update the IP address and port of the KubeEdge CloudCore in the `modules.edgehub.websocket.server` and `modules.edgehub.quic.server` field. You need set cloudcore ip address.

4. Configure the desired container runtime to be used as either docker or remote (for all CRI based runtimes including containerd). If this parameter is not specified docker runtime will be used by default

    ```yaml
    runtimeType: docker
    ```

    or

    ```yaml
    runtimeType: remote
    ```

5. If your runtime-type is remote, follow this guide [KubeEdge CRI Configuration](cri.md) to setup KubeEdge with the remote/CRI based runtimes.



    **Note:** If your KubeEdge version is before the v1.3, then just skip the steps 6-7.

6. Configure the IP address and port of the KubeEdge cloudcore in the `modules.edgehub.httpServer` which is used to apply for the certificate. For example:

    ```yaml
    modules:
      edgeHub:
        httpServer: https://10.1.11.85:10002
    ```

7. Configure the token.

    ```shell
    kubectl get secret tokensecret -n kubeedge -oyaml
    ```

    Then you get it like this:

    ```yaml
    apiVersion: v1
    data:
      tokendata: ODEzNTZjY2MwODIzMmIxMTU0Y2ExYmI5MmRlZjY4YWQwMGQ3ZDcwOTIzYmU3YjcyZWZmOTVlMTdiZTk5MzdkNS5leUpoYkdjaU9pSklVekkxTmlJc0luUjVjQ0k2SWtwWFZDSjkuZXlKbGVIQWlPakUxT0RreE5qRTVPRGw5LmpxNENXNk1WNHlUVkpVOWdBUzFqNkRCdE5qeVhQT3gxOHF5RnFfOWQ4WFkK
    kind: Secret
    metadata:
      creationTimestamp: "2020-05-10T01:53:10Z"
      name: tokensecret
      namespace: kubeedge
      resourceVersion: "19124039"
      selfLink: /api/v1/namespaces/kubeedge/secrets/tokensecret
      uid: 48429ce1-2d5a-4f0e-9ff1-f0f1455a12b4
    type: Opaque
    ```

    Decode the tokendata field by base64:

    ```shell
    echo ODEzNTZjY2MwODIzMmIxMTU0Y2ExYmI5MmRlZjY4YWQwMGQ3ZDcwOTIzYmU3YjcyZWZmOTVlMTdiZTk5MzdkNS5leUpoYkdjaU9pSklVekkxTmlJc0luUjVjQ0k2SWtwWFZDSjkuZXlKbGVIQWlPakUxT0RreE5qRTVPRGw5LmpxNENXNk1WNHlUVkpVOWdBUzFqNkRCdE5qeVhQT3gxOHF5RnFfOWQ4WFkK |base64 -d
    # then we get:
    81356ccc08232b1154ca1bb92def68ad00d7d70923be7b72eff95e17be9937d5.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODkxNjE5ODl9.jq4CW6MV4yTVJU9gAS1j6DBtNjyXPOx18qyFq_9d8XY
    ```

    Copy the decoded string to the edgecore.yaml just like follow:

    ```yaml
    modules:
      edgeHub:
        token: 81356ccc08232b1154ca1bb92def68ad00d7d70923be7b72eff95e17be9937d5.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODkxNjE5ODl9.jq4CW6MV4yTVJU9gAS1j6DBtNjyXPOx18qyFq_9d8XY
    ```

#### Configuring MQTT mode

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes (`internalMqttMode`, `bothMqttMode`, `externalMqttMode`), set `mqttMode` field in edgecore.yaml to the desired mode.
+ internalMqttMode: internal mqtt broker is enabled (`mqttMode`=0).
+ bothMqttMode: internal as well as external broker are enabled (`mqttMode`=1).
+ externalMqttMode: only external broker is enabled (`mqttMode`=2).

To use KubeEdge in double mqtt or external mode, you need to make sure that mosquitto or emqx edge is installed on the edge node as an MQTT Broker.

At this point we have completed all configuration changes related to edgecore.
