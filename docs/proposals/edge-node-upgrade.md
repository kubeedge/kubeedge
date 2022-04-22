---
title: Edge Node Upgrade
authors:
- "@gy95"
approvers:
creation-date: 2022-04-26
last-updated: 2022-04-26
status: implementable
---

# Edge Node Upgrade

## Motivation

Edge node upgrade management is a key feature required for upgrading edge nodes from remote cloud side in edge computing.
This proposal addresses how can we upgrade edge nodes from the cloud, and synchronize the edge node upgrade result status between edge nodes and cloud.

### Goals

Edge node upgrade management must:
* provide APIs for upgrading edge nodes from the cloud.
* synchronize the edge node upgrade result between cloud and edge nodes.

## Proposal
We propose using Kubernetes [Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) 
to describe Upgrade metadata/status and a controller to synchronize Upgrade between edge and cloud.

### Use Cases

* Describe Upgrade properties.
    * Users can describe Upgrade properties and access mechanisms to interact with / control the Upgrade.
* Perform CRUD operations on Upgrade from cloud.
    * Users can create, update and delete Upgrade metadata from the cloud via the CRD APIs exposed by the Kubernetes API server.
* Report Upgrade properties values.
    * Edge nodes can report upgrade result status.
  

## Upgrade Controller Design
The Upgrade controller starts two separate goroutines called  `upstream` controller and `downstream` controller.
These are not separate controllers as such but named here for clarity.
The job of the downstream controller is to synchronize the Upgrade updates from the cloud to the edge node.
The job of the upstream controller is the reverse.


## Synchronizing Upgrade

The below illustrations describe the flow of events that would occur when Upgrade property values are updated from the cloud/edge.

- Users create/modify the Upgrade CR directly using `kubectl`, or indirectly using `keadm` tool and `keadm` will call k8s API to create Upgrade CR.

- Upgrade Controller downstream receive events from K8s APIServer, and then store it in local cache,
  and send beehive message to edge.

- EdgeHub receive upgrade message, and upgrade sub module install specific version `keadm` tool, and run `keadm` related command
  to do upgrade operation.

- `keadm` backup data, and upgrade the `EdgeCore` to the specific version, and rollback upgrade if failed. And finally report
  upgrade result status to CloudHub HTTP service.

- CloudHub HTTP service receive http report upgrade status request, distribute it to Upgrade Controller Upstream.

- Upgrade Controller Upstream call K8s API to update Upgrade CR. And users can get upgrade result status using `kubectl get` command.


<img src="../images/edge-node-upgrade/upgrade.png">


## CRD Design Details

### CRD API Group and Version
The `Upgrade` CRD will be cluster-scoped.
The tables below summarize the group, kind and API version details for the CRDs.

* Upgrade

| Field                 | Description             |
|-----------------------|-------------------------|
|Group                  | upgrade.kubeedge.io     |
|APIVersion             | v1alpha2                |
|Kind                   | Upgrade                 |


### Upgrade CRD
A `Upgrade` describes the Upgrade properties exposed by the Upgrade.
A Upgrade is like a reusable template using which edge nodes can be upgraded to the specific version, easily operated from cloud side.

