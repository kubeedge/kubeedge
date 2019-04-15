---
title: KubeEdge installer scope (Issue 324)
status: Pending
authors:
    - "@samy2019"
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

- `kubeedge cloud init`
- `kubeedge cloud reset`

For edge, commands shall be:

- `kubeedge node init`
- `kubeedge node reset`
- `kubeedge node join`

**NOTE:**
`node` key is used for edge component in the command, for superficial reasons. Because `kubeedge edge init` had `edge` used twice and didn't sound nice.

2. To support download and installation of pre-requisites for KubeEdge cloud and edge components.

## Out of scope

1. To support failures reported while execution of pre-requisites while execution of KubeEdge commands.
2. To support display of KubeEdge version.

# Scope of commands

## Design of the commands

### kubeedge --help or kubedge

```
ubuntu# kubedge --help or ubuntu# kubedge
Kubeedge

    ┌──────────────────────────────────────────────────────────┐
    │ KUBEEDGE                                                 │
    │ Easily bootstrap KubeEdge cluster                        │
    │                                                          │
    │ Please give us feedback at:                              │
    │ https://github.com/kubeedge/kubeedge/issues              │
    └──────────────────────────────────────────────────────────┘

Example usage:

    Create a two-machine cluster with one cloud node
    (which controls the edge cluster), and one edge node
    (where native containerized application, in the form of
    pods and deployments run), connects to devices.

    ┌──────────────────────────────────────────────────────────┐
    │ On the first machine:                                    │
    ├──────────────────────────────────────────────────────────┤
    │ cloud-node# kubeedge cloud init <arguments>              │
    └──────────────────────────────────────────────────────────┘

    ┌──────────────────────────────────────────────────────────┐
    │ On the second machine:                                   │
    ├──────────────────────────────────────────────────────────┤
    │ edge-node# kubeedge node join <arguments>                │
    └──────────────────────────────────────────────────────────┘

    You can then repeat the second step on as many other machines as you like.

Usage:
  kubeedge [command]

Available Commands:
  cloud       Cloud component command option for KubeEdge
  node        Edge component command option for KubeEdge
  version     Displays KubeEdge release version and code commit id.

Flags:
  -h, --help   help for kubeedge

Use "kubeedge [command] --help" for more information about a command.
ubuntu#
```

### kubeedge cloud --help

```
ubuntu# kubeedge cloud --help
<Apt description to be added while implementation>

Usage:
  kubeedge cloud [command]

Example usage:
<Apt information to be added while implementation>

Available Commands:
  init        Bootstraps cloud component. Checks and install (if required) the pre-requisites.
  reset       Teardowns cloud component.

Flags:
  -h, --help   help for cloud

Use "kubeedge cloud [command] --help" for more information about a command.
```

### kubeedge cloud init --help

```
kubeedge cloud init --help
<Apt description to be added while implementation>

Usage:
  kubeedge cloud init [flags]

Example usage:
<Apt information to be added while implementation>

Flags:
  -h, --help                        help for init
      --kubeedge-version   string   use this key to download and use the required KubeEdge version (Optional, default will be Latest)
      --kubernetes-version string   use this key to download and use the required Kubernetes version (Optional, default will be Latest)
      --docker-version     string   use this key to download and use the required Docker version (Optional, default will be Latest)
```

### kubeedge cloud reset --help

```
kubeedge cloud reset --help
<Apt description to be added while implementation>

Usage:
  kubeedge cloud reset

Example usage:
<Apt information to be added while implementation>

Flags:
  -h, --help   help for reset
  
```

### kubeedge node join --help

```
kubeedge node join --help
<Apt description to be added while implementation>

Usage:
  kubeedge node join [flags]

Example usage:
<Apt information to be added while implementation>

Flags:
  -h, --help                        help for join
      --certPath           string   downloaded path of the certifcates generated by cloud component in this host (Mandatory)
      --docker-version     string   use this key to download and use the required Docker version (Optional, default will be Latest)
      --kubeedge-version   string   use this key to download and use the required KubeEdge version (Optional, default will be Latest)
      --kubernetes-version string   use this key to download and use the required Kubernetes version (Optional, default will be Latest)
  -s, --server             string   ip:port address of cloud components host/VM (Mandatory)
```

## Explaining the commands

### Cloud commands

`kubeedge cloud init`
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

       **NOTE:** If any issues or error reported from `kubeadm init`, `kubeedge cloud init` command shall be not responsible, as it may occur due to the environment. User has to resolve it. We are taking up this activity to have 1 click install approach for KubeEdge and not K8S.

    6. Update `/etc/kubernetes/manifests/kube-apiserver.yaml` with below information
    ```
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

    7. start edge-controller

`kubeedge cloud reset`
  - What is it? 
    * This command will be responsible to bring down KubeEdge cloud components edge-controller and call `kubeadm reset` (to stop K8S)

  - What shall be its scope ?
    1. It shall get the nodes and execute `kubectl drain --force`.
    2. Kill `edge-controller` process
    3. Execute `kubeadm reset`

### Edge (node) commands

`kubeedge node join`
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

`kubeedge node reset`

  - What is it? 
    * This command will be responsible to bring down KubeEdge edge component (edge_core)

  - What it will do?

    1. Remove node using `kubectl` command
    2. Kill `edge_core` process