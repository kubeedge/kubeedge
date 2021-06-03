
# EdgeSite: Cluster at edge

install edgesite-server and edgesite-agent to access kube-apiserver in other subnet.

## Install edgesite-server

1. on edgesite-server host, generate certs for edgesite-server and start edgesite-server, you need to set the edgesite server ip.

   ```bash
   bash build/tools/certgen.sh edgesiteServer -i <edgesite_server_ip1>[,edgesite_server_ip2,...]; \
   kubectl apply -f build/edgesite/edgesite-server.yaml
   ```

## Install edgesite-agent

1. generate certs for edgesite-agent on the host installed edgesite-server.

   ```bash
   bash build/tools/certgen.sh edgesiteAgent
   ```
   
2. copy **rootCA.crt** file and **edgesite-agent.key, edgesite-agent.crt** files generated in step 1 to your edgesite-agent host. 
Make sure that the /etc/kubeedge/ca/ and /etc/kubeedge/certs directories exist on the edgesite-agent host. For example,

   ```bash
   # precondition: create /etc/kubeedge/ca directory and /etc/kubeedge/certs directory on the host which will install edgesite-agent. 
   scp /etc/kubeedge/ca/rootCA.crt <account>@<edgesite_agent_ip>:/etc/kubeedge/ca/; \
   scp /etc/kubeedge/certs/edgesite-agent.key /etc/kubeedge/certs/edgesite-agent.crt <account>@<edgesite_agent_ip>:/etc/kubeedge/certs/
   ```

3. start edgesite-agent on edgesite-agent host.

   ```bash
   EDGESITE_SERVER_IP=<edgesite_server_ip> KUBE_APISERVER_IP=<kube-apiserver_ip>  envsubst < build/edgesite/edgesite-agent.yaml | kubectl apply -f -
   ```
   


