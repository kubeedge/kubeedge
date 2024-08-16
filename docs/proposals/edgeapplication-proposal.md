# Solution Proposal: Enhancing EdgeApplication for Deployment Specification Overrides and Closed-Loop Flow Control

## 1. Introduction

**Deployment Specification Overrides:**  
Currently, EdgeApplication allows overrides based on node groups. This proposal aims to extend this functionality to node labels/selectors.

**Closed-Loop Flow Control:**  
There is a need to ensure that traffic within a node group remains contained and does not spill over to other groups. This proposal outlines changes to achieve closed-loop flow control while decoupling the scope of application batch management.

## 2. Problem Statement

1. **Deployment Specification Overrides:**  
   EdgeApplication currently supports deployment based on deployment specs like replicas, images, etc. This limitation restricts the flexibility in managing deployments based on granular node characteristics.

2. **Closed-Loop Flow Control:**  
   Deployment within a node group shares a service allowing traffic to cross node group boundaries. This problem is critical when traffic isolation is crucial.

## 3. Proposed Solution

### a. Deployment Specification Overrides

**1. API Changes:**

- Modify the EdgeApplication Custom Resource Definition (CRD) to include a new field for node label selectors.
- Define a new field `nodeLabelSelectors` in the EdgeApplication CRD schema.

    ```yaml
    spec:
      deploymentOverrides:
        nodeLabelSelectors:
          - key: "example.com/label"
            operator: "In"
            values:
              - "value1"
              - "value2"
    ```

**2. Controller Logic:**

- Update the EdgeApplicationController to handle the new `nodeLabelSelectors` field.
- Implement logic to match nodes based on labels and apply deployment overrides accordingly.

**3. Testing:**

- Develop unit tests to ensure that label-based overrides are correctly applied.

### b. Closed-Loop Flow Control

**Objective:**

- Implement a mechanism to ensure that service traffic is restricted within its node group.

**1. Service Endpoint Filtering:**

- Update the cloudcore service to filter EndpointSlice objects based on node group membership.
- Implement logic to restrict service endpoints so that they are only accessible within the same node group.

    ```go
    func filterEndpointsByNodeGroup(endpoints []EndpointSlice, nodeGroup string) []EndpointSlice {
        var filteredEndpoints []EndpointSlice
        for _, ep := range endpoints {
            if ep.NodeGroup == nodeGroup {
                filteredEndpoints = append(filteredEndpoints, ep)
            }
        }
        return filteredEndpoints
    }
    ```

**2. Network Policies:**

- Introduce network policies or update them to restrict traffic within node groups.

**3. Testing:**

- Conduct thorough testing to verify that traffic is properly contained within node groups.
- Validate that the closed-loop control mechanism does not interfere with legitimate traffic within the group.

## 4. Expected Outcomes

- **Upgraded Flexibility:**  
  EdgeApplication will support deployment specification overrides based on node labels, providing more granular control over deployment management.

- **Improved Traffic Control:**  
  Closed-loop flow control will be implemented, ensuring that service traffic remains within node groups and does not affect other groups.

- **Bug-Free Implementation:**  
  Comprehensive testing will ensure that both the new feature and closed-loop flow control function as expected without introducing bugs.
