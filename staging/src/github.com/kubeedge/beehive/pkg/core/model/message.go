package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Constants for database operations and resource type settings
const (
	InsertOperation        = "insert"
	DeleteOperation        = "delete"
	QueryOperation         = "query"
	UpdateOperation        = "update"
	ResponseOperation      = "response"
	ResponseErrorOperation = "error"

	ResourceTypePod        = "pod"
	ResourceTypeConfigmap  = "configmap"
	ResourceTypeSecret     = "secret"
	ResourceTypeNode       = "node"
	ResourceTypePodlist    = "podlist"
	ResourceTypePodStatus  = "podstatus"
	ResourceTypeNodeStatus = "nodestatus"
)

// Message struct
type Message struct {
	Header  MessageHeader `json:"header"`
	Router  MessageRoute  `json:"route,omitempty"`
	Content interface{}   `json:"content"`
}

// MessageRoute contains structure of message
type MessageRoute struct {
	// where the message come from
	Source string `json:"source,omitempty"`
	// where the message will broadcast to
	Group string `json:"group,omitempty"`

	// what's the operation on resource
	Operation string `json:"operation,omitempty"`
	// what's the resource want to operate
	Resource string `json:"resource,omitempty"`
}

// MessageHeader defines message header details
type MessageHeader struct {
	// the message uuid
	ID string `json:"msg_id"`
	// the response message parentid must be same with message received
	// please use NewRespByMessage to new response message
	ParentID string `json:"parent_msg_id,omitempty"`
	// the time of creating
	Timestamp int64 `json:"timestamp"`
	// specific resource version for the message, if any.
	// it's currently backed by resource version of the k8s object saved in the Content field.
	// kubeedge leverages the concept of message resource version to achieve reliable transmission.
	ResourceVersion string `json:"resourceversion,omitempty"`
	// the flag will be set in sendsync
	Sync bool `json:"sync,omitempty"`
}

// BuildRouter sets route and resource operation in message
func (msg *Message) BuildRouter(source, group, res, opr string) *Message {
	msg.SetRoute(source, group)
	msg.SetResourceOperation(res, opr)
	return msg
}

// SetResourceOperation sets router resource and operation in message
func (msg *Message) SetResourceOperation(res, opr string) *Message {
	msg.Router.Resource = res
	msg.Router.Operation = opr
	return msg
}

// SetRoute sets router source and group in message
func (msg *Message) SetRoute(source, group string) *Message {
	msg.Router.Source = source
	msg.Router.Group = group
	return msg
}

// SetResourceVersion sets resource version in message header
func (msg *Message) SetResourceVersion(resourceVersion string) *Message {
	msg.Header.ResourceVersion = resourceVersion
	return msg
}

// IsSync : msg.Header.Sync will be set in sendsync
func (msg *Message) IsSync() bool {
	return msg.Header.Sync
}

// GetResource returns message route resource
func (msg *Message) GetResource() string {
	return msg.Router.Resource
}

// GetOperation returns message route operation string
func (msg *Message) GetOperation() string {
	return msg.Router.Operation
}

// GetSource returns message route source string
func (msg *Message) GetSource() string {
	return msg.Router.Source
}

// GetGroup returns message route group
func (msg *Message) GetGroup() string {
	return msg.Router.Group
}

// GetID returns message ID
func (msg *Message) GetID() string {
	return msg.Header.ID
}

//GetParentID returns message parent id
func (msg *Message) GetParentID() string {
	return msg.Header.ParentID
}

//GetTimestamp returns message timestamp
func (msg *Message) GetTimestamp() int64 {
	return msg.Header.Timestamp
}

//GetContent returns message content
func (msg *Message) GetContent() interface{} {
	return msg.Content
}

//GetResourceVersion returns message resource version
func (msg *Message) GetResourceVersion() string {
	return msg.Header.ResourceVersion
}

//UpdateID returns message object updating its ID
func (msg *Message) UpdateID() *Message {
	msg.Header.ID = uuid.NewV4().String()
	return msg
}

// BuildHeader builds message header. You can also use for updating message header
func (msg *Message) BuildHeader(ID, parentID string, timestamp int64) *Message {
	msg.Header.ID = ID
	msg.Header.ParentID = parentID
	msg.Header.Timestamp = timestamp
	return msg
}

//FillBody fills message  content that you want to send
func (msg *Message) FillBody(content interface{}) *Message {
	msg.Content = content
	return msg
}

// NewRawMessage returns a new raw message:
// model.NewRawMessage().BuildHeader().BuildRouter().FillBody()
func NewRawMessage() *Message {
	return &Message{}
}

// NewMessage returns a new basic message:
// model.NewMessage().BuildRouter().FillBody()
func NewMessage(parentID string) *Message {
	msg := &Message{}
	msg.Header.ID = uuid.NewV4().String()
	msg.Header.ParentID = parentID
	msg.Header.Timestamp = time.Now().UnixNano() / 1e6
	return msg
}

// Clone a message
// only update message id
func (msg *Message) Clone(message *Message) *Message {
	msgID := uuid.NewV4().String()
	return NewRawMessage().BuildHeader(msgID, message.GetParentID(), message.GetTimestamp()).
		BuildRouter(message.GetSource(), message.GetGroup(), message.GetResource(), message.GetOperation()).
		FillBody(message.GetContent())
}

// NewRespByMessage returns a new response message by a message received
func (msg *Message) NewRespByMessage(message *Message, content interface{}) *Message {
	return NewMessage(message.GetID()).SetRoute(message.GetSource(), message.GetGroup()).
		SetResourceOperation(message.GetResource(), ResponseOperation).
		FillBody(content)
}

// NewErrorMessage returns a new error message by a message received
func NewErrorMessage(message *Message, errContent string) *Message {
	return NewMessage(message.Header.ParentID).
		SetResourceOperation(message.Router.Resource, ResponseErrorOperation).
		FillBody(errContent)
}
