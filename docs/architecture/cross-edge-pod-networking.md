# Cross-Edge Pod Networking Troubleshooting

When pods start successfully but cross-node communication fails, this guide
helps determine whether to investigate the CNI plugin or EdgeMesh first.

## Quick Decision Tree

```
                     ┌─────────────────────────┐
                     │  Pod created but cross-  │
                     │  node traffic fails?     │
                     └───────────┬─────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │ Does the pod have an IP? │
                    └────────┬───────────┬────┘
                             │ No        │ Yes
                             ▼           │
                     ┌───────────┐       │
                     │ Check CNI │       │
                     └───────────┘       │
                                         │
                     ┌───────────────────▼───────────────────┐
                     │ Can pods on the same edge node         │
                     │ communicate with each other?           │
                     └────────┬──────────────────┬───────────┘
                              │ No               │ Yes
                              ▼                  │
                     ┌──────────────────┐        │
                     │ Check CNI        │        │
                     │ (local routing)  │        │
                     └──────────────────┘        │
                                                 │
                     ┌───────────────────────────▼───────────────────┐
                     │ Can pods on different edge nodes communicate? │
                     └────────┬──────────────────────┬──────────────┘
                              │ No                   │ Yes
                              ▼                      ▼
                     ┌──────────────┐        ┌──────────────┐
                     │ EdgeMesh or  │        │ All good.    │
                     │ cross-node   │        │ No issue.    │
                     │ routing      │        └──────────────┘
                     └──────┬───────┘
                            │
                     ┌──────▼───────────────────────┐
                     │ Do Kubernetes Services        │
                     │ resolve across edge nodes?    │
                     └──────┬───────────┬───────────┘
                            │ No        │ Yes
                            ▼           ▼
                     ┌──────────┐  ┌──────────────────────┐
                     │ EdgeMesh │  │ Cross-node routing    │
                     │ DNS/LB   │  │ or network reachability│
                     └──────────┘  └──────────────────────┘
```

## Rule of Thumb

| Symptom | Likely Component |
|---------|-----------------|
| Pod stuck in `ContainerCreating` | CNI |
| Pod has no IP assigned | CNI |
| Same-edge-node pod-to-pod fails | CNI (local network plugin) |
| Same-edge-node works, cross-node fails | EdgeMesh or underlying network |
| `kubectl exec` works but service name fails to resolve | EdgeMesh DNS |
| Service IP works, service name does not | EdgeMesh DNS |

## Checking CNI

CNI (Container Network Interface) is responsible for assigning IP addresses to
pods and setting up local pod networking on each edge node. It is configured as
part of EdgeCore.

### What to check

1. **Is a CNI plugin installed on the edge node?**  
   CNI plugins must be present in the configured binary directory
   (default: `/opt/cni/bin` on Linux).

2. **Is the CNI config present?**  
   Config files should exist in the configured directory
   (default: `/etc/cni/net.d` on Linux).

3. **Is EdgeCore configured to use a network plugin?**  
   In the EdgeCore config, the `NetworkPluginName` field must be set
   (e.g., `cni`, `kubenet`). If left empty, no plugin is invoked.

4. **Check EdgeCore logs for CNI errors:**  
   ```
   journalctl -u edgecore | grep -i cni
   tail -f /var/log/kubeedge/edgecore.log | grep -i cni
   ```

### Key EdgeCore config fields

| Field | Default (Linux) | Purpose |
|-------|-----------------|---------|
| `modules.edged.networkPluginName` | `""` (empty) | CNI plugin name (set to `cni` to enable) |
| `modules.edged.cniConfDir` | `/etc/cni/net.d` | Directory containing CNI config files |
| `modules.edged.cniBinDir` | `/opt/cni/bin` | Comma-separated list of directories to search for CNI plugin binaries |
| `modules.edged.cniCacheDirs` | `/var/lib/cni/cache` | CNI cache directory |
| `modules.edged.networkPluginMTU` | `1500` | MTU passed to the network plugin |

When `networkPluginName` is empty, a noop plugin is used and pod
networking may not be set up.

## Checking EdgeMesh

EdgeMesh provides service discovery, load balancing, and cross-edge
communication for pods across different edge nodes. It is a separate project
([github.com/kubeedge/edgemesh](https://github.com/kubeedge/edgemesh)) and
must be installed independently — it is not bundled with KubeEdge.

### What to check

1. **Is EdgeMesh installed?**  
   Check if the EdgeMesh daemon is running on each edge node:
   ```
   kubectl get pods -n kubeedge -l app=edgemesh
   ```

2. **Are edge nodes in the same reachable network?**  
   EdgeMesh typically requires edge nodes to have network reachability to each
   other in the underlying network. If edge nodes are on separate, isolated
   subnets, EdgeMesh may need a cross-LAN or relay configuration.

3. **Does DNS resolution work for cross-edge services?**  
   Test service discovery from a pod on one edge node to a service backed
   by pods on a different edge node:
   ```
   kubectl exec <pod-on-edge-A> -- nslookup <service-name>
   kubectl exec <pod-on-edge-A> -- curl http://<service-name>
   ```

4. **Check EdgeMesh logs:**  
   ```
   kubectl logs -n kubeedge <edgemesh-pod>
   ```

## Common Scenarios

### Pods on same edge node can communicate, cross-node cannot

This is the classic EdgeMesh indicator. The CNI plugin is working correctly
(local pod networking is functional), but cross-node routing is not set up.
Ensure EdgeMesh is installed and the edge nodes have underlying network
reachability.

### Pods get IPs but services do not resolve

EdgeMesh provides DNS-based service discovery. If service IPs work but
service names fail, EdgeMesh DNS is the likely culprit. Verify the EdgeMesh
DNS component is running.

### Pods stuck in ContainerCreating

This is a CNI issue. The runtime cannot set up the pod network. Check that
the CNI plugin binaries and config files exist at the configured paths on
the edge node.

### Cross-node works but only for pods on some edge nodes

Check whether the failing edge nodes have EdgeMesh installed and running.
Also verify either direct network reachability between the specific edge
nodes or that the EdgeMesh cross-LAN/relay configuration is set up.
