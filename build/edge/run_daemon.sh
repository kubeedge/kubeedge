#!/bin/sh

if [ ! -d /etc/kubeedge/edge/certs ] || [ ! -e /etc/kubeedge/edge/certs/edge.crt ] || [ ! -e /etc/kubeedge/edge/certs/edge.key ]; then
    mkdir -p /etc/kubeedge/edge/certs
    mkdir -p /etc/kubeedge/ca
    docker run --rm \
    -v /etc/kubeedge/ca:/etc/kubeedge/ca \
    -v /etc/kubeedge/edge/certs:/etc/kubeedge/certs \
    kubeedge/certgen:v0.2 genCertAndKey edge
fi

if [ ! -d /var/lib/kubeedge ]; then
    mkdir -p /var/lib/kubeedge
fi

if [ ! -S /var/run/docker.sock ]; then
    echo "docker engine not found"
    exit -1
fi

start() {
    local mqttBroker=$1
    local cloudHub=$2
    docker run -d --name edgecore --restart always \
    -e mqtt.server=${mqttBroker} \
    -e edgehub.websocket.url=${cloudHub} \
    -v /etc/kubeedge/edge/certs:/etc/kubeedge/edge/certs:ro \
    -v /var/lib/kubeedge:/var/lib/kubeedge \
    -v /var/run/docker.sock:/var/run/docker.sock \
    kubeedge/edgecore:v0.2
}

start $@
