---
title: KubeEdge installer scope (Issue 324)
status: Alpha
authors:
    - "@samy2019"
    - "@srivatsav123"
approvers:
  - "@m1093782566"
  - "@rohitsardesai83"
  - "@sids-b"
creation-date: 2019-04-11
last-updated: 2019-04-15
---

# Motivation

Many users shared their feedback that kubeEdge installation is too complicated and it may prevent people from trying kubeEdge. There should a simplified way to have **Getting Started with KubeEdge**, so that user can concentrate more on using it instantly, rather than
getting entangled in the installation steps.

# Proposal

KubeEdge shall have simple commands and steps to bring up both cloud and edge components.
The user experience in **Getting Started with KubeEdge** will be seamless.
Hence proposing the following commands for KubeEdge installation process.

## Inscope

1. To support first set of basic commands (listed below) to bootstrap and teardown both KubeEdge cloud and edge (node) components in different VM's or hosts.

For cloud, commands shall be:

- `kubeedge init`
- `kubeedge reset`

For edge, commands shall be:

- `kubeedge join`
- `kubeedge reset`

**NOTE:**
`node` key is used for edge component in the command, for superficial reasons. Because `kubeedge edge init` had `edge` used twice and didn't sound nice.

2. To support download and installation of pre-requisites for KubeEdge cloud and edge components.

## Out of scope

1. To support failures reported while execution of pre-requisites while execution of KubeEdge commands.
2. To support display of KubeEdge version.

# Scope of commands

## Design of the commands

**NOTE**: All the below steps are executed as root user, to execute as sudo user ,Please add sudo infront of all the commands

### kubeedge --help or kubeedge

```
    ┌──────────────────────────────────────────────────────────┐
    │ KubeEdge                                                 │
    │ Easily bootstrap a KubeEdge cluster                      │
    │                                                          │
    │ Please give us feedback at:                              │
    │ https://github.com/kubeedge/kubeedge/issues              │
    └──────────────────────────────────────────────────────────┘
	
    Create a two-machine cluster with one cloud node
    (which controls the edge node cluster), and one edge node
    (where native containerized application, in the form of
    pods and deployments run), connects to devices.

Usage:
  kubeedge [command]

Examples:

    ┌──────────────────────────────────────────────────────────┐
    │ On the first machine:                                    │
    ├──────────────────────────────────────────────────────────┤
    │ cloud-node#  kubeedge init <options>                 │
    └──────────────────────────────────────────────────────────┘

    ┌──────────────────────────────────────────────────────────┐
    │ On the second machine:                                   │
    ├──────────────────────────────────────────────────────────┤
    │ edge-node#  kubeedge join <options>                  │
    └──────────────────────────────────────────────────────────┘

    You can then repeat the second step on as many other machines as you like.


Available Commands:
  help        Help about any command
  init        Bootstraps cloud component. Checks and install (if required) the pre-requisites.
  join        Bootstraps edge component. Checks and install (if required) the pre-requisites.
              Execute it on any edge node machine you wish to join
  reset       Teardowns KubeEdge (cloud & edge) component

Flags:
  -h, --help   help for kubeedge

Use "kubeedge [command] --help" for more information about a command.
```

### kubeedge init --help

```
kubeedge init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
if not installed, this command will help in download,
installation and execution on the host.

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

### kubeedge reset --help

```
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
  -s, --server string   IP:Port address of k8s master 
  
```

### kubeedge join --help

```

"kubeedge join" command bootstraps KubeEdge's edge component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
It will also connect with cloud component to receieve 
further instructions and forward telemetry data from 
devices to cloud

Usage:
  kubeedge join [flags]

Examples:

kubeedge join --server=<ip:port>

  - For this command --server option is a Mandatory option
  - This command will download and install the default version of pre-requisites and KubeEdge

kubeedge join --server=10.20.30.40:8080 --docker-version=18.06.0 --kubeedge-version=0.2.1 --kubernetes-version=1.14.1
kubeedge join --server=10.20.30.40:8080 --docker-version --kubeedge-version=0.2.1 --kubernetes-version=1.14.1
  - Default values for --docker-version=18.06.0,--kubernetes-version=1.14.1, --kubeedge-version=0.3.0-beta.0
  - In case, any option is used in a format like as shown for "--docker-version" or "--docker-version=", without a value
		then default values will be used. 
		
