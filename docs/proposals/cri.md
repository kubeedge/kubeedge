---
title: CRI Design for edged
authors:
    - "@gpinaik"
approvers:
  - "@qizha"
  - "@CindyXing"
  - "@Baoqiang-Zhang"
  - "@m1093782566"
creation-date: 2019-04-19
last-updated: 2019-04-21
status: implementable
---

# CRI Support in edged

* [CRI Support in edged](#CRI-Support-in-edged)
  * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non\-goals](#non-goals)
  * [Proposal](#proposal)
    * [Use Cases](#use-cases)
  * [High Level Design](#high-level-design)
    * [Edged with CRI support](#edged-with-cri-support)
  * [Low Level Design](#low-level-design)
    * [Configuration parameters](#configuration-parameters)
    * [Data structure modifications](#data-structure-modifications)
    * [Edged object creation modifications](#edged-object-creation-modifications)
    * [Runtime dependent module modifications](#runtime-dependent-module-modifications)
    * [Runtime dependent functional modifications](#runtime-dependent-functional-modifications)
  * [Open Questions](#open-questions)


## Motivation
This proposal addresses the Container Runtime Interface support in edged to enable the following
1. Support light weight container runtimes on resource constrained edge node which are unable to run the existing docker runtime
2. Support multiple container runtimes like docker, containerd, cri-o etc on the edge node.

### Goals
CRI support in edged must:
* support multiple runtimes like docker, contianerd, cri-o etc.
* Support for corresponding CNI with pause container and IP will be considered later

### Non-goals

* Automatic detection of container runtimes and its selection.

## Proposal

Currently Kubernetes kubelet CRI supports container runtimes like containerd, cri-o etc and support for docker runtime is
provided using dockershim as well. However going forward even docker runtime will be supported through only CRI. However
currently kubeedge edged supports only docker runtime using the legacy dockertools. Hence we propose to support multiple
container runtime in kubeedge edged as follows
1. Include CRI support as in kubernetes kubelet to support contianerd, cri-o etc
2. Continue with docker runtime support using legacy dockertools until CRI support for the same is available i.e. support
for docker runtime using dockershim is not considered in edged


### Use Cases

* Customer can run light weight container runtime on resource constrained edge node that cannot run the existing docker runtime
* Customer has the option to choose from multiple container runtimes on his edge platform


## High Level Design

### Edged with CRI support

## Low Level Design

### Configuration parameters

The following configuration parameters need to be added

No | Parameter | Type | Values | Description
---|---|---|---|---
1 | runtimeType | string | docker/remote | Runtime name
2   | remoteRuntimeEndpoint | string | /var/run/*.sock | Endpoint of remote runtime service
3   | remoteImageEndpoint | string | same as remoteRuntimeEndpoint if not specified | Endpoint of remote image service
4   | RuntimeRequestTimeout | Duration | time value | timeout for all runtime request
5   | PodSandboxImage | string | image name | Image used for pause container in sandbox

```go
type Config struct {
       ....
       runtimeType           string
       remoteRuntimeEndpoint string
       remoteImageEndpoint   string
       RuntimeRequestTimeout metav1.Duration
       PodSandboxImage       string
       ....
 }
 ```
### Data structure modifications

The edged data structure needs to include the remote runtime and runtime name. Also need to add os interface, pod cache and container life cycle manager parameters required for initializing and executing remote runtime.

```go
//Define edged
type edged struct {
  ....
  containerRuntimeName string
  // Container runtime
  containerRuntime kubecontainer.Runtime
  podCache           kubecontainer.Cache
  os                 kubecontainer.OSInterface
  clcm           clcm.ContainerLifecycleManager
  ....
}

```

###  Edged object creation modifications

The existing newEdged() function needs to modified include creating CRI runtime object based on the runtime type including
creations of objects for runtime and image services. However the existing edged does not provide the support for all the
parameters required to create the CRI runtime object and default parameters need to be considered for the same like Image GC manager, Container GC manager, Volume manager and container lifecycle manager (clcm)

```go

//newEdged creates new edged object and initialises it
func newEdged() (*edged, error) {
       conf := getConfig()
       ......

       switch based on runtimeType {
            case DockerContainerRuntime:
	        Create runtime based on docker tools
                Set containerRuntimeName to DockerContainerRuntime
		Initialize Container GC, Image GC and Volume Plugin Manager accordingly

            case RemoteContainerRuntime:
                Set remoteImageEndpoint same as remoteRuntimeEndpoint if not explicitly specified
                Initialize the following required for initializing remote runtime
			containerRefManager
			httpClient
			runtimeService
			imageService
			clcm
			machineInfo with only memory capacity
               Initialize Generic Runtime Manager
               Set ContainerRuntimeName to RemoteContainerRuntime
               Set runtimeService
               Initialize Container GC, Image GC and Volume Plugin Manager accordingly

            default:
	       unsupported CRI runtime
       }
       .....
       .....
}

//Function to get CRI runtime and image service
func getRuntimeAndImageServices(remoteRuntimeEndpoint string, remoteImageEndpoint string, runtimeRequestTimeout metav1.Duration) (internalapi.RuntimeService, internalapi.ImageManagerService, error) {
       Initialize Remote Runtime Service
       Initialize Remote Image Service
 }
```
The function to read the configuration needs to be modified to include the parameters required for remote runtime. By default
docker runtime shall be used and also default values for the parameters need to be provided if the parameters are not provided
in the configuration file.

```
// Get Config
func getConfig() *Config {
	....
	Check for rumtime type and if not provided set to docker
        if runtimeType is remote
		Get edged memory capacity and set to default of 2G if not provided
		Get remote runtime endpoint and set to /var/run/containerd/container.sock if not provided
		Get remote image endpoint and set same as remote runtime endpoint of not provided
                Get runtime Request Timeout and set to default of 2 min if not provided
                Get PodSandboxImage and set to kubeedge/pause
 	....
}

```
### Runtime dependent module modifications
The following modules which are dependent on the runtime needs to be handled during edged start based on docker runtime or
remote CRI runtime.
1. Volume Manager
2. PLEG

```go
func (e *edged) Start(c *context.Context) {
  ....
  switch based on runtime type {
    case DockerContainerRuntime:
      Initialize volume manager based on dockertools
      Initialize PLEG based on dockertools

    case RemoteContainerRuntime:
      Initialize volume manager based on remote runtime
      Initialize PLEG based on remote runtime
  }
  ....

}
```

### Runtime dependent functional modifications

The following functionalities which are based on the docker runtime need to be modified to handle the CRI runtimes as well

```go
func (e *edged) initializeModules() error {
  ....
  switch based on runtime type {
    case DockerContainerRuntime:
	 Start with docker runtime

    case RemoteContainerRuntime:
         Start with remote runtime
  ....
}
func (e *edged) consumePodAddition(namespacedName *types.NamespacedName) error {
  ....
  switch based on runtime type {
    case DockerContainerRuntime:
	 Ensure iamge exists for docker runtime
	 Start pod with docker runtime

    case RemoteContainerRuntime:
         Get current status from pod cache
	 Sync pod with remote runtime
  ....
}

func (e *edged) consumePodDeletion(namespacedName *types.NamespacedName) error {
  ....
  switch based on runtime type {
    case DockerContainerRuntime:
	 TerminatePod with docker runtime

    case RemoteContainerRuntime:
         KillPod with remote runtime
   }

  ....
}


func (e *edged) addPod(obj interface{}) {
  ....
  UpdatePluginResources for only docker runtime
  ....
}

func (e *edged) HandlePodCleanups() error {
  ....
  switch switch based on runtime type {
    case DockerContainerRuntime:
       GetPods for docker runtime

    case RemoteContainerRuntime:
       GetPods for remote runtime
  }
  ....
  ....
}

```

Existing PLEG wrapper needs to be modified to handle remote runtime. Addition new function needs to be provided to initialize
PLEG and update pod status

```
func NewGenericLifecycleRemote(runtime kubecontainer.Runtime, probeManager prober.Manager, channelCapacity int,
        relistPeriod time.Duration, podManager podmanager.Manager, statusManager status.Manager, podCache kubecontainer.Cache, clock clock.Clock, iface string) pleg.PodLifecycleEventGenerator {
	Intialize with additional parameters
}


func (gl *GenericLifecycle) updatePodStatus(pod *v1.Pod) error {
  ....
  Get pod status based on remote/docker runtime
  Convert to API pod status for remote runtime
  Set pod status phase for remote runtime

  ....
}
```

