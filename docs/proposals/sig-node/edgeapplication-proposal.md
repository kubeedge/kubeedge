---
title: Deployment Spec Override via NodeLabels
status: implementable
authors:
    - "@EraKin575"
approvers:
  - TBD
  - TBD
creation-date: 2024-09-09
last-updated: 2024-09-17
---

# Solution Proposal: Enhancing EdgeApplication for Deployment Specification Overrides and Closed-Loop Flow Control

## 1. Introduction

**Deployment Specification Overrides:**  
Currently, EdgeApplication allows overrides based on node groups, but this functionality lacks the flexibility to deploy applications to specific nodes based on their labels. This proposal aims to extend this feature by allowing deployment specification overrides based on node labels/selectors.

## 2. Problem Statement

1. **Deployment Specification Overrides:**  
   EdgeApplication currently supports deployment overrides based on specs like replicas, images, etc., at the node group level. However, this method restricts deployment management's granularity and flexibility. There is a need for deployment overrides that can be applied to specific nodes based on node labels, allowing more fine-tuned control over deployments.

## 3. Proposed Solution

### Deployment Specification Overrides via NodeLabels

To enhance the deployment management capabilities of EdgeApplication, the solution proposes to:

#### 1. API Changes:

- Modify the EdgeApplication Custom Resource Definition (CRD) to introduce a new field for node label selectors called `targetNodeLabels`.
- This field will allow the application to deploy based on node labels and apply overrides specific to those nodes.

**Updated CRD:**

The `targetNodeLabels` field will match nodes based on labels and apply the defined deployment overrides. Below is a YAML representation of the updated `EdgeApplication` resource:

```yaml
apiVersion: apps.kubeedge.io/v1alpha1
kind: EdgeApplication
metadata:
  name: edge-app
  namespace: default
spec:
  replicas: 3
  image: my-app-image:latest
  # New field: targetNodeLabels
  targetNodeLabels:
    - labelSelector:
        - matchExpressions:
            - key: "region"
              operator: In
              values:
                - "us-west"
    overriders:
      containerImage:
        name: new-image:latest
      resources:
        limits:
          cpu: "500m"
          memory: "128Mi"
```

**Go Struct Changes:**

The `targetNodeLabels` struct will be added to the `EdgeApplication` schema as follows:

```go
type TargetNodeLabel struct {
    LabelSelector []metav1.LabelSelector `json:"labelSelector,omitempty"`
    Overriders Overriders `json:"overriders,omitempty"`
}
```

This struct includes:
- `LabelSelector`: A list of node labels to match.
- `Overriders`: A struct that allows overrides to specific deployment specifications like container images, resource limits, etc.

#### 2. Controller Logic:

The `EdgeApplicationController` will be updated to:
- Parse and handle the new `targetNodeLabels` field.
- Match nodes in the cluster based on the provided label selectors.
- Apply deployment overrides for those nodes using the specified `Overriders`.

**Steps:**
1. Extract the label selectors from `targetNodeLabels`.
2. Query the Kubernetes API to find nodes that match the label selectors.
3. Apply the appropriate deployment overrides to the matched nodes, as defined in the `Overriders` struct.

#### 3. Testing:

To ensure correct functionality, the following testing will be conducted:
- **Unit Tests:** Create unit tests to validate the behavior of the `targetNodeLabels` field. These tests will ensure that:
  - Nodes are correctly selected based on labels.
  - Deployment overrides are applied as expected for the matched nodes.
- **Integration Tests:** Test the integration of this feature within a Kubernetes/Edge cluster to verify that overrides are correctly applied in a real-world scenario.
- **EdgeApplicationController Tests:** Update existing tests to validate label-based overrides within the `EdgeApplicationController`.

## 4. Expected Outcomes

1. **Enhanced Flexibility:**  
   By introducing node label-based overrides, EdgeApplication will offer more granular control over the deployment of applications. This will allow for more complex and targeted deployment strategies based on the characteristics of individual nodes.

2. **Improved Efficiency:**  
   This solution will reduce the need for node group-based overrides and allow for more efficient use of resources by targeting only the nodes that meet specific criteria.

3. **Bug-Free Implementation:**  
   Existing tests will be updated, and new tests will be written to ensure that the feature works as expected, and does not introduce any regressions or bugs.
