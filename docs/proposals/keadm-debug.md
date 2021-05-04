---
title: KubeEdge debug (Issue 324)
status: Pending
authors:
    - "@shenkonghui"
    - "@qingchen1203"
approvers:
    -
creation-date: 2020-07-15
last-updated:  2020-11-09
---

# Motivation

Many users shared their feedback that kubeEdge edge nodes do not have good debugging and diagnostic methods, which may prevent people from trying kubeEdge.
There should be a simple and clear way to help operation and maintenance personnel to ensure the stable operation of kubeEdge, so that users can focus more on using it immediately.

## Goal

- Alpha
Collect the full amount of information related to kubeedge in the current environment, and provide it to O&M personnel in a formatted way to locate and solve difficult problems.

- Beta
  1. Diagnose specific fault scenarios in an all-roundly way and locate the cause of the fault.
  2. Check whether the system specific items meet the requirements of edgecore installation and operation.

# Proposal

KubeEdge should have simple commands to debug and troubleshoot edge components.
Therefore, it is recommended to use the following commands during operation and maintenance of KubeEdge.

## Inscope

1. To support first set of basic commands (listed below) to debug edge (node) components.

For edge, commands shall be:

  - `keadm debug help`
  - `keadm debug diagnose`
  - `keadm debug collect`
  - `keadm debug check`
  - `keadm debug get`

# Scope of commands

## Design of the commands

**NOTE**: All the below steps are executed as root user, to execute as sudo user. Please add `sudo` before all commands.

### keadm debug  --help

```
"keadm debug" command help  provide debug function to help diagnose the cluster

Usage:
  keadm debug [command]

Available Commands:
  check       Check specific information.
  collect     Obtain all the data of the current node
  diagnose    Diagnose relevant information at edge nodes
  get         Display one or many resources

Flags:
  -h, --help   help for debug

Use "keadm debug [command] --help" for more information about a command.

```

### keadm debug diagnose --help

```
keadm debug diagnose command Diagnose relevant information at edge nodes

Usage:
  keadm debug diagnose [command]

Examples:

# Diagnose whether the node is normal
keadm debug diagnose node

# Diagnose whether the pod is normal
keadm debug diagnose pod nginx-xxx -n test

# Diagnose node installation conditions
keadm debug diagnose install

# Diagnose node installation conditions and specify the detected ip
keadm debug diagnose install -i 192.168.1.2


Available Commands:
  install     Diagnose install
  node        Diagnose edge node
  pod         Diagnose pod

Flags:
  -h, --help   help for diagnose

Use "keadm debug diagnose [command] --help" for more information about a command.
```

### keadm debug check --help

```
Obtain all the data of the current node, and then provide it to the operation
and maintenance personnel to locate the problem

Usage:
  keadm debug check [command]

Examples:

        # Check all items .
        keadm debug check all

        # Check whether the node CPU meets  requirements.
        keadm debug check cpu

        # Check whether the node memory meets  requirements.
        keadm debug check mem

        # check whether the node disk meets  requirements.
        keadm debug check disk

        # Check whether the node DNS can resolve a specific domain name.
        keadm debug check dns -d www.github.com

        # Check whether the node network meets requirements.
        keadm debug check network

        # Check whether the number of free processes on the node meets requirements.
        keadm debug check pid

        # Check whether runtime(Docker) is installed on the node.
        keadm debug check runtime

Available Commands:
  all         Check all item
  cpu         Check node CPU requirements
  disk        Check node disk requirements
  dns         Check whether DNS can work
  mem         Check node memory requirements
  network     Check whether the network is normal
  pid         Check node PID requirements
  runtime     Check whether runtime can work

Flags:
  -h, --help   help for check

Use "keadm debug check [command] --help" for more information about a command.
```

### keadm debug collect --help

```
Collect all the data of the current node, and then Operations Engineer can use them to debug.

Usage:
  keadm debug collect [flags]

Examples:
keadm debug collect --path .

# Collect all items and specified the output directory path
keadm debug collect --output-path .

Flags:
  -c, --config string        Specify configuration file, defalut is /etc/kubeedge/config/edgecore.yaml (default "/etc/kubeedge/config/edgecore.yaml")
  -d, --detail               Whether to print internal log output
  -h, --help                 help for collect
  -l, --log-path string      Specify log file (default "/var/log/kubeedge/")
  -o, --output-path string   Cache data and store data compression packages in a directory that default to the current directory (default ".")
```

