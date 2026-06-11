# Viewing edge-side resources with keadm ctl

This document explains how to view edge-side workload status with `keadm ctl`
and how to combine it with `kubectl` during KubeEdge troubleshooting.

## Background

In a KubeEdge cluster, `kubectl` shows resources from the Kubernetes API server
perspective. It is still the first tool to use when checking whether nodes,
pods, deployments, and other Kubernetes resources are created and scheduled
correctly.

`keadm ctl` provides KubeEdge-specific edge-side operations. It can help inspect
or operate edge-side workloads from the KubeEdge data-plane perspective,
especially when edge-side diagnosis is needed.

`keadm ctl` does not replace `kubectl`. In most cases, use both tools together.

## Start with the Kubernetes control-plane view

Use `kubectl` first to check whether the edge node and workloads are visible
from the Kubernetes control plane.

```bash
kubectl get nodes -o wide
kubectl get pods -A -o wide
kubectl describe node <edge-node-name>
kubectl describe pod <pod-name> -n <namespace>
```

These commands help answer questions such as:

- Is the edge node registered in the cluster?
- Is the edge node `Ready` or `NotReady`?
- Is the pod scheduled to the expected edge node?
- Are there Kubernetes events showing scheduling, image pulling, or runtime
  errors?

If the Kubernetes control-plane view already shows an obvious problem, such as
a pod pending or an edge node being `NotReady`, start troubleshooting from that
information first.

## Use keadm ctl for edge-side workload inspection

After checking the Kubernetes control-plane view, use `keadm ctl` when you need
more edge-side information.

Common examples:

```bash
keadm ctl get pod -n <namespace>
keadm ctl get pod -A -o wide
keadm ctl describe pod <pod-name> -n <namespace>
keadm ctl logs <pod-name> -n <namespace>
keadm ctl exec <pod-name> -n <namespace> -- <command>
```

Typical use cases include:

- Checking pod status from the edge-side data-plane perspective.
- Viewing edge-side pod details when Kubernetes control-plane information is
  not enough.
- Reading workload logs for edge-side diagnosis.
- Executing a command in an edge-side workload when supported.

## Example troubleshooting workflow

A common workflow is to compare the Kubernetes control-plane view with the
KubeEdge edge-side view.

```bash
kubectl get nodes -o wide
kubectl get pods -A -o wide

keadm ctl get pod -n <namespace>
keadm ctl describe pod <pod-name> -n <namespace>
keadm ctl logs <pod-name> -n <namespace>
```

If `kubectl` shows that the pod is running but the workload still behaves
unexpectedly on the edge node, use `keadm ctl` to continue edge-side diagnosis.

If `kubectl` shows that the edge node is `NotReady`, first check cloud-edge
connectivity, EdgeCore status, CloudCore status, and node events. In this case,
the Kubernetes control-plane view and the edge-side view may be different.

## MetaServer dependency

Some `keadm ctl` operations rely on KubeEdge edge-side components such as
MetaServer. Before using `keadm ctl` for edge-side inspection, make sure the
related KubeEdge edge-side components are running.

Useful checks on the edge node include:

```bash
systemctl status edgecore
journalctl -u edgecore -xe
```

If MetaServer or EdgeCore is not running correctly, `keadm ctl` may not be able
to return the expected edge-side information.

## Known limitations

`keadm ctl` is focused on KubeEdge-specific edge-side operations. It is not a
full replacement for Kubernetes observability or monitoring tools.

For example:

- Use `kubectl get nodes` and `kubectl describe node` to check Kubernetes node
  readiness and node conditions.
- Use `kubectl get pods -A -o wide` to check pod scheduling and pod status from
  the Kubernetes API server perspective.
- Use metrics and monitoring tools to check CPU, memory, and other resource
  usage.
- Use device-related CRDs, mapper logs, or device-specific tools when
  troubleshooting device status.

If an expected `keadm ctl` command does not exist or does not show the required
information, combine Kubernetes-native commands, KubeEdge component logs, and
monitoring tools for a complete diagnosis.

## Summary

Use `kubectl` to understand the Kubernetes control-plane state, and use
`keadm ctl` when edge-side workload inspection is needed.

A practical rule is:

- Start with `kubectl` for cluster-level and scheduling information.
- Continue with `keadm ctl` for edge-side workload details.
- Check EdgeCore, MetaServer, and CloudCore logs when the two views do not match.
