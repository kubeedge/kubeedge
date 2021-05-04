#!/usr/bin/env bash

# Copyright 2020 Authors of Arktos.
# Copyright 2020 The KubeEdge Authors - file modified.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


VMS_CONTAINER_NAME=vmruntime_vms
LIBVIRT_CONTAINER_NAME=vmruntime_libvirt
VIRTLET_CONTAINER_NAME=vmruntime_virtlet

# Add more env as needed or support extra config with the optional command args
VIRTLET_LOGLEVEL=${VIRTLET_LOGLEVEL:-"4"}
VIRTLET_DISABLE_KVM=${VIRTLET_DISABLE_KVM:="y"}

usage() {
	echo "Invalid usage. Usage: "
	echo "\t$0 start | cleanup [optionl extra args]"
	exit 1
}

cleanup() {
	echo "Stop vm runtime docker containers"
	docker kill ${VMS_CONTAINER_NAME}
	docker kill ${LIBVIRT_CONTAINER_NAME}
	docker kill ${VIRTLET_CONTAINER_NAME}

	echo "Delete vm runtime meta data files"
	rm -f -r /var/lib/virtlet/
}

startRuntime() {
	echo "Create virtlet container bind host and log directories"
	mkdir -p /usr/libexec/kubernetes/kubelet-plugins/volume/exec
	mkdir -p /etc/libvirt/qemu
	mkdir -p /var/lib/libvirt
	mkdir -p /var/log/libvirt
	mkdir -p /var/lib/virtlet/vms
	mkdir -p /var/log/virtlet/vms
	mkdir -p /var/run/libvirt
	mkdir -p /var/run/netns
	mkdir -p /var/lib/virtlet/volumes

	echo "Start vm runtime containers"

	docker run --rm --net=host --privileged --pid=host --uts=host --ipc=host --user=root \
	--env VIRTLET_LOGLEVEL=${VIRTLET_LOGLEVEL} \
	--env VIRTLET_DISABLE_KVM=${VIRTLET_DISABLE_KVM} \
	--mount type=bind,src=/dev,dst=/dev \
	--mount type=bind,src=/var/lib,dst=/host-var-lib \
	--mount type=bind,src=/run,dst=/run \
	--mount type=bind,src=/usr/libexec/kubernetes/kubelet-plugins/volume/exec,dst=/kubelet-volume-plugins \
	--mount type=bind,src=/var/lib/virtlet,dst=/var/lib/virtlet,bind-propagation=rshared \
	--mount type=bind,src=/var/log,dst=/hostlog \
	arktosstaging/vmruntime:latest /bin/bash -c "/prepare-node.sh > /hostlog/virtlet/prepare-node.log 2>&1 "

	docker run --rm --net=host --privileged --pid=host --uts=host --ipc=host --user=root \
	--name ${VMS_CONTAINER_NAME} \
	--mount type=bind,src=/dev,dst=/dev \
	--mount type=bind,src=/lib/modules,dst=/lib/modules,readonly \
	--mount type=bind,src=/var/lib/libvirt,dst=/var/lib/libvirt \
	--mount type=bind,src=/var/lib/virtlet,dst=/var/lib/virtlet,bind-propagation=rshared \
	--mount type=bind,src=/var/log/virtlet,dst=/var/log/virtlet \
	--mount type=bind,src=/var/log/virtlet/vms,dst=/var/log/vms \
	arktosstaging/vmruntime:latest /bin/bash -c "/vms.sh > /var/log/virtlet/vms.log 2>&1 " &

	docker run --rm --net=host --privileged --pid=host --uts=host --ipc=host --user=root \
	--name ${LIBVIRT_CONTAINER_NAME} \
	--mount type=bind,src=/boot,dst=/boot,readonly \
	--mount type=bind,src=/dev,dst=/dev \
	--mount type=bind,src=/var/lib,dst=/var/lib \
	--mount type=bind,src=/etc/libvirt/qemu,dst=/etc/libvirt/qemu \
	--mount type=bind,src=/lib/modules,dst=/lib/modules,readonly \
	--mount type=bind,src=/run,dst=/run \
	--mount type=bind,src=/sys/fs/cgroup,dst=/sys/fs/cgroup \
	--mount type=bind,src=/var/lib/libvirt,dst=/var/lib/libvirt \
	--mount type=bind,src=/var/lib/virtlet,dst=/var/lib/virtlet,bind-propagation=rshared \
	--mount type=bind,src=/var/log/virtlet,dst=/var/log/virtlet \
	--mount type=bind,src=/var/log/libvirt,dst=/var/log/libvirt \
	--mount type=bind,src=/var/log/virtlet/vms,dst=/var/log/vms \
	--mount type=bind,src=/var/run/libvirt,dst=/var/run/libvirt \
	arktosstaging/vmruntime:latest /bin/bash -c "/libvirt.sh > /var/log/virtlet/libvirt.log 2>&1" &

	docker run --rm --net=host --privileged --pid=host --uts=host --ipc=host --user=root \
	--name ${VIRTLET_CONTAINER_NAME} \
	--env VIRTLET_LOGLEVEL=${VIRTLET_LOGLEVEL} \
        --env VIRTLET_DISABLE_KVM=${VIRTLET_DISABLE_KVM} \
	--mount type=bind,src=/etc/cni/net.d,dst=/etc/cni/net.d \
	--mount type=bind,src=/opt/cni/bin,dst=/opt/cni/bin \
	--mount type=bind,src=/boot,dst=/boot,readonly \
	--mount type=bind,src=/dev,dst=/dev \
	--mount type=bind,src=/var/lib,dst=/var/lib \
	--mount type=bind,src=/etc/libvirt/qemu,dst=/etc/libvirt/qemu \
	--mount type=bind,src=/lib/modules,dst=/lib/modules,readonly \
	--mount type=bind,src=/run,dst=/run \
	--mount type=bind,src=/sys/fs/cgroup,dst=/sys/fs/cgroup \
	--mount type=bind,src=/usr/libexec/kubernetes/kubelet-plugins/volume/exec,dst=/kubelet-volume-plugins \
	--mount type=bind,src=/var/lib/libvirt,dst=/var/lib/libvirt \
	--mount type=bind,src=/var/lib/virtlet,dst=/var/lib/virtlet,bind-propagation=rshared \
	--mount type=bind,src=/var/log,dst=/var/log \
	--mount type=bind,src=/var/log/virtlet,dst=/var/log/virtlet \
	--mount type=bind,src=/var/log/virtlet/vms,dst=/var/log/vms \
	--mount type=bind,src=/var/run/libvirt,dst=/var/run/libvirt \
	--mount type=bind,src=/var/run/netns,dst=/var/run/netns,bind-propagation=rshared \
	arktosstaging/vmruntime:latest /bin/bash -c "/start.sh > /var/log/virtlet/virtlet.log 2>&1" &
}

op=$1

# Should there be more OPs, change to switch
if [ "$op" = "start" ]; then
        shift
	startRuntime $*
	exit 0
fi

if [ "$op" = "cleanup" ]; then
        shift
        cleanup $*
        exit 0
fi

# Print usage for not supported operations
usage

exit 1