### keadm debug get --help

```
Prints a table of the most important information about the specified resource from the local database of the edge node.

Usage:
  keadm debug get [flags]

Examples:

# List all pod in namespace test
keadm debug get pod -n test
# List a single configmap  with specified NAME
keadm debug get configmap web -n default
# List the complete information of the configmap with the specified name in the yaml output format
keadm debug get configmap web -n default -o yaml
# List the complete information of all available resources of edge nodes using the specified format (default: yaml)
keadm debug get all -o yaml

Flags:
  -A, --all-namespaces       List the requested object(s) across all namespaces
  -p, --edgedb-path string   Indicate the edge node database path, the default path is "/var/lib/kubeedge/edgecore.db" (default "/var/lib/kubeedge/edgecore.db")
  -h, --help                 help for get
  -n, --namespace string     List the requested object(s) in specified namespaces (default "default")
  -o, --output string        Indicate the output format. Currently supports formats such as yaml|json|wide
  -l, --selector string      Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)

```

## Explaining the commands

### Worker Node (at the Edge) commands

`keadm debug diagnose`

- What is it?

- This command will be help to diagnose specific fault scenarios in an all-roundly way and locate the cause of the fault

- What shall be its scope ?
    1. Use command `all` can diagnose all resource
    2. Use command `node` can troubleshoot the cause of edge node failure with installed software
       1. check system resources is enough
       2. check container runtime is running
       3. check all edgecore components are running
       4. check all database is exist
       5. confirm that the configuration file exists
       6. check cloudercore can be connected
    3. Use command `pod` can troubleshooting specific container application instances on nodes
       1. check node status
       2. check pod msg in database
       3. check pod status and containerStatus and print key messages
    4. Use command `install` is same as "keadm debug check all"

`keadm debug check`

- What is it?

  - This command will be check whether the system specific items meet the requirements of edgecore installation and operation.

- What shall be its scope ?
  1. Check items include hardware resources or operating system resources (cpu, memory, disk, network, pid limit,etc.)
  2. Use command `cpu` can determine if the NUMBER of CPU cores meets the requirement, minimum 1Vcores. The current usage ratio should be less than 90%
  3. Use command `memory` check the system memory size, and the amount of memory left, requirements minimum 256MB.The current usage ratio should be less than 90% and reserve 128 MB.
  4. Use command `disk` check whether the disk meets the requirements, requirements minimum 1 GB.The current usage ratio should be less than 90% and reserve 512MB.
  5. Use command `dns` check whether the node domain name resolution is normal, can use parameters `-d` or `-D` to specify the domain or DNS address of the test
  6. Use command `runtime ` check whether the node container runtime is installed, can use parameter `-r` to set container runtime, default is docker
  7. Use command `network ` check whether the node can communicate with the endpoint on the cloud, can use parameter `-i` to set test ip, default to ping cloudcore.
  8. Use command `pid ` check if the current number of processes in the environment is too many. If the number of available processes is less than 5%, the number of processes is considered insufficient.

`keadm debug collect`

- What is it?

  - This command will obtain all the data of the current node as `edge-$date.tar.gz`, and provide it to the operation and maintenance personnel to locate the problem.

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

    Copy the edgecore.service files under /lib/systemd/system/

  - software version

  - certificate

    Copy all files under /etc/kubeedge/certs/

  - Edge-Core configuration file in  software

    Copy all files under /etc/kubeedge/config/

  3. Container runtime data

  - runtime version information

  - runtime container information

  - runtime log information

  - runtime configuration and log information

  - runtime image information

`keadm debug get`

