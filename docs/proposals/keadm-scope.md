---
title: KubeEdge installer scope (Issue 324)
status: implementable
authors:
    - "@samy2019"
    - "@srivatsav123"
approvers:
  - "@m1093782566"
  - "@rohitsardesai83"
  - "@sids-b"
creation-date: 2019-04-11
last-updated: 2019-05-20
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

- `keadm init`
- `keadm reset`

For edge, commands shall be:

- `keadm join`
- `keadm reset`
- `keadm diagnose`
- `keadm collect`
- `keadm check`
- `keadm get`
- `keadm describe`

**NOTE:**
`node` key is used for edge component in the command, for superficial reasons. Because `kubeedge edge init` had `edge` used twice and didn't sound nice.

2. To support download and installation of pre-requisites for KubeEdge cloud and edge components.

## Out of scope

1. To support failures reported while execution of pre-requisites while execution of KubeEdge commands.
2. To support display of KubeEdge version.

# Scope of commands

## Design of the commands

**NOTE**: All the below steps are executed as root user, to execute as sudo user. Please add `sudo` infront of all the commands.

### kubeedge --help or kubeedge

```
    ┌──────────────────────────────────────────────────────────┐
    │ KEADM                                                    │
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
    │ master node (on the cloud)#  keadm init <options>        │
    └──────────────────────────────────────────────────────────┘

    ┌──────────────────────────────────────────────────────────┐
    │ On the second machine:                                   │
    ├──────────────────────────────────────────────────────────┤
    │ worker node (at the edge)#  keadm join <options>         │
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

### keadm init --help

```
keadm init command bootstraps KubeEdge's cloud component.
It checks if the pre-requisites are installed already,
if not installed, this command will help in download,
installation and execution on the host.

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

### keadm reset --help

```
keadm reset command can be executed in both cloud and edge node
In cloud node it shuts down the cloud processes of KubeEdge
In edge node it shuts down the edge processes of KubeEdge

Usage:
keadm reset [flags]

Examples:

For cloud node:
keadm reset

For edge node:
keadm reset --k8sserverip 10.20.30.40:8080


Flags:
  -h, --help                 help for reset
  -k, --k8sserverip string   IP:Port address of cloud components host/VM

```

### keadm join --help

```

"keadm join" command bootstraps KubeEdge's edge component.
It checks if the pre-requisites are installed already,
If not installed, this command will help in download,
install and execute on the host.
It will also connect with cloud component to receieve
further instructions and forward telemetry data from
devices to cloud

Usage:
  keadm join [flags]

Examples:

keadm join --cloudcoreip=<ip address> --edgenodeid=<unique string as edge identifier>

  - For this command --cloudcoreip flag is a Mandatory flag
  - This command will download and install the default version of pre-requisites and KubeEdge

keadm join --cloudcoreip=10.20.30.40 --edgenodeid=testing123 --kubeedge-version=0.2.1 --k8sserverip=50.60.70.80:8080

  - In case, any option is used in a format like as shown for "--docker-version" or "--docker-version=", without a value
    then default values will be used.
    Also options like "--docker-version", and "--kubeedge-version", version should be in
    format like "18.06.3" and "0.2.1".


Flags:
      --docker-version string[="18.06.0"]          Use this key to download and use the required Docker version (default "18.06.0")
  -e, --cloudcoreip string                         IP address of KubeEdge CloudCore
  -i, --edgenodeid string                          KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own
  -h, --help                                       help for join
  -k, --k8sserverip string                         IP:Port address of K8S API-Server
      --kubeedge-version string[="0.3.0-beta.0"]   Use this key to download and use the required KubeEdge version (default "0.3.0-beta.0")

```

### keadm diagnose --help

```
keadm diagnose command can be help to diagnose specific fault scenarios in an all-round way and locate the cause of the fault.

Usage:
  keadm diagnose [command]

Examples:

# view the running status of node (key components such as sqlite, edgehub, metamanager, edged and many more)
keadm analysis node

Available Commands:
  all           All resource
  node          Troubleshoot the cause of edge node failure with installed software
  pod           Troubleshooting specific container application instances on nodes
  installation  It is same as "keadm check all"

```

