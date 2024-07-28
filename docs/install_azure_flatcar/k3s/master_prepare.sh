#!/bin/bash
set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi

# install kubectl same version as kubernetes, see kind-config.yaml
curl -LO https://dl.k8s.io/release/v1.28.6/bin/linux/amd64/kubectl
mv kubectl /opt/bin
chmod u+x /opt/bin/kubectl


# configure and install cnis
cp /home/core/10-containerd-net.conflist /etc/cni/net.d/

mkdir -p /opt/cni/bin
wget https://github.com/containernetworking/plugins/releases/download/v1.5.1/cni-plugins-linux-amd64-v1.5.1.tgz
tar Cxzvf /opt/cni/bin cni-plugins-linux-amd64-v1.5.1.tgz
rm cni-plugins-linux-amd64-v1.5.1.tgz

mkdir -p /etc/containerd
containerd config default > config.toml
mv config.toml /etc/containerd/

systemctl restart containerd

# install k3s
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="v1.28.6+k3s2" INSTALL_K3S_BIN_DIR=/opt/bin sh -


# install helm
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
export HELM_INSTALL_DIR="/opt/bin"
./get_helm.sh

# prepare installation of kubeedge
cp /home/core/cloudcore-chart-1.18.tar.gz /root/
tar xvzf cloudcore-chart-1.18.tar.gz && rm cloudcore-chart-1.18.tar.gz
# patch the values.yaml file
cp /home/core/values.yaml kubeedge-release-1.18/manifests/charts/cloudcore/


wget https://github.com/kubeedge/kubeedge/releases/download/v1.18.0/keadm-v1.18.0-linux-amd64.tar.gz
tar -zxvf keadm-v1.18.0-linux-amd64.tar.gz
cp keadm-v1.18.0-linux-amd64/keadm/keadm /opt/bin/
rm keadm-v1.18.0-linux-amd64.tar.gz*


cp /home/core/traefik-config.yaml /var/lib/rancher/k3s/server/manifests/
