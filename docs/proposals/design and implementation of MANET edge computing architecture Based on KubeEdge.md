## Motivation

  Based on KubeEdge, the most popular cloud-native cloud edge collaborative computing architecture in the industry, an edge computing architecture for typical MANET scenarios such as UAV/unmanned system clusters is designed and implemented to better support "cloud-edge" and "edge-edge" collaborative computing capabilities in MANET.



## Goals

1. Deploy the edge component of KubeEdge on nodes in the MANET Emulation network or physical MANET network, and deploy the cloud component of KubeEdge outside the MANET gateway node. To realize the "cloud-edge" collaborative computing between any edge node and cloud node;
2. Analyzed the "edge-edge" coordination requirements for typical MANET application requirements, designed corresponding technical solutions, and extended the edge and side components of KubeEdge;
3. Design and implement DEMO.



## Proposal

1. Implement a dynamic hierarchy determination system, so that cloud nodes can know the hierarchy of all edge nodes in real time. Supports the addition of new nodes and the exit of existing nodes, dynamically adjusting the topology.
2. Based on MANET, extend the side components of KubeEdge to achieve task uninstallation.



## Design Details

### Architecture and Modules

<img src="../images/design and implementation of MANET edge computing architecture Based on KubeEdge/image-3.png" alt="alt text"/>



### 1.Cloud-Edge Collaboration:

#### ① Initialization:

- **Cloud node initialization:** the cloud node is the root node, and its level is set to 0; It needs to periodically broadcast its own hierarchical information to all edge nodes.
- **Edge node initialization:** The initial level is set to infinity, indicating that the level has not been determined.

#### ② Broadcast and update process:

Each edge node receives the information and updates it according to the following rules, and broadcasts its own hierarchical information to all other nodes:

- If the received level + 1 is less than the current node's level, update the current node's level to the received level + 1.

- After the level is updated, the current node broadcasts the update to all other neighbor nodes to encourage other nodes in the network to update as well.

  

#### ③ Node added\deleted:

​	New node registration: When a new node is added to the network, the cloud node receives the registration information of the new node and re-broadcasts to determine the hierarchy of all nodes. (Same with node deletion)



### 2.Edge-Edge Collaboration:

Task unloading mechanism: 

​	According to the level of each node, intelligent task unloading can be carried out, so that edge nodes can dynamically assign and adjust tasks according to their own and other node levels, and optimize task execution. For example, if a task is uninstalled to a node at a lower level than the cloud node, the task is uninstalled to the root node.



### Data structure and interface design

#### 1.The cloud node needs to maintain the following fields:

```
• Node ID: unique identifier.
• Level: Node level, which indicates the depth of the node in the network.
• Parent ID: Identifies the parent node, which is used to trace the data path.
• Children List: specifies the next-level node list, which is used for data delivery and hierarchical update.
• Timestamp: The timestamp of the last hierarchical update, ensuring that the information is up to date.
 (Some fields are optional)
```

#### 2.Edge node:

```
• Design and implement a data structure on each node to store information about the hierarchy and neighboring nodes.
• Includes the node ID, level value, list of neighboring nodes and their level value.
```

#### 3.interface design：

```
• Define and implement internal interfaces to support querying, updating, and broadcasting information in a mobile AD hoc network.
```



### Implementation Step

#### 1.Cloud-Edge Collaboration:

##### ①CloudCore:

​	Add hierarchical information broadcast and new node processing logic in cloudhub:

```go
// cloudhub.go

// A function for broadcasting level information
func broadcastLevelInfo() {
    for {
        // Send hierarchy information to all edge nodes, assuming that there is a function called SendLevelInfoToAllEdges that does this
        handler.SendLevelInfoToAllEdges(0)
        time.Sleep(10 * time.Minute)
    }
}

// The new node is added
func handleNewNodeJoin(nodeID string) {
    // Re-broadcast the hierarchy information to determine the hierarchy of the new node
    handler.SendLevelInfoToAllEdges(0)
}

func StartCloudHub() {
    go broadcastLevelInfo()

    // Listen for new node joining messages
    go func() {
        for {
            nodeID := handler.ReceiveNewNodeJoin()
            handleNewNodeJoin(nodeID)
        }
    }()

    // Other startup logic...
    processor.NewProcessor(config.Config).Start()
}

```

​	The handler module implements the sending of hierarchical information and the processing of new node joining:

