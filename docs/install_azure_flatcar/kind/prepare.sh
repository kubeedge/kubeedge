#!/bin/bash
set -xe

IP_FILE="ip_addresses.txt"
RESOURCE_GROUP="KubeEdge"

write_ips_to_file() {
    # Delete the IP addresses file if it exists
    if [ -f "$IP_FILE" ]; then
        rm "$IP_FILE"
    fi

    # Get the list of VM names in the resource group
    VM_NAMES=$(az vm list --resource-group $RESOURCE_GROUP --query "[].name" -o tsv | sort -r)

    # Iterate over each VM name and get its public IP address
    for VM_NAME in $VM_NAMES; do
        IP_ADDRESS=$(az vm list-ip-addresses --resource-group $RESOURCE_GROUP --name $VM_NAME --query "[].virtualMachine.network.publicIpAddresses[0].ipAddress" -o tsv)
        echo "VM Name: $VM_NAME, IP Address: $IP_ADDRESS"
        # Write the IP address to the file
        echo "$IP_ADDRESS" >> "$IP_FILE"
    done

}

get_ip_addresses() {
    local ip_file="$1"
    local ip_addresses=()
    while IFS= read -r line; do
        ip_addresses+=("$line")
    done < "$ip_file"

    # Check if there are at least two IP addresses in the file
    if [ "${#ip_addresses[@]}" -lt 2 ]; then
        echo "The IP addresses file must contain at least two IP addresses."
        exit 1
    fi

    echo "${ip_addresses[@]}"
}

patch_template_files() {

    # Assign the first IP address to a variable
    FIRST_IP=$(echo "$IP_ADDRESSES" | awk '{print $1}')

    # Replace the IP tag in edgecore_template.yaml and write to edgecore.yaml
    sed "s|IP|$FIRST_IP|" edgecore_template.yaml > edgecore.yaml

    echo "The IP tag has been replaced with $FIRST_IP in edgecore.yaml"

    sed "s|IP|$FIRST_IP|" values_template.yaml > values.yaml
    sed "s|IP|$FIRST_IP|" edge_install.sh_template > edge_install.sh
}

get_kubeedge_charts()
{
    # Download the kubeedge charts
    # Download the tarball of the release-1.18 branch
    wget https://github.com/kubeedge/kubeedge/archive/refs/heads/release-1.18.tar.gz -O kubeedge-release-1.18.tar.gz
    # Extract the specific directory from the tarball
    tar -xzf kubeedge-release-1.18.tar.gz kubeedge-release-1.18/manifests/charts/cloudcore
    tar cvzf cloudcore-chart-1.18.tar.gz kubeedge-release-1.18/manifests/charts/cloudcore
    rm -rf kubeedge-release-1.18
    rm kubeedge-release-1.18.tar.gz

}


write_ips_to_file
IP_ADDRESSES=$(get_ip_addresses "$IP_FILE")

get_kubeedge_charts
patch_template_files

# loop over IP_ADDRESSES and scp the files to the VMs
MASTER_IP=$(echo "$IP_ADDRESSES" | awk '{print $1}')
EDGE_IP=$(echo "$IP_ADDRESSES" | awk '{print $2}')

scp -o StrictHostKeyChecking=no -i ~/.ssh/id_rsa master_prepare.sh master_install.sh master_uninstall.sh values.yaml 10-containerd-net.conflist cloudcore-chart-1.18.tar.gz kind-config.yaml core@${MASTER_IP}:~/
scp -o StrictHostKeyChecking=no -i ~/.ssh/id_rsa edge_prepare.sh edge_install.sh edge_uninstall.sh 10-containerd-net.conflist edgecore.yaml core@${EDGE_IP}:~/