### keadm check --help

```
keadm check command can be check whether the system specific items meet the requirements of edgecore installation and operation.

Usage:
  keadm check [command]

Available Commands:
  all      Check all
  arch     Determine the node hardware architecture whether support or not
  cpu      Determine if the NUMBER of CPU cores meets the requirement
  memory   Check the system memory size and the amount of memory left
  disk     Check whether the disk meets the requirements
  dns      Check whether the node domain name resolution function is normal
  docker   Check whether the node Docker function is normal
  network  Check whether the node can communicate with the endpoint on the cloud
  pid      Check if the current number of processes in the environment is too many. If the number of available processes is less than 5%, the number of processes is considered insufficient
  

Flags:
  -h, --help   help for keadm check

Use "keadm check [command] --help" for more information about a command
```



### keadm collect --help

```
Obtain all data of the current node, and then locate and use operation personnel.

Usage:
  keadm collect [flags]

Examples:

keadm collect --path . 

Flags:
  --path    Cache data and store data compression packages in a directory that defaults to the current directory
  --detail  Whether to print internal log output

```

### keadm get --help

```
"keadm get" command prints a table of the most important information about the specified resourcesv from sqlite db file.
You can filter the list using a label selector and the --selector flag. If the desired resource type 
is namespaced you will only see results in your current namespace unless you pass --all-namespaces.

Usage:
  keadm get [resource]
[(-o|--output=)json|yaml|wide|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=...]
(TYPE[.VERSION][.GROUP] [NAME | -l label] | TYPE[.VERSION][.GROUP]/NAME ...) [flags]

Examples:

# list all pod
keadm get pod

# list pod in namespace test
keadm get pod -n test

# List a single configmap  with specified NAME in ps output format.
keadm get configmap web -n default

# List the complete information of the configmap with the specified name in the yaml output format.
keadm get configmap web -n default -o yaml

Available resource:
  pod
  node
  service
  secret
  configmap
  endpoint
  persistentvolumesclaims

  
Flags:
  -A, --all-namespaces=false: If present, list the requested object(s) across all namespaces. Namespace in current
context is ignored even if specified with --namespace.
      --allow-missing-template-keys=true: If true, ignore any errors in templates when a field or map key is missing in
the template. Only applies to golang and jsonpath output formats.
      --chunk-size=500: Return large lists in chunks rather than all at once. Pass 0 to disable. This flag is beta and
may change in the future.
      --field-selector='': Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
key1=value1,key2=value2). The server only supports a limited number of field queries per type.
  -f, --filename=[]: Filename, directory, or URL to files identifying the resource to get from sqlite db file.
      --ignore-not-found=false: If the requested object does not exist the command will return exit code 0.
  -L, --label-columns=[]: Accepts a comma separated list of labels that are going to be presented as columns. Names are
case-sensitive. You can also use multiple flag options like -L label1 -L label2...
      --no-headers=false: When using the default or custom-column output format, don't print headers (default print
headers).
  -o, --output='': Output format. One of:
json|yaml|wide|name|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=...
See custom columns [http://kubernetes.io/docs/user-guide/kubectl-overview/#custom-columns], golang template
[http://golang.org/pkg/text/template/#pkg-overview] and jsonpath template
[http://kubernetes.io/docs/user-guide/jsonpath].
  -l, --selector='': Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
      --show-kind=false: If present, list the resource type for the requested object(s).
      --show-labels=false: When printing, show all labels as the last column (default hide labels column)
      --sort-by='': If non-empty, sort list types using this field specification.  The field specification is expressed
as a JSONPath expression (e.g. '{.metadata.name}'). The field in the API resource specified by this JSONPath expression
must be an integer or a string.
      --template='': Template string or path to template file to use when -o=go-template, -o=go-template-file. The
template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].

```


