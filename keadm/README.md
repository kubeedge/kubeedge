# KubeEdge Installer

Click [here](https://kubeedge.io/en/docs/setup/keadm/) for detailed documentation of KubeEdge installer.

## keadm ctl Commands

For comprehensive documentation and examples for `keadm ctl` commands, see the official documentation:
- [keadm ctl Command Documentation](https://kubeedge.io/en/docs/setup/keadm-ctl/)

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
