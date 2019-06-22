# Edgemesh test env config guide

## install Containerd 
+ please refer to the following link to install  
> <https://kubernetes.io/docs/setup/production-environment/container-runtimes/#containerd>

## install CNI plugin 
+ download cni plugin 
```bash
$ wget https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
```
+ decompression and installation  
```bash
$ mkdir -p /opt/cni/bin
$ tar -zxvf cni-plugins-linux-amd64-v0.8.1.tgz -C /opt/cni/bin
```
+ configure cni plugin
 
```bash
$ mkdir -p /etc/cni/net.d/
```
   > * please Make sure docker0 does not exist !!
   > * field "bridge" must be "docker0"
   > * field "isGateway" must be true 
   
```bash
$ cat >/etc/cni/net.d/10-mynet.conf <<EOF
{
	"cniVersion": "0.2.0",
	"name": "mynet",
	"type": "bridge",
	"bridge": "docker0",
	"isGateway": true,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
		"subnet": "10.22.0.0/16",
		"routes": [
			{ "dst": "0.0.0.0/0" }
		]
	}
}
EOF
```

## Configure port mapping manually on node on which server is running
> can see the examples in the next section. 
* step1. execute iptables command as follows 
```bash
$ iptables -t nat -N PORT-MAP
$ iptables -t nat -A PORT-MAP -i docker0 -j RETURN
$ iptables -t nat -A PREROUTING -p tcp -m addrtype --dst-type LOCAL -j PORT-MAP
$ iptables -t nat -A OUTPUT ! -d 127.0.0.0/8 -p tcp -m addrtype --dst-type LOCAL -j PORT-MAP
$ iptables -P FORWARD ACCEPT
```
* step2. execute iptables command as follows 
   > * **portIN** is the service map at the host
   > * **containerIP** is the IP in the container. Can be find out on master by **kubectl get pod -o wide**
   > * **portOUT** is the port that monitored In-container 
```bash
$ iptables -t nat -A PORT-MAP ! -i docker0 -p tcp -m tcp --dport portIN -j DNAT --to-destination containerIP:portOUT
``` 

## Example for Edgemesh test env
![edgemesh test env example](../images/edgemesh/edgemesh-test-env-example.png)