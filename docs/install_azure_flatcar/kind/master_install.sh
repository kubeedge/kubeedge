#!/bin/bash

set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi

# create kubernetes cluster
kind create cluster --config=/home/core/kind-config.yaml
kubectl cluster-info --context kind-app-1-cluster

# install cloudcore
cd kubeedge-release-1.18/manifests/charts
helm upgrade --install cloudcore ./cloudcore --namespace kubeedge --create-namespace -f ./cloudcore/values.yaml
