# Getting Started with KubeEdge Installer

Please refer to KubeEdge Installer proposal document for details on the motivation of having KubeEdge Installer.
It also explains the functionality of the proposed commands.
[KubeEdge Installer Doc](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/keadm-scope.md/)

## Limitation

- Currently support of `KubeEdge installer` is available only for Ubuntu OS. CentOS support is in-progress.

## Downloading KubeEdge Installer

1. Go to [KubeEdge Release](<https://github.com/kubeedge/kubeedge/releases>) page and download `keadm-$VERSION-$OS-$ARCH.tar.gz.`.
2. Untar it at desired location, by executing `tar -xvzf keadm-$VERSION-$OS-$ARCH.tar.gz`.
3. kubeedge folder is created after execution the command.

## Building from source

1. Download the source code either by

- `git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge`
2. `cd $GOPATH/src/github.com/kubeedge/kubeedge/keadm`
3. `make`
4. Binary `keadm` is available in current path

## Installing KubeEdge Master Node (on the Cloud) component

Referring to `KubeEdge Installer Doc`, the command to install KubeEdge cloud component (edge controller) and pre-requisites.
Port 8080, 6443 and 10000 in your cloud component needs to be accessible for your edge nodes.

- Execute `keadm init`

### Command flags
The optional flags with this command are mentioned below

```
$  keadm init --help

keadm init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.

Usage:
  keadm init [flags]

Examples:

keadm init


Flags:
      --docker-version string[="18.06.0"]          Use this key to download and use the required Docker version (default "18.06.0")
  -h, --help                                       help for init
      --kubeedge-version string[="0.3.0-beta.0"]   Use this key to download and use the required KubeEdge version (default "0.3.0-beta.0")
      --kubernetes-version string[="1.14.1"]       Use this key to download and use the required Kubernetes version (default "1.14.1")

```

1. `--docker-version`, if mentioned with any version > 18.06.0, will install the same on the host. Default is 18.06.0. It is optional.
2. `--kubernetes-version`, if mentioned with any version > 1.14.1, will install the same on the host. Default is 1.14.1. It is optional.
                           It will install `kubeadm`, `kubectl` and `kubelet` in this host.
3. `--kubeedge-version`, if mentioned with any version > 0.2.1, will install the same on the host. Default is 0.3.0-beta.0. It is optional.

command format is

```
keadm init --docker-version=<expected version> --kubernetes-version=<expected version> --kubeedge-version=<expected version>
```

**NOTE:**
Version mentioned as defaults for Docker and K8S are being tested with.

## Installing KubeEdge Worker Node (at the Edge) component

Referring to `KubeEdge Installer Doc`, the command to install KubeEdge Edge component (edge core) and pre-requisites

- Execute `keadm join <flags>`

### Command flags

The optional flags with this command are shown in below shell

```
$  keadm join --help
 
"keadm join" command bootstraps KubeEdge's edge component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
to install the prerequisites.
It will help the edge node to connect to the cloud.


Usage:
  keadm join [flags]

Examples:

keadm join --edgecontrollerip=<ip address> --edgenodeid=<unique string as edge identifier>

  - For this command --edgecontrollerip flag is a Mandatory flag
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --edgecontrollerip=10.20.30.40 --edgenodeid=testing123 --kubeedge-version=0.2.1 --k8sserverip=50.60.70.80:8080

  - In case, any option is used in a format like as shown for "--docker-version" or "--docker-version=", without a value
        then default values will be used.
        Also options like "--docker-version", and "--kubeedge-version", version should be in
        format like "18.06.3" and "0.2.1".


Flags:
      --docker-version string[="18.06.0"]          Use this key to download and use the required Docker version (default "18.06.0")
  -e, --edgecontrollerip string                    IP address of KubeEdge edgecontroller
  -i, --edgenodeid string                          KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own
  -h, --help                                       help for join
  -k, --k8sserverip string                         IP:Port address of K8S API-Server
      --kubeedge-version string[="0.3.0-beta.0"]   Use this key to download and use the required KubeEdge version (default "0.3.0-beta.0")

```

1. For KubeEdge flag the functionality is same as mentioned in `keadm init`
2. -k, --k8sserverip, It should be in the format <IPAddress:Port>, where the default port is 8080. Please see the example above.


**IMPORTANT NOTE:** The KubeEdge version used in cloud and edge side should be same.

## Reset KubeEdge Master and Worker nodes

Referring to `KubeEdge Installer Doc`, the command to stop KubeEdge cloud (edge controller). It doesn't uninstall/remove any of the pre-requisites.

- Execute `keadm reset`

### Command flags

```
keadm reset --help

keadm reset command can be executed in both cloud and edge node
In master node it shuts down the cloud processes of KubeEdge
In worker node it shuts down the edge processes of KubeEdge

Usage:
  keadm reset [flags]

Examples:

For master node:
keadm reset

For worker node:
keadm reset --k8sserverip 10.20.30.40:8080


Flags:
  -h, --help                 help for reset
  -k, --k8sserverip string   IP:Port address of cloud components host/VM

```

## Simple steps to bring up KubeEdge setup and deploy a pod

**NOTE:** All the below steps are executed as root user, to execute as sudo user ,Please add **sudo** infront of all the commands

### 1. Deploy KubeEdge edgeController (With K8S Cluster)

#### Install tools with the particular version

```
keadm init --kubeedge-version=<kubeedge Version>  --kubernetes-version=<kubernetes Version> --docker-version=<Docker version>
```

#### Install tools with the default version

```
keadm init --kubeedge-version= --kubernetes-version= --docker-version
or
keadm init
```

**NOTE:**
On the console output, obeserve the below line

kubeadm join **192.168.20.134**:6443 --token 2lze16.l06eeqzgdz8sfcvh \
         --discovery-token-ca-cert-hash sha256:1e5c808e1022937474ba264bb54fea42b05eddb9fde2d35c9cad5b83cf5ef9ac  
After Kubeedge init ,please note the **cloudIP** as highlighted above generated from console output and port is **8080**.

### 2. Manually copy certs.tgz from cloud host to edge host(s)

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

### 3. Deploy KubeEdge edge core

#### Install tools with the particular version

```
keadm join --edgecontrollerip=<cloudIP> --edgenodeid=<unique string as edge identifier> --k8sserverip=<cloudIP>:8080 --kubeedge-version=<kubeedge Version> --docker-version=<Docker version>
```

#### Install tools with the default version 

```
keadm join --edgecontrollerip=<cloudIP> --edgenodeid=<unique string as edge identifier> --k8sserverip=<cloudIP>:8080 --kubeedge-version=<kubeedge Version> --docker-version=<Docker version>
```

Sample execution output:
```
# ./keadm join --edgecontrollerip=192.168.20.50 --edgenodeid=testing123 --k8sserverip=192.168.20.50:8080
Same version of docker already installed in this host
Host has mosquit+ already installed and running. Hence skipping the installation steps !!!
Expected or Default KubeEdge version 0.3.0-beta.0 is already downloaded
kubeedge/
kubeedge/edge/
kubeedge/edge/conf/
kubeedge/edge/conf/modules.yaml
kubeedge/edge/conf/logging.yaml
kubeedge/edge/conf/edge.yaml
kubeedge/edge/edge_core
kubeedge/cloud/
kubeedge/cloud/edgecontroller
kubeedge/cloud/conf/
kubeedge/cloud/conf/controller.yaml
kubeedge/cloud/conf/modules.yaml
kubeedge/cloud/conf/logging.yaml
kubeedge/version

KubeEdge Edge Node: testing123 successfully add to kube-apiserver, with operation status: 201 Created
Content {"kind":"Node","apiVersion":"v1","metadata":{"name":"testing123","selfLink":"/api/v1/nodes/testing123","uid":"87d8d7a3-7acd-11e9-b86b-286ed488c645","resourceVersion":"3864","creationTimestamp":"2019-05-20T07:04:37Z","labels":{"name":"edge-node"}},"spec":{"taints":[{"key":"node.kubernetes.io/not-ready","effect":"NoSchedule"}]},"status":{"daemonEndpoints":{"kubeletEndpoint":{"Port":0}},"nodeInfo":{"machineID":"","systemUUID":"","bootID":"","kernelVersion":"","osImage":"","containerRuntimeVersion":"","kubeletVersion":"","kubeProxyVersion":"","operatingSystem":"","architecture":""}}}

KubeEdge edge core is running, For logs visit /etc/kubeedge/kubeedge/edge/
#
```

**Note**:Cloud IP refers to IP generated ,from the step 1 as highlighted

### 4. Edge node status on edgeController (master node) console

On cloud host run,

```
kubectl get nodes

NAME         STATUS     ROLES    AGE     VERSION
testing123   Ready      <none>   6s      0.3.0-beta.0
```
Check if the edge node is in ready state

### 5.Deploy a sample pod from Cloud VM

**https://github.com/kubeedge/kubeedge/blob/master/build/deployment.yaml**

Copy the deployment.yaml from the above link in cloud host,run

```
kubectl create -f deployment.yaml
deployment.apps/nginx-deployment created
```

### 6.Pod status

Check the pod is up and is running state

```
kubectl get pods
NAME                               READY   STATUS    RESTARTS   AGE
nginx-deployment-d86dfb797-scfzz   1/1     Running   0          44s
```

Check the deployment is up and is running state
```
kubectl get deployments

NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   1/1     1            1           63s
```

## Errata

1.If GPG key for docker repo fail to fetch from key server.
Please refer [Docker GPG error fix](<https://forums.docker.com/t/gpg-key-for-docker-repo-fail-to-fetch-from-key-server/24253>)

2.After kubeadm init, if you face any errors regarding swap memory and preflight checks please refer  [Kubernetes preflight error fix](<https://github.com/kubernetes/kubeadm/issues/610>)
