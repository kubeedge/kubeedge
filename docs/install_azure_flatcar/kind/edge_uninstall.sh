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
crictl stop $(crictl ps -q) # stop all containers
crictl rm $(crictl ps -a -q) # remove all containers
crictl rmi $(crictl images -q) # remove all images

crictl images
crictl ps -a