#!/bin/bash

set -xe

# Assert that the current working directory is /root
if [ "$(pwd)" != "/root" ]; then
    echo "Error: The script must be run from the /root directory."
    exit 1
fi

# install cloudcore
cd kubeedge-release-1.18/manifests/charts
helm upgrade --install cloudcore ./cloudcore --namespace kubeedge --create-namespace -f ./cloudcore/values.yaml

# wait until cloudcore pod is running
while [[ $(kubectl get pods -l kubeedge=cloudcore -n kubeedge -o jsonpath="{.items[0].status.phase}") != "Running" ]]; do
    echo "Waiting for cloudcore pod to be running..."
    sleep 5
done

# re-apply these if it does not work
kubectl apply -f /home/core/traefik-ingress-route-tcp0.yaml
kubectl apply -f /home/core/traefik-ingress-route-tcp1.yaml
kubectl apply -f /home/core/traefik-ingress-route-tcp2.yaml
kubectl apply -f /home/core/traefik-ingress-route-tcp3.yaml
kubectl apply -f /home/core/traefik-ingress-route-tcp4.yaml