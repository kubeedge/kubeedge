# Setup from KubeEdge Installer

Keadm is used to install the cloud and edge components of kubeedge. It is not responsible for installing K8s and runtime, 
so users must install a k8s master on cloud and runtime on edge first. Or use an existing cluster.

Please refer [kubernetes-compatibility](https://github.com/kubeedge/kubeedge#kubernetes-compatibility) to get **Kubernetes compatibility** and determine what version of Kubernetes would be installed.

Kubeedge interacts with the standard K8s API, so the K8s cluster can be installed with any tools, such as:
- [Creating kubernetes cluster with kubeadm](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)
- [Creating kubernetes cluster with minikube](<https://kubernetes.io/docs/setup/learning-environment/minikube/>)
- [Creating kubernetes cluster with kind](<https://kubernetes.io/docs/setup/learning-environment/kind/>)

## Limitation

- Currently support of `keadm` is available for Ubuntu and CentOS OS. RaspberryPi supports is in-progress.

## Getting KubeEdge Installer

There are currently two ways to get keadm

- Download from [KubeEdge Release](<https://github.com/kubeedge/kubeedge/releases>)

  1. Go to [KubeEdge Release](<https://github.com/kubeedge/kubeedge/releases>) page and download `keadm-$VERSION-$OS-$ARCH.tar.gz.`.
  2. Untar it at desired location, by executing `tar -xvzf keadm-$VERSION-$OS-$ARCH.tar.gz`.
  3. kubeedge folder is created after execution the command.

- Building from source

  1. Download the source code.
  
      ```shell
      git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
      cd $GOPATH/src/github.com/kubeedge/kubeedge
      make all WHAT=keadm
      ```

      or

      ```shell
      go get github.com/kubeedge/kubeedge/keadm/cmd/keadm
      ```

  2. Binary `keadm` is available in current path. If you are using `go` get the binary is available in `$GOPATH/bin/`

## Setup Cloud Side (KubeEdge Master Node)

By default port '10000' in your cloudcore needs to be accessible for your edge nodes.

`keadm init` will install cloudcore, generate the certs and install the CRDs. It also provide flag by which specific versions can be set.

1. Execute `keadm init` : keadm needs super user rights (or root rights) to run successfully.

    Command flags
    The optional flags with this command are mentioned below

    ```shell
    "keadm init" command install KubeEdge's master node (on the cloud) component.
    It checks if the Kubernetes Master are installed already,
    If not installed, please install the Kubernetes first.
    
    Usage:
      keadm init [flags]
    
    Examples:
    
    keadm init
    
    - This command will download and install the default version of KubeEdge cloud component
    
    keadm init --kubeedge-version=1.2.0  --kube-config=/root/.kube/config
    
      - kube-config is the absolute path of kubeconfig which used to secure connectivity between cloudcore and kube-apiserver
    
    
    Flags:
      -h, --help                                help for init
          --kube-config string                  Use this key to set kube-config path, eg: $HOME/.kube/config (default "/root/.kube/config")
          --kubeedge-version string[="1.2.0"]   Use this key to download and use the required KubeEdge version (default "1.2.0")
          --master string                       Use this key to set K8s master address, eg: http://127.0.0.1:8080
    ```

**IMPORTANT NOTE:** At least one of kubeconfig or master must be configured correctly, so that it can be used to verify the version and other info of the k8s cluster.

Examples:

 ```shell
  keadm init
 ```

Sample execution output:
```
Kubernetes version verification passed, KubeEdge installation will start...
...
KubeEdge cloudcore is running, For logs visit:  /var/log/kubeedge/cloudcore.log
```  

## Manually copy certs.tgz from cloud host to edge host(s)

Now users still need to copy the certs to the edge nodes. In the future, it will support the use of tokens for authentication.

On edge host

```
mkdir -p /etc/kubeedge
```

On cloud host

```
cd /etc/kubeedge/
scp -r certs.tgz username@ipEdgevm:/etc/kubeedge
```

On edge host untar the certs.tgz file

```
cd /etc/kubeedge
tar -xvzf certs.tgz
```


## Setup Edge Side (KubeEdge Worker Node)

`keadm join` will install edgecore and mqtt. It also provide flag by which specific versions can be set.

Execute `keadm join <flags>`

 Command flags

  The optional flags with this command are shown in below shell

  ```shell
  "keadm join" command bootstraps KubeEdge's worker node (at the edge) component.
  It will also connect with cloud component to receive 
  further instructions and forward telemetry data from 
  devices to cloud
  
  Usage:
    keadm join [flags]
  
  Examples:
  
  keadm join --cloudcore-ipport=<ip:port address> --edgenode-name=<unique string as edge identifier>
  
    - For this command --cloudcore-ipport flag is a required option
    - This command will download and install the default version of pre-requisites and KubeEdge
  
  keadm join --cloudcore-ipport=10.20.30.40:10000 --edgenode-name=testing123 --kubeedge-version=1.2.0
  
  
  Flags:
        --certPath string                     The certPath used by edgecore, the default value is /etc/kubeedge/certs (default "/etc/kubeedge/certs")
    -e, --cloudcore-ipport string             IP:Port address of KubeEdge CloudCore
    -i, --edgenode-name string                KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own
    -h, --help                                help for join
        --interfacename string                KubeEdge Node interface name string, the default value is eth0
        --kubeedge-version string[="1.2.0"]   Use this key to download and use the required KubeEdge version (default "1.2.0")
    -r, --runtimetype string                  Container runtime type
  ```

**IMPORTANT NOTE:** 
1. For this command --cloudcore-ipport flag is a Mandatory flag
1. The KubeEdge version used in cloud and edge side should be same. 

 Examples:

 ```shell
  keadm join --cloudcore-ipport=192.168.20.50:10000
 ```

Sample execution output:

```shell
Host has mosquit+ already installed and running. Hence skipping the installation steps !!!
...
KubeEdge edgecore is running, For logs visit:  /var/log/kubeedge/edgecore.log
```

## Reset KubeEdge Master and Worker nodes

`keadm reset` will stop KubeEdge components. It doesn't uninstall/remove any of the pre-requisites.

Execute `keadm reset`

Command flags

```shell
keadm reset --help

keadm reset command can be executed in both cloud and edge node
In cloud node it shuts down the cloud processes of KubeEdge
In edge node it shuts down the edge processes of KubeEdge

Usage:
  keadm reset [flags]

Examples:

For cloud node:
keadm reset

For edge node:
keadm reset


Flags:
  -h, --help   help for reset
```

## Errata

1. Error in CloudCore

    If you are getting the below error in Cloudcore.log

    ```shell
    E1231 04:37:27.397431   19607 reflector.go:125] github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager/device.go:40: Failed to list *v1alpha1.Device: the server could not find the requested resource (get devices.devices.kubeedge.io)
    E1231 04:37:27.398273   19607 reflector.go:125] github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager/devicemodel.go:40: Failed to list *v1alpha1.DeviceModel: the server could not find the requested resource (get devicemodels.devices.kubeedge.io)
    ```

    browse to the

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices
    ```

    and apply the below

    ```shell
      kubectl create -f devices_v1alpha1_devicemodel.yaml
      kubectl create -f devices_v1alpha1_device.yaml
    ```

    or

    ```shell
     kubectl create -f https://raw.githubusercontent.com/kubeedge/kubeedge/<kubeEdge Version>/build/crds/devices/devices_v1alpha1_device.yaml
     kubectl create -f https://raw.githubusercontent.com/kubeedge/kubeedge/<kubeEdge Version>/build/crds/devices/devices_v1alpha1_devicemodel.yaml
    ```
   
    Also, create ClusterObjectSync and ObjectSync CRDs which are used in reliable message delivery.

    ```shell
     cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/reliablesyncs
     kubectl create -f cluster_objectsync_v1alpha1.yaml
     kubectl create -f objectsync_v1alpha1.yaml
    ```