### keadm describe --help

```
Show details of a specific resource or group of resources

Print a detailed description of the selected resources, including related resources such as events or controllers. You
may select a single object by name, all objects of that type, provide a name prefix, or label selector. For example:

  $ kubectl describe TYPE NAME_PREFIX

 will first check for an exact match on TYPE and NAME_PREFIX. If no such resource exists, it will output details for
every resource that has a name prefixed with NAME_PREFIX.

Usage:
  keadm describe (-f FILENAME | TYPE [NAME_PREFIX | -l label] | TYPE/NAME) [options]

Examples:

  # Describe a node
  keadm describe nodes kubernetes-node-emt8.c.myproject.internal

  # Describe a pod
  keadm describe pods/nginx

  # Describe a pod identified by type and name in "pod.json"
  keadm describe -f pod.json

  # Describe all pods
  keadm describe pods

  # Describe pods by label name=myLabel
  keadm describe po -l name=myLabel


Available resource:
  pod
  node
  service
  secret
  configmap
  endpoint
  persistentvolumesclaims
  
Options:
  -A, --all-namespaces=false: If present, list the requested object(s) across all namespaces. Namespace in current
context is ignored even if specified with --namespace.
  -f, --filename=[]: Filename, directory, or URL to files containing the resource to describe
  -l, --selector='': Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
      --show-events=true: If true, display events related to the described object.

```



## Explaining the commands

### Master Node (on the Cloud) commands

