# Beehive

## Beehive Overview  

Beehive is a messaging framework based on go-channels for communication between modules of KubeEdge. A module registered with beehive can communicate with other beehive modules if the name with which other beehive module is registered or the name of the group of the module is known.
Beehive supports following module operations:

1. Add Module
2. Add Module to a group
3. CleanUp (remove a module from beehive core and all groups)

Beehive supports following message operations: 

1. Send to a module/group
2. Receive by a module
3. Send Sync to a module/group
4. Send Response to a sync message

## Message Format  

Message has 3 parts 

  1. Header:  
      1. ID: message ID (string)
      2. ParentID: if it is a response to a sync message then parentID exists (string)
      3. TimeStamp: time when message was generated (int)
      4. Sync: flag to indicate if message is of type sync (bool)
  2. Route: 
      1. Source: origin of message (string)
      2. Group: the group to which the message has to be broadcasted (string)
      3. Operation: what’s the operation on the resource (string)
      4. Resource: the resource to operate on (string)
  3. Content: content of the message (interface{})
  
## Register Module  

1. On starting edge_core,  each module tries to register itself with the beehive core.
2. Beehive core maintains a map named modules which has module name as key and implementation of module interface as value. 
3. When a module tries to register itself with beehive core, beehive core checks from already loaded modules.yaml config file to check if the module is enabled. If it is enabled, it is added in the modules map or else it is added in the disabled modules map.

## Channel Context Structure Fields  

### (_Important for understanding beehive operations_)  

1. **channels:** channels is a map of string(key) which is name of module and chan(value) of message which will used to send message to the respective module.
2. **chsLock:** lock for channels map
3. **typeChannels:** typeChannels is a map of string(key)which is group name and (map of string(key) to chan(value) of message ) (value) which is map of name of each module in the group to the channels of corresponding module.
4. **typeChsLock:** lock for typeChannels map 
5. **anonChannels:** anonChannels is a map of string(parentid) to chan(value) of message which will be used for sending response for a sync message.
6. **anonChsLock:** lock for anonChannels map

## Module Operations   

### Add Module  

1. Add module operation first creates a new channel of message type.
2. Then the module name(key) and its channel(value) is added in the channels map of channel context structure. 
3. Eg: add edged module  

```
coreContext.Addmodule(“edged”)
``` 
### Add Module to Group  

1. addModuleGroup first gets the channel of a module from the channels map.
2. Then the module and its channel is added in the typeChannels map where key is the group and in the value is a map in which (key is module name and value is the channel).
3. Eg: add edged in edged group. Here 1st edged is module name and 2nd edged is the group name.  

```
coreContext.AddModuleGroup(“edged”,”edged”)
 ```
### CleanUp  

1. CleanUp deletes the module from channels map and deletes the module from all groups(typeChannels map).
2. Then the channel associated with the module is closed.
3. Eg: CleanUp edged module  

```
coreContext.CleanUp(“edged”)
```
## Message Operations  

### Send to a Module  

1. Send gets the channel of a module from channels map.
2. Then the message is put on the channel. 
3. Eg: send message to edged.  

```
coreContext.Send(“edged”,message) 
```  

### Send to a Group  

1. Send2Group gets all modules(map) from the typeChannels map.
2. Then it iterates over the map and sends the message on the channels of all modules in the map.
3. Eg: message to be sent to all modules in edged group.  

```
coreContext.Send2Group(“edged”,message) message will be sent to all modules in edged group.
```
### Receive by a Module  

1. Receive gets the channel of a module from channels map.
2. Then it waits for a message to arrive on that channel and returns the message. Error is returned if there is any.
3. Eg: receive message for edged module  

```go
msg, err := coreContext.Receive("edged")
```
### SendSync to a Module  

1. SendSync takes 3 parameters, (module, message and timeout duration)
2. SendSync first gets the channel of the module from the channels map.
3. Then the message is put on the channel.
4. Then a new channel of message is created and is added in anonChannels map where key is the messageID.
5. Then it waits for the message(response) to be received on the anonChannel it created till timeout.
6. If message is received before timeout, message is returned with nil error or else timeout error is returned.
7. Eg: send sync to edged with timeout duration 60 seconds  

```go
response, err := coreContext.SendSync("edged",message,60*time.Second)
```
### SendSync to a Group  

1. Get the list of modules from typeChannels map for the group.
2. Create a channel of message with size equal to the number of modules in that group and put in anonChannels map as value with key as messageID.
3. Send the message on channels of all the modules.
4. Wait till timeout. If the length of anonChannel = no of modules in that group, check if all the messages in the channel have parentID = messageID. If no return error else return nil error.
5. If timeout is reached,return timeout error.
6. Eg: send sync message to edged group with timeout duration 60 seconds  

```go
err := coreContext.Send2GroupSync("edged",message,60*time.Second)
```

### SendResp to a sync message  

1. SendResp is used to send response for a sync message.
2. The messageID for which response is sent needs to be in the parentID of the response message.
3. When SendResp is called, it checks if for the parentID of response message , there exists a channel is anonChannels.
4. If channel exists, message(response) is sent on that channel.
5. Or else error is logged.
```go
coreContext.SendResp(respMessage)
```
