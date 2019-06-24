# CloudHub

## CloudHub Overview

CloudHub is the mediator between EdgeController and the Edge side. It supports both web-socket based connection as well as a [QUIC](https://quicwg.org/ops-drafts/draft-ietf-quic-applicability.html) protocol access at the same time.
The edgehub can choose one of the protocols to access to the cloudhub. CloudHub's function is to enable the communication between edge and the EdgeController.

The connection to the edge(through EdgeHub module) is done through the HTTP over websocket connection.
For internal communication it directly communicates with the EdgeController.
All the request send to CloudHub are of context object which are stored in channelQ along with the
mapped channels of event object marked to its nodeID.


The main functions performed by CloudHub are :-

- Get message context and create ChannelQ for events
- Create http connection over websocket
- Serve websocket connection
- Read message from edge
- Write message to edge
- Publish message to Controller


### Get message context and create ChannelQ for events:

The context object is stored in a channelQ.
For all nodeID channel is created and the message is converted to event object
Event object is then passed through the channel.

### Create http connection over websocket:

- TLS certificates are loaded through the path provided in the context object
- HTTP server is started with TLS configurations
- Then HTTP connection is upgraded to websocket connection receiving conn object.
- ServeConn function the serves all the incoming connections

### Read message from edge:

- First a deadline is set for keepalive interval
- Then the JSON message from connection is read
- After that Message Router details are set
- Message is then converted to event object for cloud internal communication
- In the end the event is published to EdgeController

### Write Message to Edge:

- First all event objects are received for the given nodeID
- The existence of same request and the liveness of the node is checked
- The event object is converted to message structure
- Write deadline is set. Then the message is passed to the websocket connection

### Publish Message to EdgeController:

- A default message with timestamp, clientID and event type is sent to controller
    every time a request is made to websocket connection
- If the node gets disconnected then error is thrown and an event describing
    node failure is published to the controller.

## Usage

The CloudHub can be configured in three ways as mentioned below :

- **Start the websocket server only**: Click [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/quic-design.md#start-the-websocket-server-only) to see the details.
- **Start the quic server only**: Click [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/quic-design.md#start-the-quic-server-only) to see the details.
- **Start the websocket and quic server at the same time**: Click [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/quic-design.md#start-the-quic-server-only) to see the details