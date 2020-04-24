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

2. Check whether the cert files for `modules.cloudhub.tlsCAFile`, `modules.cloudhub.tlsCertFile`,`modules.cloudhub.tlsPrivateKeyFile` exists.

### Adding the edge nodes (KubeEdge Worker Node) on the Cloud side (KubeEdge Master)

Node registration can be completed in two ways:

1. Node - Automatic Registration
2. Node - Manual Registration

#### Node - Automatic Registration

Edge node can be registered automatically if the value of field `modules.edged.registerNode` in edgecore's config [edgecore config file](https://github.com/kubeedge/kubeedge/blob/master/docs/setup/kubeedge_configure.md#create-and-set-edgecore-config-file) is set to true.

```yaml
modules:
  edged:
    registerNode: true
```

#### Node - Manual Registration

Refer [here](deploy-edge-node.md) to add edge nodes.

#### Check the existence of certificates (cloud side)

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

cert/key should exist in /etc/kubeedge/ca and /etc/kubeedge/certs. Otherwise please refer to [generate certs](https://github.com/kubeedge/kubeedge/blob/master/docs/setup/kubeedge_install_source.md#generate-certificates) to generate them.
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

2. Check whether the cert files for `modules.edgehub.tlsCaFile` and `modules.edgehub.tlsCertFile` and `modules.edgehub.tlsPrivateKeyFile` exists. If those files not exist, you need to copy them from cloud side.

3. Update the IP address and port of the KubeEdge CloudCore in the `modules.edgehub.websocket.server` and `modules.edgehub.quic.server` field. You need set cloudcore ip address.

4. Configure the desired container runtime to be used as either docker or remote (for all CRI based runtimes including containerd). If this parameter is not specified docker runtime will be used by default

    ```yaml
    runtimeType: docker
    ```

    or

    ```yaml
    runtimeType: remote
    ```

5. If your runtime-type is remote, follow this guide [KubeEdge CRI Configuration](kubeedge_cri_configure.md) to setup KubeEdge with the remote/CRI based runtimes.

#### Configuring MQTT mode

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes (`internalMqttMode`, `bothMqttMode`, `externalMqttMode`), set `mqttMode` field in edgecore.yaml to the desired mode.
+ internalMqttMode: internal mqtt broker is enabled (`mqttMode`=0).
+ bothMqttMode: internal as well as external broker are enabled (`mqttMode`=1).
+ externalMqttMode: only external broker is enabled (`mqttMode`=2).

To use KubeEdge in double mqtt or external mode, you need to make sure that mosquitto or emqx edge is installed on the edge node as an MQTT Broker.

At this point we have completed all configuration changes related to edgecore.
