---
title: change model message to  protobuf message 
authors:
    - "@kadisi"
    - "@rohitsardesai83"
approvers:
  - "@CindyXing"
  - "@sids-b"
  - "@qizha"
  - "kevin-wangzefeng"
creation-date: 2019-07-11
last-updated: 2019-07-24
status: alpha
---

# change model message to protobuf message 

* [change model message to protobuf message](#change-model-message-to-protobuf-message)
   * [Abstract](#abstract)
   * [Need to do](#need-to-do)
   * [How to do](#how-to-do)
   * [Change the model message to protobuf message  directly](#change-the-model-message-to-protobuf-message--directly)
   * [Solve the filed index problem of Resource](#solve-the-filed-index-problem-of-resource)
   * [Consider compatibility](#consider-compatibility)
* [How to use in KubeEdge project](#how-to-use-in-KubeEdge-project)
* [Which projects need to be modified](#which-projects-need-to-be-modified)

## Abstract 

Currently, in beehive framework, model message is defined and serialized using json format. The performance is slower than protobuf.

And for parsing the message resource, each field index is hardcoded, such as (https://github.com/kubeedge/beehive/blob/master/pkg/common/util/parse_resource.go).

In KubeEdge project, we can see many `Split(string, "/")` function which to parse the resource path string,  these are hard code. when we deal will the message resource after we receive message, we need to know the format of the assembly in advance, then we need use `Split()` function to get the field we want, it`s not easy to expand, it's not so extensible, so we need a more extended form.

Please see the #399 issue of kubeeedge 

In addition, currently when we use quic or websocket protocol to deliver message, we need to convert model message to probuf message (which defined in `github.com/kubeedge/viaduct/pkg/protos/message/message.pb.go`), and then send it, this also affects the efficiency of the transmission, rather than simply turning the model mesage into proto message.

## Need to do

*  Change the model message to protobuf message  directly
*  Solve the filed index problem of `Resource`
*  Consider compatibility  

## How to do 

##  Change the model message to protobuf message  directly

we need to create proto file in `model` dir to define message struct and use `go generate`(will move to makefile) to create message.pb.go file 

```
syntax = "proto3";
package model;
import "google/protobuf/any.proto";

message MessageHeader {
    string ID = 1;
    string ParentID = 2;
    int64 Timestamp = 3;
    bool Sync = 4;
}
message MessageRoute {
    string Source = 1;
    string Group = 2;
    string Operation = 3;
    google.protobuf.Any Resource = 4;
}

message Message {
    MessageHeader Header = 1;
    MessageRoute Router = 2;
    google.protobuf.Any Data = 3;
}
```

We use `Any type` to define `Resource` and `Data` field,  `Data` field is the original `Content` field.

##  Solve the filed index problem of `Resource`

this problem is the most important, currently, beehive define path string to stand for resource, such as `/node/{nodeID}/{namespace}/{resourceType}/{resource}`, different path stand for different resource.  but we need hardcord to parse `nodeid`,`resourceType`,`resourceid`, it's not so extensible, so we need a more extended form. we need change `Resource` to different struct type to stand for different resource path string, but in protobuf, there is not `interface` definition, so  we need to use `Any` type.

We can see `MessageRoute ` definition.

We use `github.com/gogo/protobuf`  package not offical `github.com/golang/protobuf` package, becasue many famous projects(like kubernetes, etcd, containerd) use it. 

`gogo/protobuf/types` package provides function to encodes `Any` types 
```
	func MarshalAny(pb proto.Message) (*Any, error) 
	func UnmarshalAny(any *Any, pb proto.Message) error 
```
But in gogo package, i can only support convert `proto.Message` and `Any` type to each other, it can not support convert custom type and `Any` type to each other.

In `pkg/core/typeurl` package of `github.com/kubeedge/beehive` project, we support convert custom types and Any types to each other. there are some functions:

* `func RegisterType(v interface{}, args ...string)`:

this function Register custom types, if we want to support convert string types and Any types to each other. we need to add:
 ```
 RegisterType("", "string")
 ``` 

if we want to support convert bool types and `Any` types to each other. we need to add
```
RegisterType(true, "bool")
```

if we want to convert v1.Pod types ("k8s.io/api/core/v1") and `Any` types to each other. we need to add

``` 
RegisterType(v1.Pod{}, "v1.pod")
```

the second parameter used to generate TypeUrl of `Any` type. the TypeUrl must be unique, otherwise it will panic.

* `func MarshalAny(v interface{}) (\*types.Any, error) `, this function takes interface and encodes it into google.protobuf.Any, and this interface must be Registerd by `RegisterType` function.
* `func UnmarshalAny(any \*types.Any) (interface{}, error)`, this function parses the `google.protobuf.Any` message into a interface.

## Consider compatibility  

In order to be more compatible, we try to avoid modifying methods of message. Only the following method has been modified, but I don't think it will have much impact。
```
func (msg *Message) BuildRouter(source, group, res, opr string) *Message {
func (msg *Message) BuildRouter(source, group string, res interface{}, opr string) *Message {
```

```
func (msg *Message) SetResourceOperation(res, opr string) *Message {
func (msg *Message) SetResourceOperation(res interface{}, opr string) *Message {
```

```
func (msg *Message) GetResource() string {
func (msg *Message) GetResource() interface{} {
```

let me just mention `FillBody()` function again:
```go
func (msg *Message) FillBody(content interface{}) *Message {
	msg.Content = content
	real, err := typeurl.MarshalAny(content)
	if err != nil {
		panic(errors.Errorf("Marshal content interface to any type error %v", err))
	}
	msg.Data = real
	return msg
}
```
The parameter `content interface{}` is not changed, but the parameter passed in must be registered by `RegisterType()`.
so if we want to fill string,  map or struct as message\`s Data, the map or struct must registerd by  `RegisterType()`. 

In KubeEdge, there are some places where we need to set `string` as Content of message. such as https://github.com/kubeedge/kubeedge/blob/5ca1f17b87d070ccc37ea72879354355ae65857b/edge/pkg/edgehub/controller.go#L239
so we pre-registered `string` and `bool` type in `pkg/core/typeurl` package of beehive project.

It also has registerd some frequently used types.
```go
func init() {
	registry = make(map[reflect.Type]string)
	revRegistry = make(map[string]reflect.Type)
	lock = sync.Mutex{}
	// Register sting
	RegisterType("", "string")
	// Register boolean
	RegisterType(true, "bool")
	// register slice
	RegisterType([]interface{}{}, "slice.interface")
	// register pod
	RegisterType(v1.Pod{}, "v1.pod")
}
```

# How to use in KubeEdge project

Take `DownstreamController` as an example.

In `DownstreamController`, it will send 6 kind of message. `Pod`, `CongfigMap`,`secret`,`node`,`service`,`endpoint`. They have similar resource formats(`/node/{nodeID}/{namespace}/{resourceType}/{resource}`).

So we need define a struct and method:

```go
type ToEdgeResource struct {
	ResourceType string
	NameSpace    string
	NodeID       string
	ID           string
}
func (this *ToEdgeResource) GetResourceType() string {
	return this.ResourceType
}
func (this *ToEdgeResource) GetNameSpace() string {
	return this.NameSpace
}
func (this *ToEdgeResource) GetNodeID() string {
	return this.NodeID
}
func (this *ToEdgeResource) GetID() string {
	return this.ID
}
```

those types must registerd by `RegisterType()` function, then we can convert those type into `Any` type.
```go
func init() {
	typeurl.RegisterType(ToEdgeResource{}, "ToEdgeResource")
}
```

Take `syncPod()`(cloud/pkg/controller/controller/downstream.go) function as a example:

Currently, it build Resource by `messagelayer.BuildResource()`

```go
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name)
			if err != nil {
				log.LOGGER.Warnf("built message resource failed with error: %s", err)
				continue
			}
```

We can change it to 
```go
    resource := &ToEdgeResource{
        ID:        pod.Name,
        NodeID:    pod.Spec.NodeName,
        NameSpace: pod.Namespace,
        ResourceType: ResourceTypePod,
    }
```

In cloudhub model (`cloud/pkg/cloudhub/channelq/channelq.go`), it will receive this message:
```go
func (q *ChannelEventQueue) dispatchMessage() {
	for {
		msg, err := q.ctx.Receive("cloudhub")
		if err != nil {
			log.LOGGER.Infof("receive not Message format message")
			continue
		}
		resource := msg.Router.Resource
		tokens := strings.Split(resource, "/")
		numOfTokens := len(tokens)
```

This main function is to get the hostid, and put the message on the hostid\`s channel. but it use `strings.Split()` function to get hostid

We can change to:
```go
    res := msg.GetResource()

	teRes, ok := res.(resources.ToEdgeResource)
	if !ok || teRes.GetNodeID() == "" {
	    continue
	}
	nodeID := teRes.GetNodeID()

```

and finnally, we convert Message to Event:
```go
event := model.MessageToEvent(&msg)
		select {
		case rChannel <- event:
		}
```

we need change Resource field type  of UserGroupInfo from
```go
type UserGroupInfo struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}
```
to
```go
type UserGroupInfo struct {
	Resource  interface `json:"resource"`
	Operation string `json:"operation"`
}
```

and the next step，cloud will send this message from quic or websocket.

For quic and websocket, they all call `translator.NewTran().Encode()` function to convert model message into protobuf message.
If we define the model message directly as proto message, this function will be useless, and it will be more efficient and the code will become simpler.


# Which projects need to be modified

* `github.com/kubeedge/kubeedge`
    
    Mainly process `Resource` problem
* `github.com/kubeedge/beehive`
    
    Define message proto file, and define `typeurl` package to support convert custom type into `Any` type.
* `github.com/kubeedge/viaduct`
    
    No need `translator.NewTran().Encode()`
