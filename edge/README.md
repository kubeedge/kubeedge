This section contains the source code for KubeEdge edge side components

## KubeEdge Edge

At the edge side, there are six major components.

- EdgeHub: a web socket client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- Edged: an agent that runs on edge nodes and manages containerized applications.
- EventBus: a MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- ServiceBus: a HTTP client to interact with HTTP servers (REST), offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge.
- DeviceTwin: responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- MetaManager: the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite).