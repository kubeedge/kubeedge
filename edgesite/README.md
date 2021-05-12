
# EdgeSite: Cluster at edge

## install proxy to access kube-apiserver in other subnet.
### 1.generate certs for proxy, you need set the proxy server ip
```console
cd vendor/sigs.k8s.io/apiserver-network-proxy
make certs PROXY_SERVER_IP=<your proxy server ip>
```

### 2.copy the certs to your proxy-server and proxy-agent host. you need set the proxy server ip and proxy-agent ip.
```bash
scp -r ./certs <proxy_server_ip>:/root
scp -r ./certs <proxy_agent_ip>:/root
```

### 3.start proxy-server and proxy-agent
#### Start **proxy-server**

- as a kubernetes pod with following configuration
```bash
kubectl apply -f {your kubeedge dir}/build/edgesite/proxy-server.yaml
```

#### Start **proxy-agent**

- as a kubernetes pod with following configuration, you need set the proxy server ip and kube-apiserver ip.
```bash
cd build/edgesite
PROXY_SERVER_IP=<your proxy server ip> KUBE_APISERVER_IP=<your kube-apiserver ip>  envsubst < ./proxy-agent.yaml | kubectl apply -f -
```
