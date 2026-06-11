# keadm ctl and kubectl

This document explains the difference between `keadm ctl` and `kubectl`, and
helps users choose the right command-line tool for common KubeEdge operations.

## Overview

`kubectl` is the standard Kubernetes command-line tool. It communicates with
the Kubernetes API server and shows resources from the Kubernetes control-plane
perspective.

`keadm ctl` is a KubeEdge command group for edge-side operations. It is designed
to help users inspect or operate resources from the edge data-plane perspective,
especially in scenarios where edge autonomy or edge-side troubleshooting is
needed.

`keadm ctl` does not replace `kubectl`. In most cases, users should still use
`kubectl` to manage Kubernetes resources from the cloud side. Use `keadm ctl`
when the operation is related to KubeEdge-specific edge-side behavior.

## Comparison

| Tool | Scope | Typical use cases |
| --- | --- | --- |
| `kubectl` | Kubernetes control plane | Manage Kubernetes resources such as Nodes, Pods, Deployments, Services, ConfigMaps, and Secrets through the API server. |
| `keadm ctl` | KubeEdge edge data plane | Inspect or operate edge-side resources and perform KubeEdge-specific troubleshooting operations. |

## When to use kubectl

Use `kubectl` when you need to manage or inspect resources through the Kubernetes
API server.

Common examples:

    kubectl get nodes
    kubectl get pods -A -o wide
    kubectl describe node <edge-node-name>
    kubectl describe pod <pod-name> -n <namespace>
    kubectl logs <pod-name> -n <namespace>

Typical scenarios:

- Check whether an edge node is registered in the Kubernetes cluster.
- Check whether a workload is scheduled to an edge node.
- Inspect Kubernetes events for a Pod, Node, Deployment, or Service.
- Manage Kubernetes resources from the cloud side.

## When to use keadm ctl

Use `keadm ctl` when you need to inspect or operate edge-side resources with
KubeEdge-specific commands.

Common examples:

    keadm ctl get pod -n <namespace>
    keadm ctl get pod -A -o wide
    keadm ctl describe pod <pod-name> -n <namespace>
    keadm ctl logs <pod-name> -n <namespace>
    keadm ctl exec <pod-name> -n <namespace> -- <command>
    keadm ctl restart pod <pod-name> -n <namespace>

Typical scenarios:

- Check edge-side Pod status from the KubeEdge data-plane perspective.
- Troubleshoot workload behavior on an edge node.
- View logs or execute commands for edge-side workloads when supported.
- Restart an edge-side Pod through KubeEdge-specific operations.

## Common workflow

For most troubleshooting cases, start with `kubectl` to check the Kubernetes
control-plane view, then use `keadm ctl` if more edge-side information is needed.

Example workflow:

    kubectl get nodes
    kubectl get pods -A -o wide
    kubectl describe pod <pod-name> -n <namespace>

    keadm ctl get pod -n <namespace>
    keadm ctl logs <pod-name> -n <namespace>

If `kubectl` shows that the edge node or Pod is not in the expected state, first
check Kubernetes scheduling, events, and resource definitions. If the Kubernetes
view looks correct but the workload still behaves unexpectedly on the edge side,
use `keadm ctl` to continue edge-side diagnosis.

## Notes and limitations

- `keadm ctl` is not a general replacement for `kubectl`.
- `kubectl` depends on the Kubernetes API server view.
- `keadm ctl` focuses on KubeEdge-specific edge-side operations.
- Some `keadm ctl` commands may depend on KubeEdge components and edge-side
  connectivity status.
- For standard Kubernetes resource management, prefer `kubectl`.
