package util

//Edge controller Configuration files and certificate generator script
var (
	CertGenSh = []byte(`#!/bin/sh

readonly caPath=${CA_PATH:-/etc/kubeedge/ca}
readonly caSubject=${CA_SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge/CN=kubeedge.io}
readonly certPath=${CERT_PATH:-/etc/kubeedge/certs}
readonly subject=${SUBJECT:-/C=CN/ST=Zhejiang/L=Hangzhou/O=KubeEdge/CN=kubeedge.io}

genCA() {
    openssl genrsa -des3 -out ${caPath}/rootCA.key -passout pass:kubeedge.io 4096
    openssl req -x509 -new -nodes -key ${caPath}/rootCA.key -sha256 -days 3650 \
    -subj ${subject} -passin pass:kubeedge.io -out ${caPath}/rootCA.crt
}

ensureCA() {
    if [ ! -e ${caPath}/rootCA.key ] || [ ! -e ${caPath}/rootCA.crt ]; then
        genCA
    fi
}

ensureFolder() {
    if [ ! -d ${caPath} ]; then
        mkdir -p ${caPath}
    fi
    if [ ! -d ${certPath} ]; then
        mkdir -p ${certPath}
    fi
}

genCertAndKey() {
    ensureFolder
    ensureCA
    local name=$1
    openssl genrsa -out ${certPath}/${name}.key 2048
    openssl req -new -key ${certPath}/${name}.key -subj ${subject} -out ${certPath}/${name}.csr
    openssl x509 -req -in ${certPath}/${name}.csr -CA ${caPath}/rootCA.crt -CAkey ${caPath}/rootCA.key \
    -CAcreateserial -passin pass:kubeedge.io -out ${certPath}/${name}.crt -days 365 -sha256
}

buildSecret() {
    local name="edge"
    genCertAndKey ${name} > /dev/null 2>&1
    cat <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: edgecontroller
  namespace: kubeedge
  labels:
    k8s-app: kubeedge
    kubeedge: edgecontroller
stringData:
  rootCA.crt: |
$(pr -T -o 4 ${caPath}/rootCA.crt)
  edge.crt: |
$(pr -T -o 4 ${certPath}/${name}.crt)
  edge.key: |
$(pr -T -o 4 ${certPath}/${name}.key)

EOF
}

$1 $2
`)
)
