# Troubleshooting: Network Connectivity vs Cloud-Edge State Synchronization

Many KubeEdge failures present similar symptoms — workloads misbehave or edge
nodes appear unhealthy — even though the root cause is different. This guide
helps you quickly determine whether the problem is a **network connectivity
failure** or a **cloud-edge state synchronization issue**.

## Quick Checklist

```
1. Can the edge node reach CloudCore?        → ping / curl CloudCore IP
2. Is EdgeHub connected?                     → check edgecore logs
3. Is the edge node Ready in kubectl?        → kubectl get node <edge-node>
4. Are device twins / pod specs up to date? → check MetaManager / DeviceTwin
5. Do logs show "timeout" or "disconnect"?  → network issue
6. Do logs show "version conflict" or "out of sync"? → sync issue
```

---

## Symptoms

### Network Connectivity Failures

| Symptom | Detail |
|---|---|
| Edge node is `NotReady` | EdgeHub cannot reach CloudCore |
| Pods stuck in `Pending` | Scheduler decisions not reaching edge |
| No new pods scheduled after cloud reconnect | Network still blocked |
| `error dialing` in edgecore logs | TLS or route failure |
| `connection refused` / `timeout` in logs | CloudCore unreachable |

### Cloud-Edge State Synchronization Issues

| Symptom | Detail |
|---|---|
| Edge node is `Ready` but workloads misbehave | Sync lag or version conflict |
| Device twin value is stale | MetaManager has outdated state |
| Pod spec is old after an update | MetaDB has cached stale spec |
| Config changes not reflected on edge | Sync queue backed up |
| `version conflict` in DeviceTwin logs | Race between cloud and edge writes |

---

## Commands and Logs to Check

### Step 1 — Verify network reachability

```bash
# From the edge node, test CloudCore websocket port (default 10000)
curl -k https://<cloudcore-ip>:10000/

# Check basic IP reachability
ping <cloudcore-ip>
```

### Step 2 — Check EdgeHub connection status

```bash
# On the edge node
journalctl -u edgecore -n 100 --no-pager | grep -E "connection failed|connection is broken|websocket read error|websocket write error"
```

Key log patterns:

| Log message | Meaning |
|---|---|
| `connection failed: ...` | Could not reach CloudCore, will retry |
| `connection is broken, will reconnect after ...` | Connection dropped mid-session |
| `websocket read error: ...` | CloudCore closed the connection |
| `websocket write error: ...` | Failed to send keepalive to CloudCore |

### Step 3 — Check edge node status from cloud

```bash
kubectl get node <edge-node-name>
kubectl describe node <edge-node-name> | grep -A5 Conditions
```

### Step 4 — Check MetaManager sync state

```bash
# On the edge node — inspect the local MetaDB
journalctl -u edgecore -n 200 --no-pager | grep -i "meta\|sync\|version"
```

Key log patterns:

| Log message | Meaning |
|---|---|
| `version conflict` | DeviceTwin write conflict |
| `failed to get meta` | MetaDB read error |
| `sync done` | State successfully reconciled |
| `out of sync` | Cloud and edge state diverged |

### Step 5 — Check DeviceTwin for device state issues

```bash
journalctl -u edgecore -n 200 --no-pager | grep -i "twin\|device"
```

---

## Decision Tree

```
Is the edge node Ready in kubectl?
│
├── NO → Network issue likely
│         • Check CloudCore reachability (Step 1)
│         • Check EdgeHub logs for "disconnect" / "timeout" (Step 2)
│         • Verify TLS certificates are valid
│
└── YES → Sync issue likely
          • Check MetaManager for "version conflict" (Step 4)
          • Check DeviceTwin logs (Step 5)
          • Restart edgecore to force re-sync:
              systemctl restart edgecore
```

---

## Common Fixes

| Root cause | Fix |
|---|---|
| CloudCore IP changed | Update `edgehub.websocket.server` in edgecore config, restart |
| Expired TLS certificate | Rotate certs with `keadm certgen` |
| MetaDB stale state | **Destructive:** Delete `/var/lib/kubeedge/edgecore.db` and restart edgecore. All cached pod specs and device states will be lost and must re-sync from CloudCore on reconnect. |
| DeviceTwin version conflict | Restart edgecore; cloud state wins on reconnect |
| Firewall blocking port 10000 | Open TCP 10000 (websocket) and 10002 (quic) on CloudCore host |

---

## Related

- [keadm debug proposal](../proposals/sig-cluster-lifecycle/keadm-debug.md)
