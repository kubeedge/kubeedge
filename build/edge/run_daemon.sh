#!/bin/bash
set -e

QEMU_VERSION="${QEMU_VERSION:-v3.0.0}"

main(){
    case $1 in
        "prepare")
            docker_prepare
            ;;
        "set")
            docker_set $@
            ;;
        "build")
            docker_build
            ;;
        "save")
            docker_save
            ;;
        "up")
            docker_up
            ;;
        "down")
            docker_down
            ;;
        "only_run_edge")
            docker_only_run_edge $@
            ;;
        *)
            usage
            exit 1
            ;;
    esac
}

usage() {
    echo "Usage:"
    echo "$0 prepare | set | build | save | up | down "
}

docker_prepare(){
    if [ ! -d /etc/kubeedge/certs ] || [ ! -e /etc/kubeedge/certs/edge.crt ] || [ ! -e /etc/kubeedge/certs/edge.key ]; then
        mkdir -p /etc/kubeedge/certs
        echo "Certificate does not exist"
        exit -1 
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

    eval $(sed -n '/CERTPATH/p' .env)
    eval $(sed -n '/CERTFILE/p' .env)
    eval $(sed -n '/KEYFILE/p' .env)
    if [ ! -d ${CERTPATH} ] || [ ! -e ${CERTFILE} ] || [ ! -e ${KEYFILE} ]; then
        mkdir -p ${CERTPATH}
        echo "Certificate does not exist"
        exit -1 
    fi

    if [[ -z $(which docker-compose) ]]; then
        curl -L "https://github.com/docker/compose/releases/download/1.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
    fi 
    echo "Container runtime environment check passed."
}

docker_set(){
    # This script accepts the following parameters:
    # 
    # * cloudhub
    # * edgename
    # * edgecore_image
    # * arch
    # * qemu_arch
    # * certpath
    # * certfile 
    # * keyfile 
    #
    # Example
    # 
    #  ./run_daemon.sh set \
    #    cloudhub=0.0.0.0:10000 \
    #    edgename=edge-node \
    #    edgecore_image="kubeedge/edgecore:latest" \
    #    arch=amd64 \
    #    qemu_arch=x86_64 \
    #    certpath=/etc/kubeedge/certs \
    #    certfile=/etc/kubeedge/certs/edge.crt \
    #    keyfile=/etc/kubeedge/certs/edge.key 

    ARGS=$@

    CONFIG=${ARGS#* }

    for line in $CONFIG; do
        eval "$line"
    done

    [[ ! -z $cloudhub ]] &&  sed -i "/CLOUDHUB=/c\CLOUDHUB=${cloudhub}" .env && echo "set cloudhub success"
    [[ ! -z $edgename ]] &&  sed -i "/EDGENAME=/c\EDGENAME=${edgename}" .env && echo "set edgename success"
    [[ ! -z $edgecore_image ]] &&  sed -i "/EDGECOREIMAGE=/c\EDGECOREIMAGE=${edgecore_image}" .env && echo "set edgecore_image success"
    [[ ! -z $arch ]] &&  sed -i "/\<ARCH\>/c\ARCH=${arch}" .env && echo "set arch success"
    [[ ! -z $qemu_arch ]] &&  sed -i "/QEMU_ARCH=/c\QEMU_ARCH=${qemu_arch}" .env && echo "set qemu_arch success"
    [[ ! -z $certpath ]] &&  sed -i "/CERTPATH=/c\CERTPATH=${certpath}" .env && echo "set certpath success"
    [[ ! -z $certfile ]] &&  sed -i "/CERTFILE=/c\CERTFILE=${certfile}" .env && echo "set certfile success"
    [[ ! -z $keyfile ]] &&  sed -i "/KEYFILE=/c\KEYFILE=${keyfile}" .env && echo "set keyfile success"
}

docker_build(){
    eval $(sed -n '/QEMU_ARCH/p' .env)

    # Prepare qemu to build images other then x86_64 on travis
    prepare_qemu ${QEMU_ARCH}

    docker-compose build
}

docker_save(){
    eval $(sed -n '/EDGECOREIMAGE/p' .env)
    docker save -o edgecore_image.tar $EDGECOREIMAGE
}

docker_up(){
    docker-compose up -d
}

docker_down(){
    docker-compose down 
}

docker_only_run_edge(){
    # This script accepts the following parameters:
    # 
    # * mqtt
    # * edgename
    # * cloudhub
    # * image
    # 
    # Example
    # 
    # ./run_daemon.sh only_run_edge mqtt=0.0.0.0:1883 cloudhub=0.0.0.0:10000 edgename=edge-node image="kubeedge/edgecore:latest"

    ARGS=$@

    CONFIG=${ARGS#* }

    for line in $CONFIG; do
        eval "$line"
    done

    mqtt=${mqtt:-"0.0.0.0:1883"}
    cloudhub=${cloudhub:-"0.0.0.0:10000"}
    edgename=${edgename:-$(hostname)}
    edgehubWebsocketUrl=wss://${cloudhub}/e632aba927ea4ac2b575ec1603d56f10/${edgename}/events 
    image=${image:-"kubeedge/edgecore:latest"}
    containername=${containername:-"edgecore"}

    docker run -d --name ${containername} --restart always \
        --cpu-period=50000 --cpu-quota=100000 --memory=1g --privileged \
        -e edgehub.websocket.certfile=/etc/kubeedge/certs/edge.crt \
        -e edgehub.websocket.keyfile=/etc/kubeedge/certs/edge.key \
        -e mqtt.server=${mqtt} \
        -e edgehub.websocket.url=${edgehubWebsocketUrl} \
        -e edged.hostname-override=${edgename} \
        -e edgehub.controller.node-id=${edgename} \
        -v /etc/kubeedge/certs:/etc/kubeedge/certs:ro \
        -v /var/lib/edged:/var/lib/edged \
        -v /var/lib/kubeedge:/var/lib/kubeedge \
        -v /var/run/docker.sock:/var/run/docker.sock \
        ${image}
}

prepare_qemu(){
    echo "PREPARE: Qemu"
    QEMU_ARCH=${1}
    # Prepare qemu to build non amd64 / x86_64 images
    docker run --rm --privileged multiarch/qemu-user-static:register --reset

    rm -rf tmp
    mkdir -p tmp
    
    pushd tmp &&
    curl -L -o qemu-${QEMU_ARCH}-static.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/$QEMU_VERSION/qemu-${QEMU_ARCH}-static.tar.gz && tar xzf qemu-${QEMU_ARCH}-static.tar.gz &&
    popd
}

main $@