```go
// handler.go (CloudCore)

// Broadcast level information to all edge nodes
Broadcast level information to all edge nodes
Broadcast level information to all edge nodes
func SendLevelInfoToAllEdges(level int) {
    msg := message.NewMessage("cloudhub/broadcast", modules.MetaGroup, level)
    // Implement message sending logic, broadcast to all edge nodes
    SendMessageToAllEdges(msg)
}

// The message that a new node was added was received
func ReceiveNewNodeJoin() string {
    // Implement message receiving logic, assuming messages are received from a certain channel
    msg := ReceiveMessage()
    if msg.Group == modules.MetaGroup && msg.GetResource() == "cloudhub/newnode" {
        nodeID := msg.GetContent().(string)
        return nodeID
    }
    return ""
}

```



##### ②EdgeCore

​	Add logic to receive and process hierarchical information in edgehub:

```go
// edgehub.go

// Initialized to the maximum int value, representing infinity
var currentNodeLevel int = int(^uint(0) >> 1) 

// A function that receives hierarchical information and updates it
func receiveAndProcessLevelInfo(receivedLevel int) {
    newLevel := receivedLevel + 1
    if newLevel < currentNodeLevel {
        currentNodeLevel = newLevel
        fmt.Printf("Updated node level to %d\n", currentNodeLevel)
        broadcastLevelToNeighbors() // The update is broadcast to neighbors
    }
}

// A function that broadcasts the current level information to the neighbor node
func broadcastLevelToNeighbors() {
    // The current tier is broadcast to the neighbor node
    handler.BroadcastLevel(currentNodeLevel)
}

func StartEdgeHub() {
    go func() {
        for {
            // Realize the function of receiving hierarchical information
            receivedLevel := handler.ReceiveLevelInfo()
            receiveAndProcessLevelInfo(receivedLevel)
        }
    }()

    // Other startup logic...
    config.InitConfigure()
    handler.InitHandler(config.Config)
}

```



​	Implement the receiving and broadcasting of hierarchical information in the handler module.

```go
// handler.go (EdgeCore)

// Receive level information
func ReceiveLevelInfo() int {
    msg := ReceiveMessage()
    if msg.Group == modules.MetaGroup && msg.GetResource() == "cloudhub/broadcast" {
        level := msg.GetContent().(int)
        return level
    }
    return -1 // 
}

func BroadcastLevel(level int) {
    msg := message.NewMessage("edgehub/broadcast", modules.MetaGroup, level)
    SendMessageToNeighbors(msg)
}
```



#### 2.Edge-Edge Collaboration:

​	Implementation of task unloading mechanism，Add task unload logic to the handler module：

```go
// handler.go (EdgeCore)

// Task unload function
func OffloadTask(task string, targetLevel int) {
    // Select the appropriate node for task uninstallation according to the target level
    targetNode := selectTargetNode(targetLevel)
    msg := message.NewMessage("edgehub/task", modules.MetaGroup, task)
    SendMessageToNode(targetNode, msg)
}

// Select the function of the target node (according to the hierarchy)
func selectTargetNode(targetLevel int) string {
    // Implement the logic of selecting the target node and return the node ID
    nodes := GetNodesByLevel(targetLevel)
    if len(nodes) > 0 {
        return nodes[0] 
    }
    return ""
}
```







## Example of implementation scheme

<img src="../images/design and implementation of MANET edge computing architecture Based on KubeEdge/image-1.png" alt="alt text"/>

Suppose that the topology of the current MANET is as shown in the figure above.
The implementation steps are as follows:
① Initial state:
• Node Cloud: Level = 0 (assuming root node)
•Node A,B, C, D: Level = ∞
② First round broadcast and update
• Cloud broadcast Level 0 to A.
When A receives the information, it updates its Level to 1, sets the Cloud to the Parent ID, and then broadcasts the update. (Repeat this step on subsequent nodes.)
③ Final state
•	Node Cloud: Level = 0
•	Node A: Level = 0
•	Node B: Level = 2
•	Node C: Level = 2
•	Node D: Level = 3



<img src="../images/design and implementation of MANET edge computing architecture Based on KubeEdge/image-1.png" alt="alt text"/>

​    According to the above state, the form maintained by the cloud node and the information stored by each edge node form a tree structure as described in the figure above.
​    In the aspect of edge collaboration: each edge node unloads tasks to the cloud node in a unified manner.



## Roadmap

Preparatory phase (7.1-7.31)

- Review and submit proposals.
- Build and deploy KubeEdge service, familiar with KubeEdge source code.
- Familiar with  MANET knowledge

 Development phase(8.1-8.31)

- Based on MANET, develop and modify KubeEdge code to realize multi-level architecture design.

- Compare different implementation schemes and select the best method.

Testing and summary phase(9.1-9.30)

- Test and optimize the code, and make the necessary comments.

- Write project documentation, presentation materials, and submit code.
- Design and implement DEMO.



