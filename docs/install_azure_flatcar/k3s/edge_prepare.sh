#!/bin/bash
set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi


# configure and install cnis => NECESSARY?
cp /home/core/10-containerd-net.conflist /etc/cni/net.d/

mkdir -p /opt/cni/bin
wget https://github.com/containernetworking/plugins/releases/download/v1.5.1/cni-plugins-linux-amd64-v1.5.1.tgz
tar Cxzvf /opt/cni/bin cni-plugins-linux-amd64-v1.5.1.tgz
rm cni-plugins-linux-amd64-v1.5.1.tgz

mkdir -p /etc/containerd
containerd config default > config.toml
mv config.toml /etc/containerd/

systemctl restart containerd

wget https://github.com/kubeedge/kubeedge/releases/download/v1.18.0/keadm-v1.18.0-linux-amd64.tar.gz
tar -zxvf keadm-v1.18.0-linux-amd64.tar.gz
cp keadm-v1.18.0-linux-amd64/keadm/keadm /opt/bin/
rm keadm-v1.18.0-linux-amd64.tar.gz*