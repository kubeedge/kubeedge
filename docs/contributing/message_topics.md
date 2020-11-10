# MQTT Message Topics
KubeEdge uses MQTT for communication between deviceTwin and devices/apps.
EventBus can be started in multiple MQTT modes and acts as an interface for sending/receiving messages on relevant MQTT topics.

The purpose of this document is to describe the topics which KubeEdge uses for communication.
Please read Beehive [documentation](../components/beehive.md) for understanding about message format used by KubeEdge.

## Subscribe Topics
On starting EventBus, it subscribes to these 5 topics:
```
1. "$hw/events/node/+/membership/get"
2. "$hw/events/device/+/state/update"
3. "$hw/events/device/+/twin/+"
4. "$hw/events/upload/#"
5. "SYS/dis/upload_records"
6. "$ke/events/+/device/data/update"
```

If the the message is received on first 3 topics, the message is sent to deviceTwin, else the message is sent to cloud via edgeHub.

We will focus on the message expected on the first 3 topics.

1. `"$hw/events/node/+/membership/get"`:
This topics is used to get membership details of a node i.e the devices that are associated with the node.
The response of the message is published on `"$hw/events/node/+/membership/get/result"` topic.

2. `"$hw/events/device/+/state/update"`:
This topic is used to update the state of the device. + symbol can be replaced with ID of the device whose state is to be updated.

3. `"$hw/events/device/+/twin/+"`:
The two + symbols can be replaced by the deviceID on whose twin the operation is to be performed and any one of(update,cloud_updated,get) respectively.

4. `"$ke/events/device/+/data/update"`
This topic is add in KubeEdge v1.4, and used for delivering time-serial data. This topic is not processed by edgecore, instead, they
should be processed by third-party component on edge node such as EMQ Kuiper.

The content of data topic should conform to following format
```json
{
	"event_id": "123e4567-e89b-12d3-a456-426655440000",
	"timestamp": 1597213444,
	"data": {
		"propertyName1": {
			"value": "123",
			"metadata": {
				"timestamp": 1597213444, //+optional
				"type": "int"
			}
		},
		"propertyName2": {
			"value": "456",
			"metadata": {
				"timestamp": 1597213444,
				"type": "int"
			}
		}
	}
}
```

Following is the explanation of the three suffix used:
1. `update`: this suffix is used to update the twin for the deviceID.
2. `cloud_updated`: this suffix is used to sync the twin status between edge and cloud.
3. `get`: is used to get twin status of a device. The response is published on `"$hw/events/device/+/twin/get/result"` topic.
