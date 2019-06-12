# What is KubeEdge

**KubeEdge** is an open source system extending native containerized application orchestration and device management to hosts at the Edge. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. It also supports MQTT and allows developers to author custom logic and enable resource constrained device communication at the Edge. Kubeedge consists of a cloud part and an edge part. Both edge and cloud parts are now opensourced.

## Advantages

The advantages of Kubeedge include mainly:

* **Edge Computing**

     With business logic running at the Edge, much larger volumes of data can be secured & processed locally where the data is produced. This reduces the network bandwidth requirements and consumption between Edge and Cloud. This increases responsiveness, decreases costs, and protects customers' data privacy.

* **Simplified development**

     Developers can write regular http or mqtt based applications, containerize these, and run them anywhere - either at the Edge or in the Cloud - whichever is more appropriate.

* **Kubernetes-native support**

     With KubeEdge, users can orchestrate apps, manage devices and monitor app and device status on Edge nodes just like a traditional Kubernetes cluster in the Cloud

* **Abundant applications**

     It is easy to get and deploy existing complicated machine learning, image recognition, event processing and other high level applications to the Edge.

## Components
KubeEdge is composed of these components:

- **[Edged](edge/edged.html):** an agent that runs on edge nodes and manages containerized applications.
- **[EdgeHub](edge/edgehub.html):** a web socket client responsible for interacting with Cloud Service for edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge and reporting edge-side host and device status changes to the cloud.
- **[CloudHub](cloud/cloudhub.html):** A web socket server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub. 
- **[EdgeController](cloud/controller.html):** an extended kubernetes controller which manages edge nodes and pods metadata so that the data can be targeted to a specific edge node.   
- **[EventBus](edge/eventbus.html):** an MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- **[DeviceTwin](edge/devicetwin.html):** responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- **[MetaManager](edge/metamanager.html):** the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite). 

## Architecture  

![KubeEdge Architecture](../images/kubeedge_arch.png)


## Getting involved

There are many ways to contribute to Kubeedge, and we welcome contributions!  

Read the [contributor's guide](../getting-started/contribute.html) to get started on the code.
