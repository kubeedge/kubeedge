#!/bin/bash

echo 'create edgemesh init Container image'

function usage() {
	echo "execute 'sh createImg.sh [rpm | deb]' to create docker image"
	echo "execute 'sh createImg.sh help for use help'"
}

path="${1}"

if [ "${path}" != "rpm" ] && [ "${path}" != "deb" ]; then
	usage
	exit 0
fi

echo "create a ${path} docker image"

cp ./script/edgemesh-iptables.sh ./"${path}"/

cd ./"${path}"/

chmod 0777 edgemesh-iptables.sh

if command -v docker > /dev/null 2>&1 ; then
	#docker build
	docker build -t edgemesh_init .
	# delete iptables script
	rm ./edgemesh-iptables.sh 
else
	echo 'the docker command is no found!!'
	exit 1
fi
