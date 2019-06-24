# Setup using Release package

## Prerequisites

+ [Install docker](https://docs.docker.com/install/)

+ [Install kubeadm/kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/)

+ [Creating cluster with kubeadm](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)

+ After initializing Kubernetes master, we need to expose insecure port 8080 for edgecontroller/kubectl to work with http connection to Kubernetes apiserver.
  Please follow below steps to enable http port in Kubernetes apiserver.

  ```shell
    vi /etc/kubernetes/manifests/kube-apiserver.yaml
    # Add the following flags in spec: containers: -command section
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
  ```

+ (**Optional**)KubeEdge also supports https connection to Kubernetes apiserver. Follow the steps in [Kubernetes Documentation](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) to create the kubeconfig file.

  Enter the path to kubeconfig file in controller.yaml
  ```yaml
  controller:
    kube:
      ...
      kubeconfig: "path_to_kubeconfig_file" #Enter path to kubeconfig file to enable https connection to k8s apiserver
  ```
  
  ## Cloud Vm
 
  **Note**:execute the below commands as root user
  ```shell
  VERSION="v0.3.0"
  OS="linux"
  ARCH="amd64"
  curl -L "https://github.com/kubeedge/kubeedge/releases/download/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz" --output kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz && tar -xf kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz  -C /etc
  
  ```
  
  ### Generate Certificates
  
  RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.
  
  ```shell
  wget -L https://github.com/kubeedge/kubeedge/blob/master/build/tools/certgen.sh
  # make script executable
  chmod +x certgen.sh
  bash -x ./certgen.sh genCertAndKey edge
  ```
  **NOTE:** The cert/key will be generated in the `/etc/kubeedge/ca` and `/etc/kubeedge/certs` respectively.
  
  + The path to the generated certificates should be updated in `etc/kubeedge/cloud/conf/controller.yaml`. Please update the correct paths for the following :
      + cloudhub.ca
      + cloudhub.cert
      + cloudhub.key
  
  + Create device model and device CRDs.
 
  ```shell
      wget -L https://github.com/kubeedge/kubeedge/blob/master/build/crds/devices/devices_v1alpha1_devicemodel.yaml
      # make script executable
      chmod +x devices_v1alpha1_devicemodel.yaml
      kubectl create -f devices_v1alpha1_devicemodel.yaml
      wget -L https://github.com/kubeedge/kubeedge/blob/master/build/crds/devices/devices_v1alpha1_device.yaml
       # make script executable
      chmod +x devices_v1alpha1_device.yaml
      kubectl create -f devices_v1alpha1_device.yaml
     ```    
  + Run cloud
  
  ```shell
      cd /etc/kubeedge/cloud
      # run edge controller
      # `conf/` should be in the same directory where edgecontroller resides
      # verify the configurations before running cloud(edgecontroller)
      ./edgecontroller
  ```
  ## Edge Vm
  ### Prerequisites
  + [Install Docker](https://docs.docker.com/install/) and/or [Containerd](https://kubernetes.io/docs/setup/cri/#containerd)
   based on the runtime to be used at edge

**NOTE:** scp kubeedge folder from cloud vm to edge vm
   
   ```shell
   In cloud
   scp -r /etc/kubeedge root@edgeip:/etc
   ```
   ### Configuring MQTT mode
   
   The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
   1) internalMqttMode: internal mqtt broker is enabled.
   2) bothMqttMode: internal as well as external broker are enabled.
   3) externalMqttMode: only external broker is enabled.
   
   Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml#L4) to select the desired mode.
   
   To use KubeEdge in double mqtt or external mode, you need to make sure that [mosquitto](https://mosquitto.org/) or [emqx edge](https://www.emqx.io/downloads/edge) is installed on the edge node as an MQTT Broker.
   
   + We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node.
   
   + Deploy node
    ```shell
         wget -L https://github.com/kubeedge/kubeedge/blob/master/build/node.json
         #Modify the node.json` file and change `metadata.name` to the name of the edge node 
         kubectl apply -f node.json
    ```
   + Modify the `/etc/kubeedge/edge/conf/edge.yaml` configuration file
       + Replace `edgehub.websocket.certfile` and `edgehub.websocket.keyfile` with your own certificate path
       + Update the IP address of the master in the `websocket.url` field. 
       + replace `fb4ebb70-2783-42b8-b3ef-63e2fd6d242e` with edge node name in edge.yaml for the below fields :
           + `websocket:URL`
           + `controller:node-id`
           + `edged:hostname-override`
       + Configure the desired container runtime in /etc/kubeedge/edge/conf/edge.yaml configuration file
       + Specify the runtime type to be used as either docker or remote (for all CRI based runtimes including containerd).
            If this parameter is not specified docker runtime will be used by default
            + `runtime-type:docker` or `runtime-type:remote`
       + Additionally specify the following parameters for remote/CRI based runtimes
            + `remote-runtime-endpoint:/var/run/containerd/containerd.sock`
            + `remote-image-endpoint:/var/run/containerd/containerd.sock`
            + `runtime-request-timeout: 2`
            + `podsandbox-image: k8s.gcr.io/pause`
            + `kubelet-root-dir: /var/run/kubelet/`
   + Run edge   
   ```shell
       # run edge_core
           # `conf/` should be in the same directory as the cloned KubeEdge repository
           cd /etc/kubeedge/edge
           # verify the configurations before running edge(edge_core)
           ./edge_core
           # or
           nohup ./edge_core > edge_core.log 2>&1 &
          
   ```
    **Note**: Running edge_core on ARM based processors,follow the above steps as mentioned for Edge Vm
   ```shell
       VERSION="v0.3.0"
       OS="linux"
       ARCH="arm"
       curl -L "https://github.com/kubeedge/kubeedge/releases/download/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz" --output kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz && tar -xf kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz  -C /etc
   ```
   + Monitoring containers status
        + If the container runtime configured to manage containers is containerd , then the following commands can be used to inspect container status and list images.
          + sudo ctr --namespace k8s.io containers ls
          + sudo ctr --namespace k8s.io images ls
          + sudo crictl exec -ti <containerid> /bin/bash