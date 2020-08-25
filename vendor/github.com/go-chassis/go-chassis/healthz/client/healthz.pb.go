package client

//Reply The response message containing the greetings
type Reply struct {
	AppID       string `protobuf:"bytes,1,opt,name=appId" json:"appId,omitempty"`
	ServiceName string `protobuf:"bytes,2,opt,name=serviceName" json:"serviceName,omitempty"`
	Version     string `protobuf:"bytes,3,opt,name=version" json:"version,omitempty"`
}
