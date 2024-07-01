---
title: Add pod restart and status query functions for the edge node
authors:
- "@luomengY"
  approvers:
  creation-date: 2024-04-15
  last-updated: 2024-04-15
  status: implementable
---

# Add pod restart and status query functions for the edge node 


## Motivation 

When the edge node is offline, it is not possible to query the status of the edge node pod and restart the pod of the edge node through kubectl in the cloud. This feature provides support for querying and restarting the pod status of the edge node.

### Goals

- By using the `keadm ctl get pod [flags]` command on edge nodes, the status of pod can be queried.
- By using the `keadm ctl restart pod [flags]` command on edge nodes, pod can be restarted.

## Background and challenges

- In edge computing, the network environment is usually poor, and edge nodes are offline most of the time. Now kubeedge does not support querying the status of pod when cloud edge is offline.
- When the edge node goes offline, the patch of the pod state of the edge node to the apiserver will fail. However, if the patch update is not done at the edge, the pod state in the edge node's metabase sqlite will not be updated. Even if kubedge has provided a metaserver and considered edge autonomy, the obtained pod state is still the state before going offline. Therefore, it is necessary to consider the patch update of the pod state of the edge node when it goes offline.
- When the edge node goes offline, users cannot restart the pod of the edge node. In many scenarios, users expect kubedge to support pod restart of the edge node. Here, we do not recommend deleting the pod at the edge node. Considering excessive permissions, edge pod restart only stops the containers in the pod, rather than killing the podsandbox.

## Design Details

### Keadm ctl get pod design.

1. Add the ctl get pod subcommand to keadm:

    ```
    "keadm ctl get pod" command get pods in edge node
    
    Usage:
      keadm ctl get pod [flags]
    
    Flags:
      -A, --all-namespaces     If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace
      -h, --help               help for pod
      -n, --namespace string   Specify a namespace (default "default")
      -o, --output string      Output format. One of: (json, yaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file, custom-columns, custom-columns-file, wide)
      -l, --selector string    Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
    ```
2. Get pod scheme design

   <img src="../images/proposals/keadm-get-pod.png">

  - When the `keadm ctl get pod [flags]` command is executed, a Restful request will be issued to MetaServer.
  - MetaServer determines whether edge and cloud are online on the network.
  - If the edge and cloud networks are connected, MetaServer will forward the restful request through a proxy to ApiServer, and then request a return result from ApiServer.
  - If the edge node goes offline, MetaServer will retrieve pod data from the edge metabase sqlite.
  - After obtaining the results, install the print format of kubectl and input it into the console.

3. example

    ```
    [root@centos-edgenode1 kubeedge]# keadm ctl get pod 
    NAME                                READY   STATUS             RESTARTS       AGE
    mysql-0                             0/1     CrashLoopBackOff   47 (55s ago)   140m
    nginx-deployment-7b79f6fd7f-wpm62   1/1     Running            0              139m
   
    [root@centos-edgenode1 kubeedge]# keadm ctl get pod -owide -A
    NAMESPACE                      NAME                                READY   STATUS             RESTARTS         AGE     IP               NODE               NOMINATED NODE   READINESS GATES
    default                        mysql-0                             0/1     CrashLoopBackOff   43 (2m55s ago)   138m    10.88.0.2        centos-edgenode1   <none>           <none>
    default                        nginx-deployment-7b79f6fd7f-wpm62   1/1     Running            0                137m    10.88.0.3        centos-edgenode1   <none>           <none>
    kube-system                    kube-proxy-lrhf2                    1/1     Running            0                6h27m   192.168.52.100   centos-edgenode1   <none>           <none>
    kubeedge                       edge-eclipse-mosquitto-4p96z        1/1     Running            0                6h42m   192.168.52.100   centos-edgenode1   <none>           <none>
    kubeedge                       edgemesh-agent-rtwr2                1/1     Running            0                5h43m   192.168.52.100   centos-edgenode1   <none>           <none>
    kubesphere-monitoring-system   node-exporter-pwcfm                 2/2     Running            0                128m    192.168.52.100   centos-edgenode1   <none>           <none>
   
   [root@centos-edgenode1 kubeedge]# keadm ctl get pod -n kubeedge -l k8s-app=kubeedge,kubeedge=edgemesh-agent -owide
   NAME                   READY   STATUS    RESTARTS   AGE     IP               NODE               NOMINATED NODE   READINESS GATES
   edgemesh-agent-rtwr2   1/1     Running   0          5h49m   192.168.52.100   centos-edgenode1   <none>           <none>
   ```

### Keadm ctl restart pod design.

1. Add the ctl restart pod subcommand to keadm:

    ```
    "keadm ctl restart pod" command delete pods in edge node
    
    Usage:
      keadm ctl restart pod [flags]
    
    Flags:
      -h, --help               help for pod
      -n, --namespace string   Specify a namespace (default "default")
    ```

2. Restart pod scheme design
   
   <img src="../images/proposals/keadm-restart-pod.png">
  
  - After executing `keadm ctl restart pod [flags]`, initiate a Restful API request to MetaServer to retrieve pod data.
  - Create an `internalapi.RuntimeService` through `remote.NewRemoteRuntimeService`.
  - Use the `io.kubernetes.pod.name` and `io.kubernetes.pod.namespace` tag selectors to filter the containers in the `remoteRuntimeService` interface that need to be restarted in the pod.
  - After obtaining the container list, using `remoteRuntimeService.StopContainer` to stop containers.

