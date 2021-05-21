
# EdgeSite: Cluster at edge

install proxy to access kube-apiserver in other subnet.

## Install proxy-server

1. generate certs for proxy and start proxy-server, you need set the proxy server ip

   ```bash
   sh certgen.sh proxyServer -i <proxy server ip1>,<proxy server ip2>,...; \
   kubectl apply -f ./build/edgesite/proxy-server.yaml
   ```

## Install proxy-agent

1. generate certs for proxy in the host installed proxy-server.  You need set the proxy server ip.

   ```bash
   sh certgen.sh proxyAgent
   ```
   
2. copy **ca.crt**  file to the path **/etc/kubeedge/edgesite**  and  copy **proxy-agent.key**、**proxy-agent.crt** file to the path **/etc/kubeedge/cert** of your proxy-agent host . For example,

   ```bash
   scp ca.crt <proxy_agent_ip>:/etc/kubeedge/ca; \
   scp proxy-agent.key proxy-agent.crt <proxy_agent_ip>:/etc/kubeedge/cert
   ```

3. start proxy-agent

   ```bash
   cd vendor/sigs.k8s.io/apiserver-network-proxy
   PROXY_SERVER_IP=<your proxy server ip> KUBE_APISERVER_IP=<your kube-apiserver ip>  envsubst < ./build/edgesite/proxy-agent.yaml | kubectl apply -f -
   ```

   
