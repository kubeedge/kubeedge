#!/bin/bash

echo 'create edgemesh init Container image'

function usage() {
	echo "sh createImg.sh [rpm | deb]"
}

path="${1}"

if [ "${path}" != "rpm" ] || [ "${path}" != "deb" ]; then
	usage
fi

cp ./script/edgemesh-iptables.sh ./"${path}"/

cd ./"${path}"/

if command -v docker > /dev/null 2>&1 ; then
	#docker build
	docker build -t edgemesh_init .
	# delete iptables script
	rm ./"${path}"/edgemesh-iptables.sh 
else
	exit 1
fi