### Upgrade Type Definition
```go
/*
Copyright 2022 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Upgrade stores the state of the Upgrade, nonNamespaced object that was used to upgrade edge node from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type Upgrade struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  // Specification of the desired behavior of Upgrade.
  // +optional
  Spec UpgradeSpec `json:"spec,omitempty"`
  // Most recently observed status of the Upgrade.
  // +optional
  Status []UpgradeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UpgradeList is a list of Upgrade.
type UpgradeList struct {
  // Standard type metadata.
  metav1.TypeMeta `json:",inline"`

  // Standard list metadata.
  metav1.ListMeta `json:"metadata,omitempty"`

  // List of Upgrades.
  Items []Upgrade `json:"items"`
}

// UpgradeSpec is a description of Upgrade.
type UpgradeSpec struct {
  // +Required: Version is the EdgeCore version to upgrade.
  Version string `json:"version,omitempty"`
  // UpgradeInstaller is the cmd to install the latest upgrader
  // If it is empty, will install the required version keadm from docker images.
  // +optional
  UpgradeInstaller string `json:"upgradeInstaller,omitempty"`
  // UpgradeCmd is the cmd to upgrade the edge node to the required version.
  // If it is empty, will use keadm to do upgrade operation.
  // +optional
  UpgradeCmd string `json:"upgradeCmd,omitempty"`
  // NodeNames is a request to select some specific nodes. If it is non-empty,
  // the Upgrade simply select these edge nodes to do upgrade operation
  // +optional
  NodeNames []string `json:"nodeNames,omitempty"`
  // LabelSelector is a selector which must be true for the Upgrade to fit on a node.
  // Selector which must match a node's labels for the Selector to be operated on that node.
  // +optional
  LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// UpgradeOperationStatus describe the result status of upgrade operation on edge nodes.
// +kubebuilder:validation:Enum=upgrading;upgrade_success;upgrade_failed_rollback_success;upgrade_failed_rollback_failed
type UpgradeOperationStatus string

// upgrade operation status
const (
  UpgradeUpgrading             UpgradeOperationStatus = "upgrading"
  UpgradeSuccess               UpgradeOperationStatus = "upgrade_success"
  UpgradeFailedRollbackSuccess UpgradeOperationStatus = "upgrade_failed_rollback_success"
  UpgradeFailedRollbackFailed  UpgradeOperationStatus = "upgrade_failed_rollback_failed"
)

const MaxStatusHistory = 20

// UpgradeStatus stores the status of Upgrade.
// contains multi nodes status
// key is edge node name, value is []Status
// +kubebuilder:validation:Type=object
type UpgradeStatus struct {
  NodeName string    `json:"nodeName,omitempty"`
  History  []History `json:"history,omitempty"`
}

type History struct {
  // FromVersion is the version Upgrade from
  FromVersion string `json:"fromVersion,omitempty"`
  // ToVersion is the version Upgrade to
  ToVersion string `json:"toVersion,omitempty"`
  // Status is the status of Upgrade..
  Status UpgradeOperationStatus `json:"operationStatus,omitempty"`
  // Reason is the error reason of Upgrade failure.
  Reason string `json:"reason,omitempty"`
}

```

### Upgrade sample
```yaml
apiVersion: upgrade.kubeedge.io/v1alpha2
kind: Upgrade
metadata:
  name: upgrade-example
  labels:
    description: upgrade-label
spec:
  version: "v1.10.0"
  upgradeInstaller: ""
  upgradeCmd: ""
  labelSelector:
    matchLabels:
      "node-role.kubernetes.io/edge": ""
      node-role.kubernetes.io/agent: ""


```
Shown above is an example Upgrade for upgrading edge node. It has four properties:
- `version`: this property describes which version that we want to upgrade to.
- `nodeNames`: this property describes a list of edge node, `Upgrade Controller` will select those nodes, and then upgrade those edge nodes.
- `labelSelector`: this property defines label selectors, `Upgrade Controller` will select nodes that match the label, 
  and the upgrade those edge nodes.
- `upgradeInstaller`: this property describes how to install the required installer, users can use this field to customize installer install.
- `upgradeCmd`: this property describes how to upgrade edge node, users can define this field value to customize upgrade command.

### Validation
[Open API v3 Schema based validation](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#validation) can be used to guard against bad requests.
Invalid values for fields ( example string value for a boolean field etc) can be validated using this.
In some cases , we also need custom validations (e.g create an Upgrade instance which not specify nodes ) .
[Validation admission web hooks](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook) can be used to implement such custom validation rules.

Here is a list of validations we need to support :

#### Upgrade Validations
- Don't allow Upgrade creation if any `Required` fields are missing ( like version etc.)
- nodeNames and labelSelector cannot be both empty. That means that we must specify at least one valid node.
- Don't allow update CR from cloud if no feedback from edge
- Don't allow update nodeNames or LabelSelector once Upgrade CR is created.
- Only allow update KubeEdge Version that we want to upgrade only when the upgrade receive feedback from edge.
- If upgrade failed, users need to debug why it failed by themselves through K8s Upgrade CR status fields. And maybe users
  also need to upgrade manually.
- Upgrade status will only contain 20 history status per node.






