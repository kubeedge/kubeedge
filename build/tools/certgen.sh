#!/usr/bin/env bash

set -o errexit

readonly caPath=${CA_PATH:-/etc/kubeedge/ca}
readonly certPath=${CERT_PATH:-/etc/kubeedge/certs}
readonly subject=${SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge/CN=kubeedge.io}

readonly keyType=${KEY_TYPE:-ec}
readonly keyPassword=${KEY_PASSWORD:-kubeedge.io}
readonly usePassword=${USE_PASSWORD:-false}

genCA() {
    if [ "$keyType" = "rsa" ]; then
        if [ "$usePassword" = "true" ]; then
            openssl genrsa -des3 -passout pass:$keyPassword -out "${caPath}/rootCA.key" 4096
        else
            openssl genrsa -out "${caPath}/rootCA.key" 4096
        fi

        local reqCmd=(
            openssl req -x509 -new -key "${caPath}/rootCA.key"
            -sha256 -days 3650
            -subj "$subject"
            -out "${caPath}/rootCA.crt"
        )

        if [ "$usePassword" = "true" ]; then
            reqCmd+=( -passin pass:$keyPassword )
        fi

        "${reqCmd[@]}"
    else
        if [ "$usePassword" = "true" ]; then
            openssl ecparam -name prime256v1 -genkey -noout \
                | openssl ec -aes256 -passout pass:$keyPassword -out "${caPath}/rootCA.key"
        else
            openssl ecparam -name prime256v1 -genkey -noout -out "${caPath}/rootCA.key"
        fi

        local reqCmd=(
            openssl req -x509 -new -key "${caPath}/rootCA.key"
            -sha256 -days 3650
            -subj "$subject"
            -out "${caPath}/rootCA.crt"
        )

        if [ "$usePassword" = "true" ]; then
            reqCmd+=( -passin pass:$keyPassword )
        fi

        "${reqCmd[@]}"
    fi
}

ensureCA() {
    if [ ! -e "${caPath}/rootCA.key" ] || [ ! -e "${caPath}/rootCA.crt" ]; then
        genCA
    fi
}

ensureFolder() {
    if [ ! -d "${caPath}" ]; then
        mkdir -p "${caPath}"
    fi
    if [ ! -d "${certPath}" ]; then
        mkdir -p "${certPath}"
    fi
}


ensureCommand() {
    echo "checking if $1 command exists."
    if command -v "$1" >/dev/null 2>&1; then
        echo "$1 exists."
    else
        echo "Error: $1 not found, please install $1 command."
        exit 1
    fi
}

genCsr() {
    local name=$1
    if [ "$keyType" = "rsa" ]; then
        openssl genrsa -out "${certPath}/${name}.key" 2048
    else
        openssl ecparam -name prime256v1 -genkey -noout -out "${certPath}/${name}.key"
    fi
    openssl req -new -key "${certPath}/${name}.key" -subj "$subject" -out "${certPath}/${name}.csr"
}

genCert() {
    local name=$1
    local ips=(${@:2})
    local subjAlt="subjectAltName = IP.1:127.0.0.1"
    local index=1

    for ip in ${ips[*]}; do
        index=$((index + 1))
        subjAlt="${subjAlt},IP.${index}:${ip}"
    done

    echo "$subjAlt" > /tmp/server-extfile.cnf

    local cmd=(
        openssl x509 -req -in "${certPath}/${name}.csr"
        -CA "${caPath}/rootCA.crt"
        -CAkey "${caPath}/rootCA.key"
        -CAcreateserial
        -out "${certPath}/${name}.crt"
        -days 365 -sha256
        -extfile /tmp/server-extfile.cnf
    )

    if [ "$usePassword" = "true" ]; then
        cmd+=( -passin pass:$keyPassword )
    fi

    "${cmd[@]}"
}

genCertAndKey() {
    ensureFolder
    ensureCA
    local name=$1
    genCsr "$name"
    genCert "$name" "${@:2}"
}

stream() {
    ensureFolder
    ensureCommand openssl
    readonly streamSubject=${SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge}
    readonly streamKeyFile=${certPath}/stream.key
    readonly streamCsrFile=${certPath}/stream.csr
    readonly streamCrtFile=${certPath}/stream.crt

    readonly k8sCaFile=${K8SCA_FILE:-/etc/kubernetes/pki/ca.crt}
    readonly k8sCaKeyFile=${K8SCA_KEY_FILE:-/etc/kubernetes/pki/ca.key}

    if [ -z "${CLOUDCOREIPS}" ] && [ -z "${CLOUDCORE_DOMAINS}" ]; then
        echo "You must set at least one of CLOUDCOREIPS or CLOUDCORE_DOMAINS Env.These environment
variables are set to specify the IP addresses or domains of all cloudcore, respectively."
        echo "If there are more than one IP or domain, you need to separate them with a space within a single env."
        exit 1
    fi

    index=1
    subjectAltName="subjectAltName = IP.1:127.0.0.1"
    for ip in ${CLOUDCOREIPS}; do
        index=$((index+1))
        subjectAltName="${subjectAltName},IP.${index}:${ip}"
    done

    for domain in ${CLOUDCORE_DOMAINS}; do
        subjectAltName="${subjectAltName},DNS:${domain}"
    done

    cp "${k8sCaFile}" "${caPath}/streamCA.crt"
    echo "$subjectAltName" > /tmp/server-extfile.cnf

    if [ "$keyType" = "rsa" ]; then
        openssl genrsa -out "${streamKeyFile}" 2048
    else
        openssl ecparam -name prime256v1 -genkey -noout -out "${streamKeyFile}"
    fi

    openssl req -new -key "${streamKeyFile}" -subj "$streamSubject" -out "${streamCsrFile}"
    # verify
    openssl req -in ${streamCsrFile} -noout -text
    openssl x509 -req -in "${streamCsrFile}" -CA "${k8sCaFile}" -CAkey "${k8sCaKeyFile}" -CAcreateserial \
        -out "${streamCrtFile}" -days 5000 -sha256 -extfile /tmp/server-extfile.cnf
    # verify
    openssl x509 -in ${streamCrtFile} -text -noout
}

opts(){
  usage() { echo "Usage: $0 [-i] ip1,ip2,..."; exit; }
  local OPTIND
  while getopts ':i:h' opt; do
    case $opt in
        i) IFS=','
           ips=($OPTARG)
           ;;
        h) usage;;
        ?) usage;;
    esac
  done
    echo "${ips[*]}"
}

edgesiteServer(){
    serverIPs="$(opts $*)"
    if [[ $serverIPs == *"Usage:"* ]];then
        echo $serverIPs
        exit 1
    fi
    local name=edgesite-server
    ensureFolder
    ensureCA
    genCsr "$name"
    genCert "$name" $serverIps
    genCsr server
    genCert server $serverIps
}

edgesiteAgent() {
    ensureFolder
    ensureCA
    local name=edgesite-agent
    genCsr "$name"
    genCert "$name"
}

buildSecret() {
    local name="edge"
    genCertAndKey "$name" > /dev/null 2>&1
    cat <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cloudcore
  namespace: kubeedge
  labels:
    k8s-app: kubeedge
    kubeedge: cloudcore
stringData:
  rootCA.crt: |
$(pr -T -o 4 "${caPath}/rootCA.crt")
  edge.crt: |
$(pr -T -o 4 "${certPath}/${name}.crt")
  edge.key: |
$(pr -T -o 4 "${certPath}/${name}.key")
EOF
}

$@
