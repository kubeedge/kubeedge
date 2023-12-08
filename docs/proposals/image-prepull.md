---
title: Images PrePull
authors:
- "@Shelley-BaoYue"
  approvers:
  creation-date: 2023-12-08
  last-updated: 2023-12-08
  status: implementable
---

# KubeEdge supports images prepull at edge nodes

## Motivation 

In some scenarios with unstable edge networks or limited bandwidth, large-scale batch application updates, or mobile edge environments, we can first complete the steps of image prepull when the network is stable, thereby improving the efficiency of application creation or updates.

### Goals 

- Provide APIs for create image prepull tasks.
- Support image prepull on multi edge nodes or nodegroup.
- Support prepull image list on edge nodes.

### Non-goals

- Deal with the problem that images on edge nodes may be recycled

## Design details 

### Architecture

![image-prepull architecture](../images/proposals/image-prepull.png)

### ImagePrePullController

`ImagePrePullController` is responsible for dealing with ImagePrePullJob, watches the resource with list-watch and sends image prepull message to edgenodes. 

- Use List-Watch mechanism to monitor ImagePrePullJob CRD resources, after receiving events from K8s APIServer, then store it in local cache using map.
- Use K8s informer to get node list according to NodeNames or LabelSelector specified in CR, and filter out nodes that don't meet image prepull requirements(1. not edge node, without label "node-role.kubernetes.io/edge": "" 2. edge node is in  NotReady state 3. remove duplicated nodes)
- Deal with the status reported from edge nodes and update the ImagePrePullJob.