`keadm init`
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
    5. Start `keadm init`.

       **NOTE:** Issues encountered while performing kubeadm init need to be resolved by the user
    6. Update `/etc/kubernetes/manifests/kube-apiserver.yaml` with below information
    ```
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

    7. start edge-controller

`keadm reset`
  - What is it?
    * This command will be responsible to bring down KubeEdge cloud components edge-controller and call `kubeadm reset` (to stop K8S)

  - What shall be its scope ?
    1. It shall get the nodes and execute `kubectl drain --force`.
    2. Kill `edge-controller` process
    3. Execute `kubeadm reset`

### Worker Node (at the Edge) commands

`keadm join`

  - What is it?
    * This command will be responsible to install pre-requisites and make modifications needed for KubeEdge edge component (edgecore) and start it

  - What shall be its scope ?

    1. Check version of OS and install subsequently the required pre-requisites using supported steps. Currently we will support **ONLY** (Ubuntu & CentOS)
    2. Check and install all the pre-requisites before executing edge-controller, which are
        * Docker (currently 18.06.0ce3-0~ubuntu) and check is service is up.
        * mosquitto (latest available in OS repos) and check if running.
    3. This command will take `--certPath` (string type) as mandatory option which shall be the certificates path; wherein the certs were transferred from cloud node and uncompressed. It will modify `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml` file against `edgehub.websocket.certfile` and `edgehub.websocket.keyfile` fields.
    4. Create `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` and apply it using `curl` command to api-server
    5. This command will take mandatory `-e` or `--cloudcoreip` flag to specify the address of Kubeedge cloudcore
    6. Create `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml`
        * Use `--cloudcoreip` flag to update the `websocket.url` field.
        * Use `--edgenodeid` flags value to update `controller.node-id`,`edged.hostname-override` field.
    7. Register or add node to K8S cluster, Using Flag `-k` or `--k8sserverip` value to connect with the api-server.
        * Create `node.json` file and update it with `-i` or `--edgenodeid` flags value in `metadata.name` field.
        * Apply it using `curl` command to api-server

    8. start edgecore

`keadm reset`

  - What is it?
    * This command will be responsible to bring down KubeEdge edge component (edgecore)

  - What it will do?

    1. Remove node using `curl` command from K8S cluster
    2. Kill `edgecore` process

`keadm diagnose`

- What is it?

  
- This command will be help to diagnose specific fault scenarios in an all-round way and locate the cause of the fault
  
- What shall be its scope ?
    1. Use command `all` can diagnose all resource
    2. Use command `node` can roubleshoot the cause of edge node failure with installed software
    3. Use command `pod` can troubleshooting specific container application instances on nodes
    4. Use command `installation` is same as "keadm check all"



`keadm check`

- What is it?
  
  - This command will be check whether the system specific items meet the requirements of edgecore installation and operation.
  
- What shall be its scope ?

  1. Check items include hardware resources or operating system resources (cpu, memory, disk, network, pid limit,etc.)

  2. Use command `arch` can check node hardware architecture:

     - x86_64 architecture
       Ubuntu 16.04 LTS (Xenial Xerus), Ubuntu 18.04 LTS (Bionic Beaver), CentOS 7.x and RHEL 7.x, Galaxy Kylin 4.0.2, ZTE new fulcrum v5.5, winning the bid Kylin v7.0

     - armv7i (arm32) architecture
       Raspbian GNU/Linux 9 (stretch)

     - aarch64 (arm64) architecture
       Ubuntu 18.04.2 LTS (Bionic Beaver)

  3. Use command `cpu` can cetermine if the NUMBER of CPU cores meets the requirement, minimum 1Vcores.

  4. Use command `memory` check the system memory size and the amount of memory left, requirements minimum 256MB.

  5. Use command `disk` check whether the disk meets the requirements, requirements minimum 1 GB.

  6. Use command `dns` Check whether the node domain name resolution function is normal.

  7. Use command `docker `  Check whether the node Docker function is normal, the Docker version must be higher than 17.06, it is recommended to use the 18.06.3 version.


  8. Use command `network `  check whether the node can communicate with the endpoint on the cloud,  default to ping clusterdns.

  11. Use command `pid ` check if the current number of processes in the environment is too many. If the number of available processes is less than 5%, the number of processes is considered insufficient.




`keadm collect`

- What is it?

  - This command will be obtain all related data of the current node, and then locate and use  operation personnel.

- What shall be its scope ?

  1. system data

    - Hardware architecture

      Collect arch command output and determine the type of  installation

    - CPU information

      Parse the /proc/cpuinfo file and output the cpu information file

    - Memory information

      Collect free -h command output

    - Hard disk information

      Collect df -h command output, and mount command output

    - Internet Information

      Collect netstat -anp command output and copy /etc/resolv.conf and /etc/hosts files

    - Process information

      Collect ps -aux command output

    - Time information

      Collect date and uptime command output

    - History command input

      Collect all the commands entered by the current user

  2. Edgecore data

  - database data

    Copy the /var/lib/kubeedge/edgecore.db file

  - log files

    Copy all files under /var/*log*/*kubeedge*/

  - service file

    Copy the edgecore.service, edgelogger.service, edgemonitor.service, edgedaemon.service files under /lib/systemd/system/

  - software version

  - certificate

    Copy all files under /etc/kubeedge/certs/

  - Edge-Core configuration file in  software (including Edge-daemon)

    Copy all files under /etc/kubeedge/config/

  3. docker data

  - Docker version information

    Collect docker version command output

  - Docker information

    Collect docker info command output

  - Docker log information

    Collect journalctl -u docker.service command output

  - Docker container information

    Collect docker ps -a command output

  - Docker container configuration and log information

    Copy all files under /var/lib/docker/containers

  - Docker image information

    Collect docker images command output


`keadm get`

- What is it?
  
  - This command can be used to get some resources
  
- What shall be its scope ?

  1. use `<resource>` parameter can obtain specific resources，resource include pod,configmap,secret,node etc. that can be seen at the edge node

  2. Use `-n <namespace>` flag to select namespace and use `--all-namespace` flag to show all namespace.

  3. Use `-l <selector>` flag to selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)

  4. ...

     

`keadm describe`

- What is it?
  
  - This command can be used to obtain part of the resource description
- What shall be its scope ?

  1. use `<resource> <resource_name>` parameter can obtain specific resources,resource include pod,configmap,secret,node etc. that can be seen at the edge node

     
  