
# EdgeSite: Cluster at edge

install proxy to access kube-apiserver in other subnet.

## Install proxy-server

1. on proxy-server host, generate certs for proxy-server and start proxy-server, you need to set the proxy server ip.

   ```bash
   bash build/tools/certgen.sh proxyServer -i <proxy_server_ip1>[,proxy_server_ip2,...]; \
   kubectl apply -f build/edgesite/proxy-server.yaml
   ```

## Install proxy-agent

1. generate certs for proxy-agent on the host installed proxy-server.

   ```bash
   bash build/tools/certgen.sh proxyAgent
   ```
   
2. copy **rootCA.crt** file and **proxy-agent.key、proxy-agent.crt** files generated in step 1 to your proxy-agent host. 
Make sure that the /etc/kubeedge/ca/ and /etc/kubeedge/certs directories exist on the proxy-agent host. For example,

   ```bash
   # precondition: create /etc/kubeedge/ca directory and /etc/kubeedge/certs directory on the host which will install proxy-agent. 
   scp /etc/kubeedge/ca/rootCA.crt <account>@<proxy_agent_ip>:/etc/kubeedge/ca/; \
   scp /etc/kubeedge/certs/proxy-agent.key /etc/kubeedge/certs/proxy-agent.crt <account>@<proxy_agent_ip>:/etc/kubeedge/certs/
   ```

3. start proxy-agent on proxy-agent host.

   ```bash
   PROXY_SERVER_IP=<proxy_server_ip> KUBE_APISERVER_IP=<kube-apiserver_ip>  envsubst < build/edgesite/proxy-agent.yaml | kubectl apply -f -
   ```
   


