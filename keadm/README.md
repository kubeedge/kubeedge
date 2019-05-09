# Getting Started with KubeEdge Installer

Please refer to KubeEdge Installer proposal document for details on the motivation of having KubeEdge Installer.
It also explains the functionality of the proposed commands.
[KubeEdge Installer Doc](<https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/kubeedgeadm-scope.md/>)

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
4. Binary `kubeedge` is available in current path

## Installing KubeEdge Cloud component

Referring to `KubeEdge Installer Doc`, the command to install KubeEdge cloud component (edge controller) and pre-requisites

- Execute `kubeedge init`

**NOTE:**
Device CRD yamls need to be applied in a K8S cluster for device management. Support for deploying these CRD's as a part of the installer will be available in the next release.

### Command flags
The optional flags with this command are mentioned below

```
$  kubeedge init --help

kubeedge init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.

Usage:
  kubeedge init [flags]

Examples:

kubeedge init


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
kubeedge init --docker-version=<expected version> --kubernetes-version=<expected version> --kubeedge-version=<expected version>
```

**NOTE:**
Version mentioned as defaults for Docker and K8S are being tested with.

## Installing KubeEdge Edge component

Referring to `KubeEdge Installer Doc`, the command to install KubeEdge Edge component (edge core) and pre-requisites

- Execute `kubeedge join <flags>`

### Command flags

The optional flags with this command are shown in below shell

```
$  kubeedge join --help
 
"kubeedge join" command bootstraps KubeEdge's edge component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
to install the prerequisites.
It will help the edge node to connect to the cloud.


Usage:
  kubeedge join [flags]

Examples:

kubeedge join --server=<ip:port>

  - For this command --server option is a Mandatory option
  - This command will download and install the default version of pre-requisites and KubeEdge

kubeedge join --server=10.20.30.40:8080 --docker-version= --kubeedge-version=0.2.1 --kubernetes-version=1.14.1
kubeedge join --server=10.20.30.40:8080 --docker-version --kubeedge-version=0.2.1 --kubernetes-version=1.14.1
  
  - Default values for --docker-version=18.06.0,--kubernetes-version=1.14.1, --kubeedge-version=0.3.0-beta.0 
  - In case, any option is used in a format like as shown for "--docker-version" or "--docker-version=", without a value
  
Flags:
      --docker-version string[="18.06.0"]          Use this key to download and use the required Docker version (default "18.06.0")
  -h, --help                                       help for join
      --kubeedge-version string[="0.3.0-beta.0"]   Use this key to download and use the required KubeEdge version (default "0.3.0-beta.0")
      --kubernetes-version string[="1.14.1"]       Use this key to download and use the required Kubernetes version (default "1.14.1")
  -s, --server string                              IP:Port address of cloud components host/VM

```

1. For Docker, K8S and KubeEdge flags the functionality is same as mentioned in `kubeedge init`
2. -s, --server, It should be in the format <IPAddress:Port>, where the default port is 8080. Please see the example above.
                It is a mandatory flag

**IMPORTANT NOTE:** The versions used for Docker, KubeEdge and K8S, should be same in both Cloud and Edge side.

## Reset KubeEdge Cloud and Edge components

Referring to `KubeEdge Installer Doc`, the command to stop KubeEdge cloud (edge controller). It doesn't uninstall/remove any of the pre-requisites.

- Execute `kubeedge reset`

### Command flags

```
kubeedge reset --help

kubeedge reset command can be executed in both cloud and edge node
In cloud node it shuts down the cloud processes of KubeEdge
In edge node it shuts down the edge processes of KubeEdge

Usage:
  kubeedge reset [flags]

Examples:

For cloud node:
kubeedge reset

For edge node:
kubeedge reset --server 10.20.30.40:8080
    - For this command --server option is a Mandatory option


Flags:
  -h, --help            help for reset
  -s, --server string   IP:Port address of cloud components host/VM

```

## Simple steps to bring up a KubeEdge setup and deploy a pod
**NOTE:** All the below steps are executed as root user, to execute as sudo user ,Please add **sudo** infront of all the commands
### 1.EdgeController (With K8S API-Server)
##### Install tools with the particular version
```
kubeedge init --kubeedge-version=<kubeedge Version>  --kubernetes-version=<kubernetes Version> --docker-version=<Docker version>
```
##### Install tools with the Default version version
```
kubeedge init --kubeedge-version= --kubernetes-version= --docker-version
or
kubeedge init 
```

**NOTE:**
kubeadm join **192.168.20.134**:6443 --token 2lze16.l06eeqzgdz8sfcvh \
         --discovery-token-ca-cert-hash sha256:1e5c808e1022937474ba264bb54fea42b05eddb9fde2d35c9cad5b83cf5ef9ac  
         After Kubeedge init ,please note the **CloudIp** as highlighted above generated from console output and port is **8080**.

### 2.Manually copy Certs.tgz. to /etc/kubeedge in Edge vm

In Edge VM
```
mkdir -p /etc/kubeedge
```
In Cloud VM

```
scp -r certs.tgz username@ipEdgevm:/etc/kubeedge
```

In Edge VM untar the certs,tgz file

```
cd /etc/kubeedge
tar -xvzf certs.tgz
```

### 3.Edge Node join
##### Install tools with the Default version 
```
kubeedge join --kubeedge-version=<kubeedge Version>  --kubernetes-version=<kubernetes Version> --docker-version=<Docker version> --server=<CloudIp:8080>
```

```
kubeedge join --kubeedge-version= --kubernetes-version= --docker-version= --server=CloudIp:8080
or
kubeedge join --server=CloudIp:8080
```
**Note**:Cloud ip refers to Ip generated ,from the step 1 as highlighted 

### 4.Node status on EdgeController console
In the Cloud Vm run,
```
kubectl get nodes

NAME             STATUS     ROLES    AGE     VERSION
192.168.20.135   Ready      <none>   6s      0.3.0-beta.0
```
Check if the edge node is in ready state

### 5.Deploy a sample pod from Cloud VM
**https://github.com/kubeedge/kubeedge/blob/master/build/deployment.yaml**

copy the deployment.yaml from the above link in cloud Vm,run
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

### Errata
1.If GPG key for docker repo fail to fetch from key server.
Please refer [Docker GPG error fix](<https://forums.docker.com/t/gpg-key-for-docker-repo-fail-to-fetch-from-key-server/24253>)


2.After kubeadm init, if you face any errors regarding swap memory and preflight checks please refer  [Kubernetes preflight error fix](<https://github.com/kubernetes/kubeadm/issues/610>)
