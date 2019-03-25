#!/bin/sh

if [ ! -d /etc/kubeedge/edge/certs ] || [ ! -e /etc/kubeedge/edge/certs/edge.crt ] || [ ! -e /etc/kubeedge/edge/certs/edge.key ]; then
    mkdir -p /etc/kubeedge/edge/certs
    mkdir -p /etc/kubeedge/ca
    ../tools/certgen.sh genCertAndKey edge
fi

if [ ! -d /var/lib/kubeedge ]; then
    mkdir -p /var/lib/kubeedge
fi

if [ ! -d /var/lib/edged ]; then
    mkdir -p /var/lib/edged
fi

if [ ! -S /var/run/docker.sock ]; then
    echo "docker engine not found"
    exit -1
fi

readonly edgeCoreImage=${EDGE_CORE_IMAGE:-kubeedge/edgecore}

start() {
    local mqttBroker=$1
    local cloudHub=$2
    local edgeCoreImageVersion=${3:-latest}
    docker run -d --name edgecore --restart always \
    --cpu-period=50000 --cpu-quota=100000 --memory=1g --privileged \
    -e mqtt.server=${mqttBroker} \
    -e edgehub.websocket.url=${cloudHub} \
    -v /etc/kubeedge/edge/certs:/etc/kubeedge/edge/certs:ro \
    -v /var/lib/edged:/var/lib/edged \
    -v /var/lib/kubeedge:/var/lib/kubeedge \
    -v /var/run/docker.sock:/var/run/docker.sock \
    ${edgeCoreImage}:${edgeCoreImageVersion}
}

start $@
