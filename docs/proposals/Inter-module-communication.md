# Inter module communication

## Motivation

In a weak or no network environment

- When the processing capacity is slow, the downstream websocket channel will be blocked
- When there are too many messages, the upstream edgehub channel will be blocked
- When there is too much channel data, the sent message will have to wait a long time to be processed, and of course it may be discarded

### Goals

- Edgecore can respond to cloud requests instantly
- Messages are cached in the cloud, not at the edge
- The edge will not be blocked by messages sent from the cloud

## Proposal

Currently `KubeEdge` uses beehive communication, the channel cache is not handled properly, it will cause the above-mentioned series of problems.

Therefore, it is recommended to remove the beehive communication module from `KubeEdge` and change it to a method of autonomous calling between modules.

- When the cloud message is sent, it is directly processed by the corresponding module
- Increase the coroutine pool in the message processing part, increase concurrent processing, and ensure the amount of concurrency
- Edgehub adds pending message check. If there are too many, the cloud will be notified to suspend message delivery