# KubeEdge runtime configuration

## containerd

Docker 18.09 ships with containerd, so you needn’t install `containerd` manually, otherwise install `containerd` in this way,

```bash
# Install containerd
apt-get update && apt-get install -y containerd.io

# Configure containerd
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml

# Restart containerd
systemctl restart containerd
```

In case of using `containerd` shipped with Docker, you still need to update containerd’s configuration, this is because “cri” plugin is disabled by default, you need to enable it so that KubeEdge can use the `containerd` as the runtime.

```bash
# Configure containerd
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
```

Update `edgecore` config file `edgecore.yaml`, specify the following parameters for `containerd` based runtime,

```yaml
remoteRuntimeEndpoint: unix:///var/run/containerd/containerd.sock
remoteImageEndpoint: unix:///var/run/containerd/containerd.sock
runtimeRequestTimeout: 2
podSandboxImage: k8s.gcr.io/pause:3.2
runtimeType: remote
```

By default, cgroup driver of CRI is configured as `cgroupfs`, if this is not the case, you can switch to `system` manually in `edgecore.yaml`,

```yaml
modules:
  edged:
    cgroupDriver: system
```

Set `systemd_cgroup` to true in containerd’s configuration file (/etc/containerd/config.toml ), restart contained service after that.

```toml
# /etc/containerd/config.toml
systemd_cgroup = true
```

```bash
# Restart containerd
systemctl restart containerd
```

Create the `nginx` application and check the container is created with `containerd` on edge side,

```bash
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
deployment.apps/nginx-deployment created

ctr --namespace=k8s.io container ls
CONTAINER                                                           IMAGE                              RUNTIME
41c1a07fe7bf7425094a9b3be285c312127961c158f30fc308fd6a3b7376eab2    docker.io/library/nginx:1.15.12    io.containerd.runtime.v1.linux
```

NOTE: since cri doesn't support multi-tenancy while containerd does, namespace for containers are set to "k8s.io" defaultly and no way to change that until the [cri's support](https://github.com/containerd/cri/pull/1462) has done.




## CRI-O

Follow the [CRI-O install guide](https://github.com/cri-o/cri-o/blob/master/tutorials/setup.md) to setup CRI-O.

If your edge node is running on ARM platform and your distro is ubuntu18.04, you might need to build the binaries form source and install after that, since CRI-O packages is not available in [Kubic](https://build.opensuse.org/project/show/devel:kubic:libcontainers:stable) repository for this combination.

```bash
git clone https://github.com/cri-o/cri-o
cd cri-o
make
sudo make install
# generate and install configuraion files
sudo make install.config
```

Setup CNI networking, please follow this guide [setup CNI](https://github.com/cri-o/cri-o/blob/master/contrib/cni/README.md) to set up the CNI networking.
Update edgecore config file, specify the following parameters for `CRI-O` based runtime,

```yaml
remoteRuntimeEndpoint: unix:///var/run/crio/crio.sock
remoteImageEndpoint: unix:////var/run/crio/crio.sock
runtimeRequestTimeout: 2
podSandboxImage: k8s.gcr.io/pause:3.2
runtimeType: remote
```

By default, `CRI-O` uses `cgroupfs` as a cgroup driver manager, update the `CRI-O` config file (/etc/crio/crio.conf.d/00-default.conf) like this if you want to switch to `system` instead.

```conf
# Cgroup management implementation used for the runtime.
cgroup_manager = "systemd"
```

*NOTE: pause image should be updated if you are on ARM platform and the `pause` image you are using is not a multi-arch image.*

Here is an example, update the `CRI-O` config file to make the pause image points to the correct image,
```conf
pause_image = "k8s.gcr.io/pause-arm64:3.1"
```

Remember to update `edgecore.yaml` as well for your cgroup driver manager,

```yaml
modules:
  edged:
    cgroupDriver: system
```

Start `CRI-O` and `edgecore` services (assume both services are taken care of by `systemd`),

```bash
sudo systemctl daemon-reload
sudo systemctl enable crio
sudo systemctl start crio
sudo systemctl start edgecore
```


Create the application and check the container is created with `CRI-O` on edge side,

```bash
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
deployment.apps/nginx-deployment created

# crictl ps
CONTAINER ID        IMAGE               CREATED             STATE               NAME                ATTEMPT             POD ID
41c1a07fe7bf7       f6d22dec9931b       2 days ago          Running             nginx               0                   51f727498b06f
```

## Kata-container

Kata-container is created to primarily address the security challenges in the multi-tenant untrusted cloud environment, multi-tenancy support is still in KubeEdge’s [backlog](https://github.com/kubeedge/kubeedge/issues/268). If you have a downstream customized KubeEdge which supports multi-tenancy already then Kata-container as a lightweight and secure container runtime is good option for you.

Follow the [install guide]( https://github.com/kata-containers/documentation/blob/master/how-to/containerd-kata.md) to install and configure containerd and  Kata Containers.

If you have “kata-runtime” installed, run this command to check if your host system can run and create a Kata Container,
```bash
kata-runtime kata-check
```

`RuntimeClass` is a feature for selecting the container runtime configuration to use to run a pod’s containers that is supported since `containerd` v1.2.0, thus, if your `containerd` is later than  v1.2.0, you will have two choices to configure containerd to use Kata Containers, “Kata Containers as  a RuntimeClass” or “Kata Containers as the runtime for untrusted workload".
Suppose you have configured Kata Containers as the runtime for untrusted workload, here is the way you can verify whether it works on your edge node,

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
