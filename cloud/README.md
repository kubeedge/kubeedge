This section contains the source code for KubeEdge cloud side components

## KubeEdge Cloud

At the cloud side, there are two major components: EdgeController and CloudHub. 
EdgeController is an extended Kubernetes controller. It watches nodes and pods against APIServer for the cluster.
Upon changes in nodes/pods, KubeEdge will convert the pod/node binding info. in the format of node -- pods. 
This way, an edge node can obtain pods targeted for itself. It enhances efficiency and reduces the network bandwidth requirement between cloud & edge. 
