# KubeEdge runtime configuration

## containerd

Docker 18.09 and up ship with `containerd`, so you should not need to install it manually. If you do not have `containerd`, you may install it by running the following:

```bash
# Install containerd
apt-get update && apt-get install -y containerd.io

# Configure containerd
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml

# Restart containerd
systemctl restart containerd
```

When using `containerd` shipped with Docker, the cri plugin is disabled by default. You will need to update `containerd`’s configuration to enable KubeEdge to use `containerd` as its runtime:

```bash
# Configure containerd
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
```

Update the `edgecore` config file `edgecore.yaml`, specifying the following parameters for the `containerd`-based runtime:

```yaml
remoteRuntimeEndpoint: unix:///var/run/containerd/containerd.sock
remoteImageEndpoint: unix:///var/run/containerd/containerd.sock
runtimeRequestTimeout: 2
podSandboxImage: k8s.gcr.io/pause:3.2
runtimeType: remote
```

By default, the cgroup driver of cri is configured as `cgroupfs`. If this is not the case, you can switch to `systemd` manually in `edgecore.yaml`:

```yaml
modules:
  edged:
    cgroupDriver: systemd
```

Set `systemd_cgroup` to `true` in `containerd`’s configuration file (/etc/containerd/config.toml), and then restart `containerd`:

```toml
# /etc/containerd/config.toml
systemd_cgroup = true
```

```bash
# Restart containerd
systemctl restart containerd
```

Create the `nginx` application and check that the container is created with `containerd` on the edge side:

```bash
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
deployment.apps/nginx-deployment created

ctr --namespace=k8s.io container ls
CONTAINER                                                           IMAGE                              RUNTIME
41c1a07fe7bf7425094a9b3be285c312127961c158f30fc308fd6a3b7376eab2    docker.io/library/nginx:1.15.12    io.containerd.runtime.v1.linux
```

NOTE: since cri doesn't support multi-tenancy while `containerd` does, the namespace for containers are set to "k8s.io" by default. There is not a way to change that until [support in cri](https://github.com/containerd/cri/pull/1462) has been implemented.

## CRI-O

Follow the [CRI-O install guide](https://github.com/cri-o/cri-o/blob/master/tutorials/setup.md) to setup CRI-O.

If your edge node is running on the ARM platform and your distro is ubuntu18.04, you might need to build the binaries from source and then install, since CRI-O packages are not available in the [Kubic](https://build.opensuse.org/project/show/devel:kubic:libcontainers:stable) repository for this combination.

```bash
git clone https://github.com/cri-o/cri-o
cd cri-o
make
sudo make install
# generate and install configuration files
sudo make install.config
```

Set up CNI networking by following this guide: [setup CNI](https://github.com/cri-o/cri-o/blob/master/contrib/cni/README.md).
Update the edgecore config file, specifying the following parameters for the `CRI-O`-based runtime:

```yaml
remoteRuntimeEndpoint: unix:///var/run/crio/crio.sock
remoteImageEndpoint: unix:////var/run/crio/crio.sock
runtimeRequestTimeout: 2
podSandboxImage: k8s.gcr.io/pause:3.2
runtimeType: remote
```

By default, `CRI-O` uses `cgroupfs` as a cgroup driver manager. If you want to switch to `systemd` instead, update the `CRI-O` config file (/etc/crio/crio.conf.d/00-default.conf):

```conf
# Cgroup management implementation used for the runtime.
cgroup_manager = "systemd"
```

*NOTE: the `pause` image should be updated if you are on ARM platform and the `pause` image you are using is not a multi-arch image. To set the pause image, update the `CRI-O` config file:*

```conf
pause_image = "k8s.gcr.io/pause-arm64:3.1"
```

Remember to update `edgecore.yaml` as well for your cgroup driver manager:

```yaml
modules:
  edged:
    cgroupDriver: systemd
```

Start `CRI-O` and `edgecore` services (assume both services are taken care of by `systemd`),

```bash
sudo systemctl daemon-reload
sudo systemctl enable crio
sudo systemctl start crio
sudo systemctl start edgecore
```

Create the application and check that the container is created with `CRI-O` on the edge side:

```bash
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
deployment.apps/nginx-deployment created

# crictl ps
CONTAINER ID        IMAGE               CREATED             STATE               NAME                ATTEMPT             POD ID
41c1a07fe7bf7       f6d22dec9931b       2 days ago          Running             nginx               0                   51f727498b06f
```

## Kata Containers

Kata Containers is a container runtime created to address security challenges in the multi-tenant, untrusted cloud environment. However, multi-tenancy support is still in KubeEdge’s [backlog](https://github.com/kubeedge/kubeedge/issues/268). If you have a downstream customized KubeEdge which supports multi-tenancy already then Kata Containers is a good option for a lightweight and secure container runtime.

Follow the [install guide]( https://github.com/kata-containers/documentation/blob/master/how-to/containerd-kata.md) to install and configure containerd and  Kata Containers.

If you have “kata-runtime” installed, run this command to check if your host system can run and create a Kata Container:
```bash
kata-runtime kata-check
```

`RuntimeClass` is a feature for selecting the container runtime configuration to use to run a pod’s containers that is supported since `containerd` v1.2.0.  If your `containerd` version is later than v1.2.0, you have two choices to configure `containerd` to use Kata Containers:
- Kata Containers as a RuntimeClass
- Kata Containers as a runtime for untrusted workloads

Suppose you have configured Kata Containers as the runtime for untrusted workloads. In order to verify whether it works on your edge node, you can run:

```yaml
cat nginx-untrusted.yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-untrusted
  annotations:
    io.kubernetes.cri.untrusted-workload: "true"
spec:
  containers:
  - name: nginx
    image: nginx
```

```bash
kubectl create -f nginx-untrusted.yaml

# verify the container is running with qemu hypervisor on edge side,
ps aux | grep qemu
root      3941  3.0  1.0 2971576 174648 ?      Sl   17:38   0:02 /usr/bin/qemu-system-aarch64

crictl pods
POD ID              CREATED              STATE               NAME                NAMESPACE           ATTEMPT
b1c0911644cb9       About a minute ago   Ready               nginx-untrusted     default             0
```

## Virtlet

Make sure no libvirt is running on the worker nodes.

### Steps
1. **Install CNI plugin:**

	Download CNI plugin release and extract it:

	```bash
	$ wget https://github.com/containernetworking/plugins/releases/download/v0.8.2/cni-plugins-linux-amd64-v0.8.2.tgz

	# Extract the tarball
	$ mkdir cni
	$ tar -zxvf v0.2.0.tar.gz -C cni

	$ mkdir -p /opt/cni/bin
	$ cp ./cni/* /opt/cni/bin/
	```

	Configure CNI plugin:

	```bash
	$ mkdir -p /etc/cni/net.d/

	$ cat >/etc/cni/net.d/bridge.conf <<EOF
	{
	  "cniVersion": "0.3.1",
	  "name": "containerd-net",
	  "type": "bridge",
	  "bridge": "cni0",
	  "isGateway": true,
	  "ipMasq": true,
	  "ipam": {
	    "type": "host-local",
	    "subnet": "10.88.0.0/16",
	    "routes": [
	      { "dst": "0.0.0.0/0" }
	    ]
	  }
	}
	EOF
	```

1. **Setup VM runtime:**
 Use the script [`hack/setup-vmruntime.sh`](../../hack/setup-vmruntime.sh) to set up a VM runtime. It makes use of the Arktos Runtime release to start three containers:

	 	vmruntime_vms
		vmruntime_libvirt
		vmruntime_virtlet