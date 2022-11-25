# Proposal for refactoring keadm cmds on the cloud

---
title: Reliable message delivery
authors:
  - "@fisherxu"
  - "@zc2638"
  - "@gy95"
  - "@zhu733756"
approvers:
  - "@fisherxu"
  - "@zc2638"
  - "@gy95"
creation-date: 2021-12-28
last-updated: 2022-01-14
status: Implememted
---

## Plan A: using operator and CRD

By using k8s CRDs to install cloud components, this plan can be flexibly installed according to different scenarios like istioctl.

There are two kinds of definition, helm style and profile style.

For helm style,  the CR can be described as follows:

```
apiVersion: helm.keadm.kubeedge.io/v1alpha1
kind: KeadmConfiguration
spec:
  cloudCore:
    replicaCount: 1
    hostNetWork: "true"
    image:
      repository: "kubeedge/cloudcore"
      tag: "v1.8.2"
      pullPolicy: "IfNotPresent"
      pullSecrets: []
    securityContext: 
      privileged: true
    labels:
      k8s-app: kubeedge
      kubeedge: cloudcore
    annotations: {}
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
          - key: node-role.kubernetes.io/edge
              operator: DoesNotExist
    tolerations: []
    nodeSelector: {}
    resources:
      limits:
        cpu: 200m
        memory: 1Gi
      requests:
        cpu: 100m
        memory: 512Mi
    modules:
      cloudHub:
        advertiseAddress:   # Causion!: Leave this entry to empty will cause CloudCore to exit abnormally once KubeEdge is enabled. 
          - ""              # At least a public IP Address or an IP which can be accessed by edge nodes must be provided!           
        nodeLimit: "1000"
        websocket:
          enable: "true"
        quic:
          enable: "false"
          maxIncomingStreams: "10000"
        https:
          enable: "true"
        cloudStream:
          enable: "true"
        dynamicController:
          enable: "false"
        router:
          enable: "false"
    service:
        enable: "true"
        type: "NodePort"
        cloudhubNodePort: "30000"
        cloudhubQuicNodePort: "30001"
        cloudhubHttpsNodePort: "30002"
        cloudstreamNodePort: "30003"
        tunnelNodePort: "30004"

  iptablesManager:
    enable: "true"
    mode: "internal"
    hostNetWork: true
    image:
      repository: "kubeedge/iptables-manager"
      tag: "v1.8.2"
      pullPolicy: "IfNotPresent"
      pullSecrets: []
    securityContext: 
      capabilities:
      add:
        - NET_ADMIN
        - NET_RAW
    labels:
      k8s-app: iptables-manager
      kubeedge: iptables-manager
    annotations: {}
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/edge
                  operator: DoesNotExist
    tolerations: []
    nodeSelector: {}
    resources:
      limits:
        cpu: 200m
        memory: 50Mi
      requests:
        cpu: 100m
        memory: 25Mi

  edgemesh:
    enable: false
    gateway:
      enable: false
      nodeName: "worker-2"
    server:
      nodeName: "worker-1"
      advertiseAddress: "10.10.102.242"  # if not given, will use cloudcore.advertiseAddress[0]
    edged:
      clusterDNS: 169.254.96.16"
      clusterDomain: "cluster.local"
```

For profile style, which aims to kubeedge quickstart, core section defines the Indispensable values or values that need to be overridden, the profiles section defines the optional profiles that need to enable. An example CR can be described as follows, very simple:

``` 
apiVersion: profile.keadm.kubeedge.io/v1alpha1
kind: KeadmOperator
spec:
  core:
    cloudCoreAdvertiseAddress: 
      - "10.10.102.242" 
    haModeKeepAlivedIpPorts:
      - "10.10.102.242:10000"
    edgemesh:
      server:
        nodeName: "worker-1"
        serverAdvertiseAddress: "10.10.102.242" 
      gateway:
        enable: false
        nodeName: "worker-2"
  profiles:
    # The following four fields represent four scenarios that can be combined or applied individually.
    version: "v1.9.0"     # will use the default recommended configuration to install.
    iptablesMgrMode: "external" # default is internal.
    enableCloudCoreHaMode: true
    enableEdgemesh: true
```

The workflow can be described as the following:

```flow
start=>start: keadm config --type=helm/profile [--minconfig/default] > keadm-config.yaml
edit=>operation: vim keadm-config.yaml
init=>operation: keadm init-beta -f keadm-config.yaml
end=>end: deployed

start->edit->init->end
```

> Every profile-style CR could convert to the helm-style CR. And the two CRDs are in the cluster level.

## Plan B: using helm-style cmd

Since the default charts have been compiled in the keadm binary,  this plan will directly use the istio or helm APIs to render the default Charts to the expected Charts. If everything is ok, the components will successfully apply to the cluster.

```flow
start=>start: keadm init-beta --set cloudCore.modules.cloudHub.advertiseAddress[0]=192.168.88.6 --profile iptablesMgrMode=external
keadm init-beta --set cloudCore.modules.cloudHub.advertiseAddress[0]=192.168.88.6 --profile version=1.9.0

render=>operation: accept the key-value pairs, apply the custom charts to the cluster
apply=>end: deployed

start->render->apply
```

> Refer to the above configuration in Plan A for the profile parameters

## Referenced command line arguments for Plan B

### Init-beta

> The former keadm init cmd will keep several versions, later, the init-beta cmd can be replaced by init.

This cmd is used for installing the core components, such as cloudcore, external iptableMgr, edgemesh, cloudcore hamode, etc. 


#### --profile

version=1.9.0
> the given version will seek the recommended parameters to install

iptablesMgrMode=external
> the given version will seek the recommended parameters to install

enableCloudCoreHaMode=true
> enable ha mode for cloudcore

enableEdgeMesh=true
> enable edgemesh component

#### --set

A list of set flags like helm flags.

#### --external-helm-root
External helm root path to install charts for kubeedge.

##### --files, -f, --manifests
Allow appending file paths of manifests to keadm, separated by commas

#### --namespace

> The namespace to install, default is kubeedge.

#### --dry-run

> print the generated k8s resources on the stdout, not actual execute. Always use in debug mode.

### config

Like kubeadm config images list/pull

```
keadm config images list/pull
```
### manifest

Also support the "--set" list above for custom modification.

#### generate

Generate the k8s resources.

```
keadm manifest generate <your original installation options> | kubectl delete -f -
```
<!-- 
##### --kustomize
will generate all the files to one file, by default will be separated. -->

##### --files, -f
Allow appending file paths of manifests to keadm, separated by commas

#### --charts
Allow appending file directories of charts to keadm, separated by commas

### reset-beta

> Uninstall the existing helm charts or manifests on the cloud.

#### --delete-namespace, -D

> This flag will forced-delete all resources and the namespace. 

#### --grace-period

> All resources will be gracefully removed after the interval.

### profile

```
keadm profile list
> iptablesMgrMode
> version
> enableCloudCoreHaMode
> enableEdgeMesh
```

## Sub tasks for plan B

- Support haproxy for helm
- keadm  init-beta
- keadm config images list/pull
- keadm reset-beta
- keadm manifest  > k8s resources


### References

[istio/helm](https://github.com/istio/istio/tree/master/operator/pkg/helm)

[istio/helmreconcile](https://github.com/istio/istio/blob/master/operator/pkg/helmreconciler/reconciler.go) 

[istioctl-completion](https://istio.io/latest/docs/reference/commands/istioctl/#istioctl-completion)

[istioctl-uninstall](https://istio.io/latest/docs/setup/install/istioctl/#uninstall-istio)