- What is it?

  - This command will get and format the specified resource`s information from the local database of the edge node

- What shall be its scope ?

  1. Format resource information from the local database, and available resource types:
    - `all`
    - `pod`
    - `node`
    - `service`
    - `secret`
    - `configmap`
    - `endpoint`
  2. Use flag `-n, --namespace=''` to indicate the scope of resource acquisition, if the flag `-A, --all-namespaces` is used, information of the specified resource will be obtained from all ranges
  3. Use flag `-o, --output=''` to indicate output format of the information
  4. Use flag `-l, --selector=''` to indicate which specified field is used to filter the data in the range
  5. Use flag `-p, --edgedb-path''` to indicate the edge node database path, default to `/var/lib/kubeedge/edgecore.db`

## Example
### keadm debug get

get pod list
```
[root@localhost bin]# keadm debug get pod
NAME                               READY   STATUS               RESTARTS   AGE
nginx-ds-85jch                     1/1     Running              4          21d
nginx-deployment-dbbffc676-wprs8   0/1     ContainerCannotRun   98         5d21h
```

get pod  specify the database file
```
[root@localhost tmp]# keadm debug get pod  -p /var/lib/kubeedge/edgecore.db
NAME                               READY   STATUS               RESTARTS   AGE
nginx-ds-85jch                     1/1     Running              5          21d
nginx-deployment-dbbffc676-wprs8   0/1     ContainerCannotRun   105        5d22h
```

get pod with wide
```
[root@localhost bin]# keadm debug get pod -o wide
NAME                               READY   STATUS               RESTARTS   AGE     IP           NODE                     NOMINATED NODE   READINESS GATES
nginx-ds-85jch                     1/1     Running              4          21d     172.17.0.3   centos-kubeedge.shared   <none>           <none>
nginx-deployment-dbbffc676-wprs8   0/1     ContainerCannotRun   98         5d21h   172.17.0.4   centos-kubeedge.shared   <none>           <none>
```

get one pod with json
```
[root@localhost bin]# keadm debug get pod nginx-ds-85jch -o json
{
    "apiVersion": "v1",
    "kind": "pod",
    "metadata": {
        "creationTimestamp": "2020-10-19T02:55:07Z",
        "generateName": "nginx-ds-",
        "labels": {
            "controller-revision-hash": "69b66994b8",
            "name": "nginx-ds",
            "pod-template-generation": "1"
        },
        ...
    },
    "spec": {
    ...
    },
    "status": {
      ...
    }
}
```

get pod with all namespace
```
[root@localhost bin]# keadm debug get pod -A
NAMESPACE     NAME                               READY   STATUS               RESTARTS   AGE
kube-system   calico-node-9jh2l                  0/1     Running              4          21d
kube-system   kube-proxy-2qrdt                   1/1     Running              0          73d
default       nginx-ds-85jch                     1/1     Running              4          21d
default       nginx-deployment-dbbffc676-wprs8   0/1     ContainerCannotRun   98         5d22h
```

get configmap with all namespace
```
[root@localhost bin]# keadm debug get cm -A


NAMESPACE       NAME            DATA   AGE
calico-system   typha-ca        1      118d
kube-system     kube-proxy      2      118d
calico-system   cni-config      1      118d
kube-system     calico-config   4      116d
```

get secret with all namespace
```
[root@localhost tmp]# keadm debug get secret -A


NAMESPACE       NAME                                  TYPE                                  DATA   AGE
kubeedge        default-token-44mgz                   kubernetes.io/service-account-token   3      118d
calico-system   calico-node-token-q28zz               kubernetes.io/service-account-token   3      118d
calico-system   calico-typha-token-2jnz2              kubernetes.io/service-account-token   3      118d
calico-system   calico-typha-token-xv4qk              kubernetes.io/service-account-token   3      116d
calico-system   calico-node-token-8kjcp               kubernetes.io/service-account-token   3      116d
calico-system   calico-kube-controllers-token-hkz7h   kubernetes.io/service-account-token   3      116d
kube-system     calico-kube-controllers-token-v4kfx   kubernetes.io/service-account-token   3      116d
calico-system   node-certs                            Opaque                                3      116d
calico-system   typha-certs                           Opaque                                3      116d
kube-system     kube-proxy-token-nqnvx                kubernetes.io/service-account-token   3      118d
kube-system     calico-node-token-lfmzn               kubernetes.io/service-account-token   3      116d
default         default-token-w5skc                   kubernetes.io/service-account-token   3      118d
```

get ep
```
[root@localhost tmp]# keadm debug get ep