### Task Manager
Use the task manager defined in PR [#5274](https://github.com/kubeedge/kubeedge/pull/5274)

### EdgeCore (EdgeHub)

- Check the received tasks.
- The registered functions of various tasks, such as checkDisk, imagePrepull, and so on, can be called according to the task category.

### Image PrePull API


```yaml
// ImagePrePullJob is the Schema for the imagePrePullJob API
type ImagePrePullJob struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  // Spec represents the specification of the desired behavior of ImagePrePullJob.
  // +required
  Spec ImagePrePullSpec `json:"spec"`

  // Status represents the status of ImagePrePullJob.
  // +optional
  Status ImagePrePullJobStatus `json:"status,omitempty"`
}
  
// ImagePrePullSpec represents the specification of the desired behavior of ImagePrePullJob.
type ImagePrePullSpec struct {
  // ImagePrePullTemplate represents original templates of imagePrePull
  ImagePrePullTemplate ImagePrePullTemplate `json:"imagePrePullTemplate,omitempty"`
  
  // ImagePrePullScope represents the override rules that would apply on imagePrePull job.
  // Will support in next version, more details in [Plan](#Plan) 
  //ImagePrePullScope []ImagePrePullOverrider `json:"imagePrepullScope,omitempty"`
}

  // ImagePrePullTemplate represents original templates of imagePrepull
type ImagePrePullTemplate struct {
  // Images is the image list to be prepull
  Images []string `json:"images`
  
  // NodeNames is a request to select some specific nodes. If it is non-empty,
  // the upgrade job simply select these edge nodes to do upgrade operation.
  // Please note that sets of NodeNames and LabelSelector are ORed.
  // Users must set one and can only set one.
  // +optional
  NodeNames []string `json:"nodeNames,omitempty"`
  // LabelSelector is a filter to select member clusters by labels.
  // It must match a node's labels for the NodeUpgradeJob to be operated on that node.
  // Please note that sets of NodeNames and LabelSelector are ORed.
  // Users must set one and can only set one.
  // +optional
  LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

  // CheckItems specifies the items need to be checked before the task is executed.
  // The default CheckItems value is disk.
  // +optional
  CheckItems []string `json:"checkItems,omitempty"`

  // FailureTolerate specifies the task tolerance failure ratio.
  // The default FailureTolerate value is 0.1.
  // +optional
  FailureTolerate string `json:"failureTolerate,omitempty"`

  // Concurrency specifies the maximum number of edge nodes that can pull images at the same time.
  // The default Concurrency value is 1.
  // +optional
  Concurrency int32 `json:"concurrency,omitempty"`
  
  // ImageSecret specifies the secret for image pull if private registry used. 
  // Use {namespace}/{secretName} in format. 
  // +optional
  ImageSecret string `json:"imageSecrets,omitempty"`

  // TimeoutSeconds limits the duration of the image prepull job on each edgenode.
  // Default to 300.
  // If set to 0, we'll use the default value 300.
  // +optional
  TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`
  
  // RetryTimes specifies the retry times if image pull failed on each edgenode.
  // Default to 0
  // +optional
  RetryTimes int32 `json:"retryTimes,omitempty"`
}

// ImagePrePullJobStatus stores the status of ImagePrePullJob.
// contains images prepull status on multiple edge nodes.
type ImagePrePullJobStatus struct {
  // State represents for the state phase of the ImagePrePullJob.
  // There are five possible state values: "", checking, pulling, successful, failed.
  State api.State `json:"state,omitempty"`

  // Event represents for the event of the ImagePrePullJob.
  // There are four possible event values: Init, Check, Pull, TimeOut.
  Event string `json:"event,omitempty"`

  // Action represents for the action of the ImagePrePullJob.
  // There are two possible action values: Success, Failure.
  Action api.Action `json:"action,omitempty"`

  // Reason represents for the reason of the ImagePrePullJob.
  Reason string `json:"reason,omitempty"`

  // Time represents for the running time of the ImagePrePullJob.
  Time string `json:"time,omitempty"`

  // Status contains image prepull status for each edge node.
  Status []ImagePrePullStatus `json:"status,omitempty"`
}

// ImagePrePullStatus stores image prepull status for each edge node.
type ImagePrepullStatus struct {
  // TaskStatus represents the status for each node
  *TaskStatus `json:"nodeStatus,omitempty"`

  // ImageStatus represents the prepull status for each image
  ImageStatus []ImageStatus `json:"imageStatus,omitempty"`
}

// TaskStatus stores the status of Upgrade for each edge node.
// +kubebuilder:validation:Type=object
type TaskStatus struct {
  // NodeName is the name of edge node.
  NodeName string `json:"nodeName,omitempty"`
  
  // State represents for the upgrade state phase of the edge node.
  // There are several possible state values: "", Upgrading, BackingUp, RollingBack and Checking.
  State api.State `json:"state,omitempty"`
  
  // Event represents for the event of the ImagePrePullJob.
  // There are three possible event values: Init, Check, Pull.
  Event string `json:"event,omitempty"`
  
  // Action represents for the action of the ImagePrePullJob.
  // There are three possible action values: Success, Failure, TimeOut.
  Action api.Action `json:"action,omitempty"`
  
  // Reason represents for the reason of the ImagePrePullJob.
  Reason string `json:"reason,omitempty"`
  
  // Time represents for the running time of the ImagePrePullJob.
  Time string `json:"time,omitempty"`
}

// ImageStatus represents the image pull status on the edge node
type ImageStatus struct {
  // Image is the name of the image
  Image string `json:"image,omitempty"`

  // State represents for the state phase of this image pull on the edge node.
  // There are two possible state values: successful and failed.
  State api.State `json:"state,omitempty"`
  
  // Reason represents the fail reason if image pull failed
  // +optional
  Reason string `json:"reason,omitempty"`
}

```

#### ImagePrepullJob Sample 

```yaml
apiVersion: operations.kubeedge.io/v1alpha1
kind: ImagePrePullJob
metadata:
  name: ImagePrepullExample
  labels:
    description: ImagePrepullLabel
spec:
  imagePrePullTemplate:  
    images: 
      - nginx:latest
      - example:v1.0.0
    nodes: 
      - node1
      - node2
    #labelSelector:
    #  matchLabels:
    #    "node-role.kubernetes.io/edge": ""
    #    node-role.kubernetes.io/agent: ""
    imageSecret: default/secret
    timeoutSeconds: 300
    retryTimes: 1
  imagePrePullOverriders: 
    - imagePrePullOverrider:
        nodeLabel: "labelexample"
        overriders:
          imageOverriders:
            - component: "Registry"
              operator: "replace"
              value: "hangzhou.registry.io"
          secretOverriders: 
            - component: "SecretName"
              operator: "replace"
              value: "hangzhou-secret"              

``` 

## Plan 

In alpha implement, each task execution only supports images from the same image repository currently. Similarly, only one secret is supported to be config. In next version, We will support overriding images and secrets as below.

```yaml
// ImagePrePullOverrider represents the override rules that would apply on imagePrePull job.
type ImagePrePullOverrider struct{
  // NodeLabel represents a group of nodes with same label
  NodeLabel string `json:nodeLabel,omitempty`
  
  // Overriders represents the override rules that would apply on resources.  
  Overriders Overriders `json:overriders,omitempty`
}

// Overriders represents the override rules that would apply on resources.
type Overriders struct {
  // ImageOverriders represents the rules dedicated to handling image overrides.
  // +optional
  ImageOverrider []ImageOverrider `json:"imageOverriders,omitempty"`
  
  // SecretOverrider represents the rules dedicated to handling secret overrides.
  // +optional
  SecretOverrider []SecretOverrider `json:"secretOverriders,omitempty"`
}

// ImageOverriders represents the rules dedicated to handling image overrides.  
type ImageOverrider struct {
  // Component is part of image name.
  // Basically we presume an image can be made of '[registry/]repository[:tag]'.
  // The registry could be:
  // - k8s.gcr.io
  // - fictional.registry.example:10443
  // The repository could be:
  // - kube-apiserver
  // - fictional/nginx
  // The tag cloud be:
  // - latest
  // - v1.19.1
  // - @sha256:dbcc1c35ac38df41fd2f5e4130b32ffdb93ebae8b3dbe638c23575912276fc9c
  //
  // +kubebuilder:validation:Enum=Registry;Repository;Tag
  // +required
  Component ImageComponent `json:"component"`

  // Operator represents the operator which will apply on the image.
  // +kubebuilder:validation:Enum=add;remove;replace
  // +required
  Operator OverriderOperator `json:"operator"`

  // Value to be applied to image.
  // Must not be empty when operator is 'add' or 'replace'.
  // Defaults to empty and ignored when operator is 'remove'.
  // +optional
  Value string `json:"value,omitempty"`
}

// Overriders represents the override rules that would apply on resources. 
type SecretOverrider struct {
  // Component is part of pull secret.
  // Currently we only support namespace and secretName.
  Component SecretComponent `json:"component"`

  // Operator represents the operator which will apply on the image.
  // +kubebuilder:validation:Enum=add;remove;replace
  // +required
  Operator OverriderOperator `json:"operator"`

  // Value to be applied to image.
  // Must not be empty when operator is 'add' or 'replace'.
  // Defaults to empty and ignored when operator is 'remove'.
  // +optional
  Value string `json:"value,omitempty"`
}

// ImageComponent indicates the components for image.
type ImageComponent string

// SecretComponent indicates the components for secret.
type SecretComponent string

// OverriderOperator is the set of operators that can be used in an overrider.
type OverriderOperator string

// These are valid overrider operators.
const (
  OverriderOpAdd     OverriderOperator = "add"
  OverriderOpRemove  OverriderOperator = "remove"
  OverriderOpReplace OverriderOperator = "replace"
)
```