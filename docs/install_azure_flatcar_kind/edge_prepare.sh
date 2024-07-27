#!/bin/bash
set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi


wget https://github.com/kubeedge/kubeedge/releases/download/v1.17.0/keadm-v1.17.0-linux-amd64.tar.gz
tar -zxvf keadm-v1.17.0-linux-amd64.tar.gz
cp keadm-v1.17.0-linux-amd64/keadm/keadm /opt/bin/
rm keadm-v1.17.0-linux-amd64.tar.gz*