NAME         ENDPOINTS                                            AGE
kubernetes   10.211.55.6:6443                                     118d
nginx        192.168.1.209:80,192.168.1.235:80,192.168.1.237:80   73d
```
### keadm debug collect

collect data
```
[root@localhost tmp]# keadm debug collect
Start collecting data
Data collected successfully, path: /root/tmp/edge_2020_1109_173744.tar.gz
```

collect data and print detail
```
[root@localhost tmp]# keadm debug collect
Start collecting data
Data collected successfully, path: /root/tmp/edge_2020_1109_173744.tar.gz
[root@localhost tmp]# keadm debug collect -d
Start collecting data
create tmp file: /tmp/edge_2020_1109_173800
create tmp file: /tmp/edge_2020_1109_173800/system
Execute Shell: arch > /tmp/edge_2020_1109_173800/system/arch
Copy File: cp -r /proc/cpuinfo /tmp/edge_2020_1109_173800/system/
Copy File: cp -r /proc/meminfo /tmp/edge_2020_1109_173800/system/
Execute Shell: df -h > /tmp/edge_2020_1109_173800/system/disk
Copy File: cp -r /etc/hosts /tmp/edge_2020_1109_173800/system/
Copy File: cp -r /etc/resolv.conf /tmp/edge_2020_1109_173800/system/
Execute Shell: ps -axu > /tmp/edge_2020_1109_173800/system/process
Execute Shell: date > /tmp/edge_2020_1109_173800/system/date
Execute Shell: uptime > /tmp/edge_2020_1109_173800/system/uptime
Execute Shell: history -a && cat ~/.bash_history  > /tmp/edge_2020_1109_173800/system/history
Execute Shell: netstat -pan > /tmp/edge_2020_1109_173800/system/network
collect systemd data finish
create tmp file: /tmp/edge_2020_1109_173800/edgecore
Copy File: cp -r /var/lib/kubeedge/edgecore.db /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /var/log/kubeedge/ /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /lib/systemd/system/edgecore.service /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /etc/kubeedge/config/ /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /etc/kubeedge/certs/server.crt /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /etc/kubeedge/certs/server.key /tmp/edge_2020_1109_173800/edgecore/
Copy File: cp -r /etc/kubeedge/ca/rootCA.crt /tmp/edge_2020_1109_173800/edgecore/
Execute Shell: edgecore  --version > /tmp/edge_2020_1109_173800/edgecore/version
collect edgecore data finish
create tmp file: /tmp/edge_2020_1109_173800/runtime
Copy File: cp -r /lib/systemd/system/docker.service /tmp/edge_2020_1109_173800/runtime/
Execute Shell: docker version > /tmp/edge_2020_1109_173800/runtime/version
Execute Shell: docker info > /tmp/edge_2020_1109_173800/runtime/info
Execute Shell: docker images > /tmp/edge_2020_1109_173800/runtime/images
Execute Shell: docker ps -a > /tmp/edge_2020_1109_173800/runtime/containerInfo
Execute Shell: journalctl -u docker  > /tmp/edge_2020_1109_173800/runtime/log
collect runtime data finish
Data compressed successfully
Remove tmp data finish
Data collected successfully, path: /root/tmp/edge_2020_1109_173800.tar.gz
```

specify output directory
```
[root@localhost tmp]# keadm debug collect -o /tmp
Start collecting data
Data collected successfully, path: /tmp/edge_2020_1109_173842.tar.gz
```

specify configuration file
```
[root@localhost tmp]# keadm debug collect -c /etc/kubeedge/config/edgecore.yaml
Start collecting data
Data collected successfully, path: /root/tmp/edge_2020_1109_173931.tar.gz
```
### keadm debug check

check cpu
```
[root@localhost tmp]# keadm debug check  cpu
CPU total: 1 core, Allowed > 1 core
CPU usage rate: 0.00, Allowed rate < 0.9

|-----------------|
|check cpu succeed|
|-----------------|
```

check mem
```
[root@localhost tmp]# keadm debug check  mem
Memory total: 1833.33 MB, Allowed > 256 MB
Memory Free total: 1074.40 MB, Allowed > 128 MB
Memory usage rate: 0.11, Allowed rate < 0.9

|-----------------|
|check mem succeed|
|-----------------|
```


check disk
```
[root@localhost tmp]# keadm debug check  disk
Disk total: 50268.47 MB, Allowed > 1024 MB
Disk Free total: 39087.80 MB, Allowed > 512MB
Disk usage rate: 0.18, Allowed rate < 0.9

|------------------|
|check disk succeed|
|------------------|
```

check dns
```
[root@localhost tmp]# keadm debug check  dns
dns resolution success, domain: www.github.com ip: 192.30.255.113

|-----------------|
|check dns succeed|
|-----------------|
```

check dns with k8s service name
```
[root@k8s ~]# kubectl get svc -n kube-system |grep dns
kube-dns                   ClusterIP   10.96.0.10       <none>        53/UDP,53/TCP,9153/TCP   118d
[root@k8s ~]# kubectl get svc
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   10.96.0.1       <none>        443/TCP   118d
nginx        ClusterIP   10.97.180.216   <none>        80/TCP    73d
[root@localhost tmp]# keadm debug check  dns -D 10.96.0.10 -d nginx
dns resolution success, domain: nginx ip: 10.97.180.216

