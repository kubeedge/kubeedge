# MQTT Message Topics
KubeEdge uses MQTT for communication between deviceTwin and devices/apps.
EventBus can be started in multiple MQTT modes and acts as an interface for sending/receiving messages on relevant MQTT topics.

The purpose of this document is to describe the topics which KubeEdge uses for communication.
Please read Beehive [documentation](../modules/beehive.html) for understanding about message format used by KubeEdge. 

## Subscribe Topics
On starting EventBus, it subscribes to these 5 topics:
```
1. "$hw/events/node/+/membership/get"
2. "$hw/events/device/+/state/update"
3. "$hw/events/device/+/twin/+"
4. "$hw/events/upload/#"
5. "SYS/dis/upload_records"
```  

If the the message is received on first 3 topics, the message is sent to deviceTwin, else the message is sent to cloud via edgeHub.

We will focus on the message expected on the first 3 topics.

1. `"$hw/events/node/+/membership/get"`:
This topics is used to get membership details of a node i.e the devices that are associated with the node.
The response of the message is published on `"$hw/events/node/+/membership/get/result"` topic.  

2. `"$hw/events/device/+/state/update`":
This topic is used to update the state of the device. + symbol can be replaced with ID of the device whose state is to be updated.  

3. `"$hw/events/device/+/twin/+"`:
The two + symbols can be replaced by the deviceID on whose twin the operation is to be performed and any one of(update,cloud_updated,get) respectively.  

Following is the explanation of the three suffix used:  
1. `update`: this suffix is used to update the twin for the deviceID.  
2. `cloud_updated`: this suffix is used to sync the twin status between edge and cloud.  
3. `get`: is used to get twin status of a device. The response is published on `"$hw/events/device/+/twin/get/result"` topic.