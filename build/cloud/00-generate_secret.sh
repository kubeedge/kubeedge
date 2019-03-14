#!/bin/sh

mkdir -p /etc/kubeedge/ca

docker run --rm -v /etc/kubeedge/ca:/etc/kubeedge/ca kubeedge/certgen:v0.2 buildSecret | tee ./06-secret.yaml