|-----------------|
|check dns succeed|
|-----------------|
```

check network
```
[root@localhost tmp]# keadm debug check network
ping 172.17.0.1 success
check cloudhubServer 10.211.55.6:10000 success
check edgecoreServer 127.0.0.1:10350 success

|---------------------|
|check network succeed|
|---------------------|
```

check pid
```
Maximum PIDs: 32768; Running processes: 129

|-----------------|
|check pid succeed|
|-----------------|
```

check runtime
```
[root@localhost tmp]# keadm debug check runtime
docker is running

|---------------------|
|check runtime succeed|
|---------------------|
```

check all  and at the same time you can use the above parameters
```
[root@localhost tmp]# keadm debug check all
CPU total: 1 core, Allowed > 1 core
CPU usage rate: 0.10, Allowed rate < 0.9
Memory total: 1833.33 MB, Allowed > 256 MB
Memory Free total: 605.50 MB, Allowed > 128 MB
Memory usage rate: 0.17, Allowed rate < 0.9
Disk total: 50268.47 MB, Allowed > 1024 MB
Disk Free total: 39085.29 MB, Allowed > 512MB
Disk usage rate: 0.18, Allowed rate < 0.9
dns resolution success, domain: www.github.com ip: 192.30.255.112
ping 172.17.0.1 success
check cloudhubServer 10.211.55.6:10000 success
check edgecoreServer 127.0.0.1:10350 success
Maximum PIDs: 32768; Running processes: 129
docker is running

|-----------------|
|check all succeed|
|-----------------|
```
### keadm debug diagnose

diagnose node
```
[root@localhost tmp]# keadm debug diagnose node
edgecore is running
edge config is exists: /etc/kubeedge/config/edgecore.yaml
docker is running
dataSource is exists: /var/lib/kubeedge/edgecore.db
cloudcore websocket connection success
|---------------------|
|diagnose node succeed|
|---------------------|
```

diagnose pod success
```
[root@localhost tmp]# keadm debug diagnose pod nginx-ds-85jch -n default
edgecore is running
edge config is exists: /etc/kubeedge/config/edgecore.yaml
docker is running
dataSource is exists: /var/lib/kubeedge/edgecore.db
cloudcore websocket connection successDatabase /var/lib/kubeedge/edgecore.db is exist
Pod nginx-ds-85jch is exist
PodStatus nginx-ds-85jch is exist
pod nginx-ds-85jch phase is Running
containerConditions nginx-ds is ready
Pod nginx-ds-85jch is Ready
|--------------------|
|diagnose pod succeed|
|--------------------|
```

diagnose pod failed
```
[root@localhost tmp]# keadm debug diagnose pod nginx-ds-85jch -n kube-system
edgecore is running
edge config is exists: /etc/kubeedge/config/edgecore.yaml
docker is running
dataSource is exists: /var/lib/kubeedge/edgecore.db
cloudcore websocket connection successDatabase /var/lib/kubeedge/edgecore.db is exist
not find kube-system/pod/nginx-ds-85jch in datebase

|-------------------|
|diagnose pod failed|
|-------------------|
```

```
[root@localhost tmp]# keadm debug diagnose pod nginx-deployment-dbbffc676-wprs8
edgecore is running
edge config is exists: /etc/kubeedge/config/edgecore.yaml
docker is running
dataSource is exists: /var/lib/kubeedge/edgecore.db
cloudcore websocket connection successDatabase /var/lib/kubeedge/edgecore.db is exist
Pod nginx-deployment-dbbffc676-wprs8 is exist
PodStatus nginx-deployment-dbbffc676-wprs8 is exist
pod nginx-deployment-dbbffc676-wprs8 phase is Running
conditions is not true, type: Ready ,message: containers with unready status: [nginx] ,reason: ContainersNotReady
containerConditions nginx Terminated, message: oci runtime error: container_linux.go:235: starting container process caused "exec: \"/abc\": stat /abc: no such file or directory"
, reason: ContainerCannotRun, RestartCount: 104
Pod nginx-deployment-dbbffc676-wprs8 is not Ready

|-------------------|
|diagnose pod failed|
|-------------------|
```
