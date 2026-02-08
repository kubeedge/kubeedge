# KubeEdge Installer

Click [here](https://kubeedge.io/en/docs/setup/keadm/) for detailed documentation of KubeEdge installer.

## keadm ctl Command Examples

For comprehensive examples and usage patterns for `keadm ctl` commands, see [ctl-examples.md](docs/ctl-examples.md).

### Quick Examples

```bash
# Get pods from edge node
keadm ctl get pods --node edge-node-01

# Restart edgecore service
keadm ctl restart edgecore --node edge-node-01

# Get pod logs
keadm ctl logs nginx-pod --node edge-node-01 --follow

# Execute command in pod
keadm ctl exec nginx-pod --node edge-node-01 -- /bin/bash

# Describe node status
keadm ctl describe node edge-node-01