Flags:
      --docker-version string[="18.06.0"]          Use this key to download and use the required Docker version (default "18.06.0")
  -h, --help                                       help for join
      --kubeedge-version string[="0.3.0-beta.0"]   Use this key to download and use the required KubeEdge version (default "0.3.0-beta.0")
      --kubernetes-version string[="1.14.1"]       Use this key to download and use the required Kubernetes version (default "1.14.1")
  -s, --server string                              IP:Port address of cloud components host/VM

```

## Explaining the commands

### Cloud commands

`kubeedge init`
  - What is it?
     * This command will be responsible to bring up KubeEdge cloud components like edge-controller and K8S (using kubeadm)
   
  - What shall be its scope ?
    1. Check version of OS and install subsequently the required pre-requisites using supported steps. Currently we will support **ONLY** (Ubuntu & CentOS)
    2. Check and install all the pre-requisites before executing edge-controller, which are
        * docker (currently 18.06.0ce3-0~ubuntu) and check if service is up
        * kubelet, kubeadm & kubectl (latest version)
        * openssl (latest available in OS repos)
    3. Generate certificates using openssl and save the certs in a predefined static path.
    It will also compress the folder and display on the terminal so that user can pick it up and transfer it to edge node (VM/host) manually.
    4. It will update the certificate information in `controller.yaml`
    5. Start `kubeadm init`.

       **NOTE:** Issues encountered while performing kubeadm init need to be resolved by the user
    6. Update `/etc/kubernetes/manifests/kube-apiserver.yaml` with below information
    ```
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

    7. start edge-controller

`kubeedge reset`
  - What is it? 
    * This command will be responsible to bring down KubeEdge cloud components edge-controller and call `kubeadm reset` (to stop K8S)

  - What shall be its scope ?
    1. It shall get the nodes and execute `kubectl drain --force`.
    2. Kill `edge-controller` process
    3. Execute `kubeadm reset`

### Edge (node) commands

`kubeedge join`
  - What is it? 
    * This command will be responsible to install pre-requisites and make modifications needed for KubeEdge edge component (edge_core) and start it

  - What shall be its scope ?

    1. Check version of OS and install subsequently the required pre-requisites using supported steps. Currently we will support **ONLY** (Ubuntu & CentOS)
    2. Check and install all the pre-requisites before executing edge-controller, which are
        * Docker (currently 18.06.0ce3-0~ubuntu) and check is service is up.
        * mosquitto (latest available in OS repos) and check if running.
        * kubectl
    3. This command will take `--certPath` (string type) as mandatory option which shall be the certificates path; wherein the certs were transfered from cloud node and uncompressed. It will modify `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml` file against `edgehub.websocket.certfile` and `edgehub.websocket.keyfile` fields.
    4. Modify `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` and apply it using `kubectl` command to api-server
    5. This command will take mandatory `-s` or `--server` flag to specify the address and port of the Kubernetes API server
    6. Modify `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml`
        * Update the IP address of the master in the `websocket.url` field.
        * Replace `fb4ebb70-2783-42b8-b3ef-63e2fd6d242eq` with edge node ip in `edge.yaml` for the fields: `controller.node-id`,`edged.hostname-override`
        * In `websocket.URL`, replace `0.0.0.0` with server ip from `-s` option.
    7. Register or add node to master, Using Flag `-s` or `--server` mandatory field, it will connect with the master (api-server). Modify `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` and apply it using `kubectl` command to api-server

      **NOTE:** you can use the `-s` or `--server` flags to specify the address and port of the Kubernetes API server. Refer [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/)

    4. start edge_core

`kubeedge reset`

  - What is it? 
    * This command will be responsible to bring down KubeEdge edge component (edge_core)

  - What it will do?

    1. Remove node using `kubectl` command
    2. Kill `edge_core` process
