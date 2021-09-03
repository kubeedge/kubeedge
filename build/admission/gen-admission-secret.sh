#!/bin/bash

set -e

SERVICE=${SERVICE:-"kubeedge-admission-service"}
SECRET=${SECRET:-"kubeedge-admission-secret"}
NAMESPACE=${NAMESPACE:-kubeedge}
CERTDIR=${CERTDIR:-"/etc/kubeedge/admission/certs"}
ENABLE_CREATE_SECRET=${ENABLE_CREATE_SECRET:-true}

if [[ ! -x "$(command -v openssl)" ]]; then
    echo "openssl not found"
    exit 1
fi

function createCerts() {
  echo "creating certs in dir ${CERTDIR} "

  cat <<EOF > ${CERTDIR}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
EOF

  openssl genrsa -out ${CERTDIR}/ca.key 2048
  openssl req -x509 -new -days 3650 -nodes -key ${CERTDIR}/ca.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ${CERTDIR}/ca.crt

  openssl genrsa -out ${CERTDIR}/server.key 2048
  openssl req -new -days 3650 -key ${CERTDIR}/server.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -out ${CERTDIR}/server.csr -config ${CERTDIR}/csr.conf

  openssl x509 -req -days 3650 -in  ${CERTDIR}/server.csr -CA  ${CERTDIR}/ca.crt -CAkey  ${CERTDIR}/ca.key \
  -CAcreateserial -out  ${CERTDIR}/server.crt \
  -extensions v3_req -extfile  ${CERTDIR}/csr.conf
}

function createObjects() {
  # `ENABLE_CREATE_SECRET` should always be set to `true` unless it has been already created.
  if [[ "${ENABLE_CREATE_SECRET}" = true ]]; then
      kubectl get ns ${NAMESPACE} || kubectl create ns ${NAMESPACE}

      # create the secret with CA cert and server cert/key
      kubectl create secret generic ${SECRET} \
          --from-file=tls.key=${CERTDIR}/server.key \
          --from-file=tls.crt=${CERTDIR}/server.crt \
          --from-file=ca.crt=${CERTDIR}/ca.crt \
          -n ${NAMESPACE}
  fi
}

function checkCertDir() {
  if [[ -d ${CERTDIR} ]]; then
    echo -n -e "certs dir already exits, do you want to overwrite the certs and generate them againi? [y/N]> "
    read -r OVERWRITE
    if [[ "${OVERWRITE}" =~ ^[nN]$ ]]; then
      echo "certs is not generated, please remove the certs directory if you want to generate them again."
      exit 0
    elif [[ "${OVERWRITE}" =~ ^[yY]$ ]]; then
      createCerts
    else
      echo -e "Invalid response, please try again."
      checkCertDir
    fi
  else
    mkdir -p ${CERTDIR}
    createCerts
  fi
}

checkCertDir
createObjects