3. example

   ```
   [root@centos-edgenode1 kubeedge]# keadm ctl restart pod -n kubeedge edge-eclipse-mosquitto-j2db9
   4b9efa598c80ffc59705a1e49aeba0b5fec2db6513905c1cceb8aee7a2ae453d
   b63fa1d05f0163b5556663c33827e8df673d8c8c386da49c3b3ddf3ccd7efb84
   [root@centos-edgenode1 kubeedge]# keadm ctl get pod -n kubeedge edge-eclipse-mosquitto-j2db9 
   kubeedge                       edge-eclipse-mosquitto-j2db9              1/1     Running             2 (1m ago)   11d

   [root@centos-edgenode1 kubeedge]# keadm ctl get pod  -l k8s-app=kubeedge,kubeedge=edgemesh-agent -owide -A
   NAMESPACE   NAME                   READY   STATUS    RESTARTS   AGE     IP               NODE               NOMINATED NODE   READINESS GATES
   kubeedge    edgemesh-agent-rtwr2   1/1     Running   0          5h52m   192.168.52.100   centos-edgenode1   <none>           <none>
   [root@centos-edgenode1 kubeedge]# keadm ctl restart pod -n kubeedge    edgemesh-agent-rtwr2 
   689c25f7ca270b539dd4ae9288ba101ab5ca341140d09ee6497385446bac6f30
   [root@centos-edgenode1 kubeedge]# keadm ctl get pod  -l k8s-app=kubeedge,kubeedge=edgemesh-agent -owide -A
   NAMESPACE   NAME                   READY   STATUS    RESTARTS     AGE     IP               NODE               NOMINATED NODE   READINESS GATES
   kubeedge    edgemesh-agent-rtwr2   1/1     Running   1 (7s ago)   5h52m   192.168.52.100   centos-edgenode1   <none>           <none>

   [root@centos-edgenode1 kubeedge]# keadm ctl restart pod -n kubeedge    edgemesh-agent-rtwr2 
   4d6c7f11c98bc44902b87268b87a2a2091c3389eb1fa4f58325fb001ff655924
   [root@centos-edgenode1 kubeedge]# keadm ctl get pod  -l k8s-app=kubeedge,kubeedge=edgemesh-agent -owide -A
   NAMESPACE   NAME                   READY   STATUS    RESTARTS      AGE     IP               NODE               NOMINATED NODE   READINESS GATES
   kubeedge    edgemesh-agent-rtwr2   1/1     Running   2 (14s ago)   5h54m   192.168.52.100   centos-edgenode1   <none>           <none>
   ```

### Design of pod's status patch when edge nodes are offline.

1. When edge and cloud networks are connected, the process of updating the pod status of edge nodes is shown in the following figure:

   <img src="../images/proposals/edge-online-patchpodstatus.png">

  - When the state of the container in the pod changes, plege will perceive and send the pod with the changed state to pleCh.
  - SyncLoopIteration consumes pod data from plegCh and initiates pod status updates. In ManagePodLoop, it coordinates based on the pod status obtained from pleg and podStatusManager, calculates the latest pod status, and sends it to StatusManager through syncPod.
  - StatusManger then initiates a patch request for pod to ApiServer, and on the edge node, sends a patch pod request to Cloudcore through MetaClient. After receiving the patch message, Cloudcore's upstream sends the patch pod to ApiServer through Kubeclient.
  - The downstream in Cloudcore will list-watch the changes in pod in ApiServer. If a change in pod status is found in ApiServer, the downstream will send an update message to the edge,and save the latest pod status to sqlite, which is actually Reconciling the pod status in the edge.
  - In managerPodLoop, the latest pod status is calculated from pleg and statusManager and sent to synPod, and then to statusManager. The entire process is an infinite loop.

    **NOTE:** However, when the edge node goes offline, the patch pod to Apiserver will fail, and the process from 13 to 17 will not be executed. As a result, the status of the edge pod will not be updated, and the pod status obtained from the metaServer will still be the state before going offline. Therefore, the next step is to consider when the edge node goes offline, the pod status will be in the edge patch.
2. When the edge node goes offline, the status patch design of the edge node pod is shown in the following figure:
   
   <img src="../images/proposals/edge-offline-patchpodstatus.png">
  - When the edge node goes offline, in step 10, a patch pod message request will be directly sent to the MetaManager in the MetaClient.
  - When the status of the pod changes, steps 11-12 will be executed to ensure that the pod status in the metabase is updated in real-time when the edge node is offline.
  - In the offline scenario, interrupt step 13 and do not change the pod state in the managerPod. The purpose is to update the pod state that changed during the offline process to the api-server when the edge node returns to online offline.

### Will there be any impact of data inconsistency on cloud edges when edge nodes transition from offline to online?

In [the online process of edge nodes's patch pod](../images/proposals/edge-online-patchpodstatus.png),steps 13-17 will be executed, but in managePodLoop, the pod status will be obtained from pleg and statusManger, and then the latest pod status will be calculated. The pod status obtained from pleg is obtained from the container runtime, that is, the latest pod status of the current pod. Then, a patch pod request will be initiated through statusManager. Therefore, when the edge node is online, the pod status in ApiServer will be immediately updated to the latest pod status.

The code snippet can be found in the podWorkerLoop function in the kubelet module, as shown below:
```
status, err = p.podCache.GetNewerThan(update.Options.Pod.UID, lastSyncTime)
```

The state of generating the final pod can be found in the SyncPod function in kubelet.go, as shown below:
```
// Generate final API pod status with pod and status manager status
apiPodStatus := kl.generateAPIPodStatus(pod, podStatus, false)
```