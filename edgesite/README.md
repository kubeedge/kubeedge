
# EdgeSite: Cluster at edge

install proxy to access kube-apiserver in other subnet.

## Install proxy-server

1. generate certs for proxy and start proxy-server, you need set the proxy server ip. certs will 

   ```bash
   bash build/tools/certgen.sh proxyServer -i <proxy_server_ip1>[,proxy_server_ip2,...]; \
   kubectl apply -f build/edgesite/proxy-server.yaml
   ```

## Install proxy-agent

1. generate certs for proxy in the host installed proxy-server.  You need set the proxy server ip.

   ```bash
   bash build/tools/certgen.sh proxyAgent
   ```
   
2. copy **rootCA.crt**  file to the path **/etc/kubeedge/ca** and  copy **proxy-agent.key、proxy-agent.crt** file to the path **/etc/kubeedge/certs** of your proxy-agent host . For example,

   ```bash
   # create a folder /etc/kubeedge/ca and /etc/kubeedge/certs in advance
   scp /etc/kubeedge/ca/rootCA.crt <account>@<proxy_agent_ip>:/etc/kubeedge/ca/; \
   scp /etc/kubeedge/certs/proxy-agent.key /etc/kubeedge/certs/proxy-agent.crt <account>@<proxy_agent_ip>:/etc/kubeedge/certs/
   ```

3. start proxy-agent

   ```bash
   PROXY_SERVER_IP=<proxy_server_ip> KUBE_APISERVER_IP=<kube-apiserver_ip>  envsubst < build/edgesite/proxy-agent.yaml | kubectl apply -f -
   ```
   


