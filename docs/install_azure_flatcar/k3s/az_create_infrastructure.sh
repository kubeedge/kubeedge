#!/bin/bash
set -xe

# Check for --location argument
if [[ "$#" -lt 2 || "$1" != "--location" ]]; then
    echo "Usage: $0 --location <location>"
    exit 1
fi
# Assign the location value
LOCATION=$2

# List of VM names
VM_NAMES=(
    "kube-master"
    "kube-edge1"
   #  "kube-edge2"
)

RESOURCE_GROUP="KubeEdge1"


create_rg() {
    # Create a resource group

    az group create --name $RESOURCE_GROUP --location $LOCATION
}

create_network() {
    VM_NAME=$1
    local VNET_NAME="MyVnet-$VM_NAME"
    local SUBNET_NAME="MySubnet-$VM_NAME"
    local PUBLIC_IP_NAME="MyPublicIP-$VM_NAME"
    local NSG_NAME="MyNSG-$VM_NAME"
    local NIC_NAME="MyNIC-$VM_NAME"
    # Create a virtual network and subnet
    az network vnet create --resource-group $RESOURCE_GROUP --name $VNET_NAME --subnet-name $SUBNET_NAME

    # Create a public IP address
    az network public-ip create --resource-group $RESOURCE_GROUP --name $PUBLIC_IP_NAME

    # Create a network security group and allow SSH
    az network nsg create --resource-group $RESOURCE_GROUP --name $NSG_NAME

    # Create a network interface card
    az network nic create --resource-group $RESOURCE_GROUP --name $NIC_NAME --vnet-name $VNET_NAME --subnet $SUBNET_NAME --public-ip-address $PUBLIC_IP_NAME --network-security-group $NSG_NAME
}

create_fw_rules() {
    VM_NAME=$1
    local NSG_NAME="MyNSG-$VM_NAME"
    az network nsg rule create --resource-group $RESOURCE_GROUP --nsg-name $NSG_NAME --name AllowSSH --protocol Tcp --priority 1000 --destination-port-range 22 --access Allow --direction Inbound

    # # only for master ecs-cloud
    if [ "$VM_NAME" = "kube-master" ]; then
        az network nsg rule create --resource-group $RESOURCE_GROUP --nsg-name $NSG_NAME --name cloudcore --protocol Tcp --priority 1001 --destination-port-range 30000-30020 --access Allow --direction Inbound
        # az network nsg rule create --resource-group $RESOURCE_GROUP --nsg-name $NSG_NAME --name kubernetes --protocol Tcp --priority 1002 --destination-port-range 6443 --access Allow --direction Inbound
    fi
}

create_vm() {
    VM_NAME=$1
    local NIC_NAME="MyNIC-$VM_NAME"
    # Create a virtual machine
    VM_SIZE="Standard_B4ms"
    IMAGE="kinvolk:flatcar-container-linux-free:stable-gen2:3815.2.5" # pinned version !
    # IMAGE="kinvolk:flatcar-container-linux-free:stable-gen2:latest"
    ADMIN_USERNAME="core"
    # ADMIN_PASSWORD="xxxx"
    
    az vm image terms accept --publish kinvolk --offer flatcar-container-linux-free --plan stable-gen2

    #az vm image list --all -p kinvolk -f flatcar -s stable-gen2 --query '[-1]'
    az vm create --resource-group $RESOURCE_GROUP --location $LOCATION --name $VM_NAME --nics $NIC_NAME --image $IMAGE --size $VM_SIZE --admin-username $ADMIN_USERNAME --user-data ./ignition.json
}



create_rg
# patch the ignition file with the ssh key
sed "s|SSHKEYHERE|$(cat ~/.ssh/id_rsa.pub)|" ignition_template.json > ignition.json

for vm_name in "${VM_NAMES[@]}"; do
    echo "VM Key: $vm_key, VM Name: $vm_name"
    create_network $vm_name
    create_fw_rules $vm_name
    create_vm $vm_name
done
