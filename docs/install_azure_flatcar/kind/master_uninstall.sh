#!/bin/bash
set -xe


# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi

kubectl delete ns kubeedge # sufficient to delete cloudcore components

kind delete cluster --name app-1-cluster
