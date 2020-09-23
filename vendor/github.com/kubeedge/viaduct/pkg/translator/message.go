package translator

import (
	"encoding/json"
	"fmt"

	"k8s.io/klog"

	"github.com/golang/protobuf/proto"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/protos/message"
)

type MessageTranslator struct {
}

func NewTran() *MessageTranslator {
	return &MessageTranslator{}
}

func (t *MessageTranslator) protoToModel(src *message.Message, dst *model.Message) error {
	dst.BuildHeader(src.Header.ID, src.Header.ParentID, int64(src.Header.Timestamp)).
		BuildRouter(src.Router.Source, src.Router.Group, src.Router.Resouce, src.Router.Operaion).
		FillBody(src.Content)

	// TODO:
	dst.Header.Sync = src.Header.Sync

	return nil
}

func (t *MessageTranslator) modelToProto(src *model.Message, dst *message.Message) error {
	dst.Header.ID = src.GetID()
	dst.Header.ParentID = src.GetParentID()
	dst.Header.Timestamp = int64(src.GetTimestamp())
	dst.Header.Sync = src.IsSync()
	dst.Router.Source = src.GetSource()
	dst.Router.Group = src.GetGroup()
	dst.Router.Resouce = src.GetResource()
	dst.Router.Operaion = src.GetOperation()
	if content := src.GetContent(); content != nil {
		switch content.(type) {
		case []byte:
			dst.Content = content.([]byte)
		case string:
			dst.Content = []byte(content.(string))
		default:
			bytes, err := json.Marshal(content)
			if err != nil {
				klog.Error("failed to marshal")
				return err
			}
			dst.Content = bytes
		}
	}
	return nil
}

func (t *MessageTranslator) Decode(raw []byte, msg interface{}) error {
	modelMessage, ok := msg.(*model.Message)
	if !ok {
		return fmt.Errorf("bad msg type")
	}

	protoMessage := message.Message{}
	err := proto.Unmarshal(raw, &protoMessage)
	if err != nil {
		klog.Error("failed to unmarshal payload")
		return err
	}
	t.protoToModel(&protoMessage, modelMessage)
	return nil
}

func (t *MessageTranslator) Encode(msg interface{}) ([]byte, error) {
	modelMessage, ok := msg.(*model.Message)
	if !ok {
		return nil, fmt.Errorf("bad msg type")
	}

	protoMessage := message.Message{
		Header: &message.MessageHeader{},
		Router: &message.MessageRouter{},
	}

	err := t.modelToProto(modelMessage, &protoMessage)
	if err != nil {
		klog.Error("failed to copy message")
		return nil, err
	}

	msgBytes, err := proto.Marshal(&protoMessage)
	if err != nil {
		klog.Error("failed to marshal message")
		return nil, err
	}

	return msgBytes, nil
}
