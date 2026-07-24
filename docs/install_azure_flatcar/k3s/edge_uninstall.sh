#!/bin/bash
set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi


systemctl stop edgecore
rm -rf /etc/kubeedge/
# since not using docker, nor containerd but crio
crictl --runtime-endpoint unix:///run/containerd/containerd.sock stop $(crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -q )  # stop all containers
crictl --runtime-endpoint unix:///run/containerd/containerd.sock rm $(crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -a -q) # remove all containers
crictl --runtime-endpoint unix:///run/containerd/containerd.sock rmi $(crictl --runtime-endpoint unix:///run/containerd/containerd.sock images -q) # remove all images

crictl --runtime-endpoint unix:///run/containerd/containerd.sock images
crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps -a
