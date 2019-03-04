# What is KubeEdge

**KubeEdge** is an open source system extending native containerized application orchestration and device management to hosts at the Edge. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. It also supports MQTT and allows developers to author custom logic and enable resource constrained device communication at the Edge. Kubeedge consists of a cloud part and an edge part. The edge part has already been open sourced and the cloud part is coming soon!  

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

- **Edged:** Edged is an agent running on edge node for managing user's application.
- **[EdgeHub](modules/edgehub.html):** EdgeHub is a web socket client, which is responsible for interacting with **Huawei Cloud IEF service**, including sync cloud side resources update, report edged side host and device status changes.
- **[EventBus](modules/eventbus.html):** EventBus is a MQTT client to interact with MQTT server(mosquitto), offer subscribe and publish capability to other components.
- **[DeviceTwin](modules/devicetwin.html):** DeviceTwin is responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- **[MetaManager](modules/metamanager.html):** MetaManager is the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database(SQLite). 

## Architecture

![KubeEdge Architecture](images/kubeedge_arch.png)


## Getting involved

There are many ways to contribute to Kubeedge, and we welcome contributions!  
Read the [contributor's guide](./contribute.html) to get started on the code.
