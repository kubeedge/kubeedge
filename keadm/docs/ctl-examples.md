# keadm ctl Command Examples

This document provides practical examples for using `keadm ctl` commands to manage edge nodes and workloads effectively.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Common Usage Patterns](#common-usage-patterns)
- [Command Examples](#command-examples)
  - [get](#get)
  - [restart](#restart)
  - [logs](#logs)
  - [exec](#exec)
  - [describe](#describe)
  - [edit](#edit)
  - [confirm](#confirm)
  - [unhold-upgrade](#unhold-upgrade)
- [Troubleshooting Scenarios](#troubleshooting-scenarios)
- [Integration with Batch Operations](#integration-with-batch-operations)

## Prerequisites

Before using `keadm ctl` commands, ensure:

1. **KubeEdge is installed** on both cloud and edge nodes
2. **Edge nodes are connected** to cloud hub
3. **kubectl is configured** to access Kubernetes cluster
4. **Proper permissions** are set for edge node management

```bash
# Verify keadm installation
keadm version

# Check edge node status
kubectl get nodes

# Test connectivity
keadm ctl get pods --node <edge-node-name>
```

## Common Usage Patterns

### Targeting Specific Nodes

Most `keadm ctl` commands operate on specific edge nodes:

```bash
# Single node operations
keadm ctl [command] --node edge-node-01

# Multiple nodes (use with batch operations)
keadm ctl [command] --nodes edge-node-01,edge-node-02,edge-node-03
```

### Namespace Operations

```bash
# Default namespace
keadm ctl get pods --node edge-node-01

# Specific namespace
keadm ctl get pods --node edge-node-01 --namespace kube-system

# All namespaces
keadm ctl get pods --node edge-node-01 --all-namespaces
```

### Label Selectors

```bash
# Filter by labels
keadm ctl get pods --node edge-node-01 --selector app=nginx

# Multiple selectors
keadm ctl get pods --node edge-node-01 --selector app=nginx,env=production
```

## Command Examples

### get

Retrieve resources from edge nodes.

#### Get Pods

```bash
# Basic pod listing
keadm ctl get pods --node edge-node-01

# Wide format output
keadm ctl get pods --node edge-node-01 --output wide

# JSON output
keadm ctl get pods --node edge-node-01 --output json

# All namespaces
keadm ctl get pods --node edge-node-01 --all-namespaces

# With label selector
keadm ctl get pods --node edge-node-01 --selector app=database

# Specific namespace
keadm ctl get pods --node edge-node-01 --namespace monitoring
```

#### Get Devices

```bash
# List all devices on edge node
keadm ctl get devices --node edge-node-01

# Filter by device type
keadm ctl get devices --node edge-node-01 --selector type=sensor
```

### restart

Restart resources on edge nodes.

#### Restart EdgeCore

```bash
# Restart edgecore on specific node
keadm ctl restart edgecore --node edge-node-01

# Restart edgecore with timeout
keadm ctl restart edgecore --node edge-node-01 --timeout 30s
```

#### Restart Pods

```bash
# Restart specific pod
keadm ctl restart pod nginx-pod --node edge-node-01

# Restart multiple pods
keadm ctl restart pod nginx-pod database-pod --node edge-node-01

# Force restart
keadm ctl restart pod nginx-pod --node edge-node-01 --force
```

### logs

Retrieve logs from edge pods.

#### Basic Log Retrieval

```bash
# Get recent logs
keadm ctl logs nginx-pod --node edge-node-01

# Follow logs in real-time
keadm ctl logs nginx-pod --node edge-node-01 --follow

# Get last 100 lines
keadm ctl logs nginx-pod --node edge-node-01 --tail 100

# Logs from specific time
keadm ctl logs nginx-pod --node edge-node-01 --since 1h
```

#### Advanced Log Operations

```bash
# Previous container logs
keadm ctl logs nginx-pod --node edge-node-01 --previous

# With timestamps
keadm ctl logs nginx-pod --node edge-node-01 --timestamps

# Specific container
keadm ctl logs nginx-pod --node edge-node-01 --container nginx
```

### exec

Execute commands in edge pods.

#### Command Execution

```bash
# Interactive shell
keadm ctl exec nginx-pod --node edge-node-01 -- /bin/bash

# Single command
keadm ctl exec nginx-pod --node edge-node-01 -- ls -la /app

# Multiple commands
keadm ctl exec nginx-pod --node edge-node-01 -- "cd /app && ls -la"
```

#### Advanced Execution

```bash
# With specific container
keadm ctl exec nginx-pod --node edge-node-01 --container sidecar -- ps aux

# Pass environment variables
keadm ctl exec nginx-pod --node edge-node-01 --env VAR=value -- printenv

# Non-interactive mode
keadm ctl exec nginx-pod --node edge-node-01 --stdin -- "echo 'hello world'" < input.txt
```

### describe

Get detailed information about resources.

#### Describe Pods

```bash
# Pod details
keadm ctl describe pod nginx-pod --node edge-node-01

# Events for pod
keadm ctl describe pod nginx-pod --node edge-node-01 --show-events

# Wide output format
keadm ctl describe pod nginx-pod --node edge-node-01 --output wide
```

#### Describe Nodes

```bash
# Node information
keadm ctl describe node edge-node-01

# Node conditions
keadm ctl describe node edge-node-01 --conditions-only
```

### edit

Edit resources on edge nodes.

#### Edit Pods

```bash
# Edit pod configuration
keadm ctl edit pod nginx-pod --node edge-node-01

# Open in specific editor
keadm ctl edit pod nginx-pod --node edge-node-01 --editor vim

# Edit with validation
keadm ctl edit pod nginx-pod --node edge-node-01 --validate
```

#### Edit ConfigMaps

```bash
# Edit ConfigMap
keadm ctl edit configmap app-config --node edge-node-01 --namespace default
```

### confirm

Send confirmation signals to MetaService API.

#### Confirm Operations

```bash
# Confirm node readiness
keadm ctl confirm node --node edge-node-01

# Confirm pod status
keadm ctl confirm pod nginx-pod --node edge-node-01

# Confirm with timeout
keadm ctl confirm node --node edge-node-01 --timeout 60s
```

### unhold-upgrade

Release upgrade holds for pods or nodes.

#### Unhold Operations

```bash
# Release upgrade hold for pod
keadm ctl unhold-upgrade nginx-pod --node edge-node-01

# Release hold for all pods on node
keadm ctl unhold-upgrade --all --node edge-node-01

# Force unhold
keadm ctl unhold-upgrade nginx-pod --node edge-node-01 --force
```

## Troubleshooting Scenarios

### Connection Issues

```bash
# Check if edge node is accessible
keadm ctl get pods --node edge-node-01

# If connection fails, check network
ping edge-node-01

# Verify edgecore status
systemctl status edgecore  # on edge node
```

### Pod Not Found

```bash
# Check all namespaces
keadm ctl get pods --node edge-node-01 --all-namespaces

# Search for pod with partial name
keadm ctl get pods --node edge-node-01 --selector name=nginx

# Describe node to check status
keadm ctl describe node edge-node-01
```

### Permission Denied

```bash
# Check current user context
kubectl config current-context

# Verify permissions
kubectl auth can-i get pods --node edge-node-01

# Use service account if needed
keadm ctl get pods --node edge-node-01 --as system:serviceaccount
```

### High Resource Usage

```bash
# Check pod resource usage
keadm ctl describe pod nginx-pod --node edge-node-01

# Monitor node resources
keadm ctl describe node edge-node-01

# Restart resource-heavy pods
keadm ctl restart pod memory-hog --node edge-node-01
```

## Integration with Batch Operations

The `keadm ctl` commands can be combined with batch operations for managing multiple edge nodes efficiently.

### Using Node Selector

```bash
# Get pods from multiple nodes
keadm ctl get pods --nodes edge-node-01,edge-node-02,edge-node-03

# Restart services on multiple nodes
keadm ctl restart pod nginx-pod --nodes edge-node-01,edge-node-02

# Get logs from multiple nodes
keadm ctl logs nginx-pod --nodes edge-node-01,edge-node-02 --follow
```

### Using Label Selectors

```bash
# Operate on nodes with specific labels
keadm ctl get pods --selector region=us-west,env=production

# Restart database pods across regions
keadm ctl restart pod --selector app=database --all-namespaces

# Update configuration for specific environment
keadm ctl edit configmap --selector env=staging
```

### Batch Workflow Examples

```bash
# 1. Check status of all production nodes
keadm ctl get pods --selector env=production --all-namespaces

# 2. Restart problematic services
keadm ctl restart pod --selector app=legacy --all-namespaces

# 3. Verify restart success
keadm ctl get pods --selector app=legacy --all-namespaces

# 4. Monitor logs
keadm ctl logs --selector app=legacy --follow --all-namespaces
```

## Best Practices

1. **Use specific namespaces** to avoid conflicts
2. **Label resources consistently** for easier management
3. **Test commands on single nodes** before batch operations
4. **Use timeouts** for operations that might hang
5. **Monitor logs** during restart operations
6. **Backup configurations** before making changes

## Tips and Tricks

### Output Formatting

```bash
# Table format (default)
keadm ctl get pods --node edge-node-01

# JSON for scripting
keadm ctl get pods --node edge-node-01 --output json | jq '.items[].metadata.name'

# Wide format for more details
keadm ctl get pods --node edge-node-01 --output wide
```

### Scripting Examples

```bash
#!/bin/bash
# Restart all nginx pods across edge nodes
NODES=("edge-node-01" "edge-node-02" "edge-node-03")

for node in "${NODES[@]}"; do
    echo "Processing node: $node"
    keadm ctl restart pod --selector app=nginx --node "$node"
done
```

### Environment Variables

```bash
# Set default namespace
export KUBECONFIG=/path/to/kubeconfig
export KUBECTL_NAMESPACE=production

# Use with keadm ctl
keadm ctl get pods --node edge-node-01  # Uses production namespace
```

## Related Commands

- [`keadm reset`](https://kubeedge.io/en/docs/setup/keadm/#reset) - Reset edge nodes
- [`keadm join`](https://kubeedge.io/en/docs/setup/keadm/#join) - Join edge nodes to cluster
- [`keadm upgrade`](https://kubeedge.io/en/docs/setup/keadm/#upgrade) - Upgrade edge nodes

## Getting Help

For more information about any command:

```bash
# General help
keadm ctl --help

# Command-specific help
keadm ctl get --help
keadm ctl restart --help
keadm ctl logs --help
```

## Contributing

To contribute to this documentation:

1. **Test examples** before submitting
2. **Include real-world scenarios** 
3. **Provide context** for examples
4. **Update related commands** when adding new examples

For issues or suggestions, please [open an issue](https://github.com/kubeedge/kubeedge/issues).
